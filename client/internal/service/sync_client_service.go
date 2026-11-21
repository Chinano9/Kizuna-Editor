/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kizuna/client/internal/audio"
	"kizuna/client/internal/crypto"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// EventEmitter defines an abstraction to send events to the frontend (Wails or test mocks).
type EventEmitter func(ctx context.Context, eventName string, data ...interface{})

// defaultEventEmitter emits events via Wails runtime only when the context contains the Wails frontend/events engine.
func defaultEventEmitter(ctx context.Context, eventName string, data ...interface{}) {
	if ctx == nil {
		return
	}
	// Check if this context has Wails events runtime injected (prevents log.Fatalf in non-Wails contexts / tests)
	if ctx.Value("events") == nil {
		return
	}
	wailsRuntime.EventsEmit(ctx, eventName, data...)
}

// progressReader counts transferred bytes and emits progress payloads.
type progressReader struct {
	reader        io.Reader
	ctx           context.Context
	operation     string
	totalBytes    int64
	transferred   int64
	attempt       int
	maxAttempts   int
	emitter       EventEmitter
	mu            sync.Mutex
	lastEmitTime  time.Time
	throttleDelay time.Duration
}

func newProgressReader(
	ctx context.Context,
	reader io.Reader,
	operation string,
	totalBytes int64,
	attempt int,
	maxAttempts int,
	emitter EventEmitter,
) *progressReader {
	return &progressReader{
		reader:        reader,
		ctx:           ctx,
		operation:     operation,
		totalBytes:    totalBytes,
		attempt:       attempt,
		maxAttempts:   maxAttempts,
		emitter:       emitter,
		throttleDelay: 50 * time.Millisecond,
	}
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.mu.Lock()
		pr.transferred += int64(n)
		current := pr.transferred
		total := pr.totalBytes
		now := time.Now()
		shouldEmit := now.Sub(pr.lastEmitTime) >= pr.throttleDelay || (total > 0 && current >= total) || err == io.EOF
		if shouldEmit {
			pr.lastEmitTime = now
		}
		pr.mu.Unlock()

		if shouldEmit {
			var percentage float64
			if total > 0 {
				percentage = float64(current) / float64(total) * 100.0
				if percentage > 100.0 {
					percentage = 100.0
				}
			}
			phase := "transferring"
			if total > 0 && current >= total {
				phase = "completed"
			}
			pr.emitProgress(phase, current, total, percentage)
		}
	}
	return n, err
}

func (pr *progressReader) emitProgress(phase string, current, total int64, percentage float64) {
	if pr.emitter == nil || pr.ctx == nil {
		return
	}
	payload := models.SyncProgressPayload{
		Operation:        pr.operation,
		BytesTransferred: current,
		TotalBytes:       total,
		Percentage:       percentage,
		Phase:            phase,
		Attempt:          pr.attempt,
		MaxAttempts:      pr.maxAttempts,
	}
	pr.emitter(pr.ctx, "sync:progress", payload)
}

const MaskedCloudTitle = "[E2EE] Encrypted Project"

// SyncClientService coordinates local project packaging, E2EE encryption/decryption, cloud synchronization, and resilient retries (Issues #13, #14, #16, #22, #23).
type SyncClientService struct {
	songRepo      repository.SongRepository
	trackRepo     repository.TrackRepository
	audioRepo     repository.AudioRepository
	client        *http.Client
	retryDelays   []time.Duration
	eventEmitter  EventEmitter
	mu            sync.RWMutex
	encryptionKey []byte
	cloudIDMu     sync.RWMutex
	cloudIDMap    map[int]int
}

// NewSyncClientService instantiates a SyncClientService with HTTP client timeout, exponential backoff, and local E2EE key.
func NewSyncClientService(
	songRepo repository.SongRepository,
	trackRepo repository.TrackRepository,
	audioRepo repository.AudioRepository,
) *SyncClientService {
	key, err := crypto.LoadOrGenerateKey("")
	if err != nil {
		// Fallback to in-memory random key if disk resolution failed
		key, _ = crypto.GenerateKey()
	}

	return &SyncClientService{
		songRepo:      songRepo,
		trackRepo:     trackRepo,
		audioRepo:     audioRepo,
		encryptionKey: key,
		cloudIDMap:    make(map[int]int),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryDelays:  []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second},
		eventEmitter: defaultEventEmitter,
	}
}

// SetCloudProjectID associates a local SQLite song ID with its remote Kizuna Cloud project ID.
func (s *SyncClientService) SetCloudProjectID(localSongID int, cloudProjectID int) {
	if localSongID <= 0 || cloudProjectID <= 0 {
		return
	}
	s.cloudIDMu.Lock()
	defer s.cloudIDMu.Unlock()
	if s.cloudIDMap == nil {
		s.cloudIDMap = make(map[int]int)
	}
	s.cloudIDMap[localSongID] = cloudProjectID
}

// GetCloudProjectID resolves the remote Kizuna Cloud project ID mapped to a local SQLite song ID.
func (s *SyncClientService) GetCloudProjectID(localSongID int) int {
	s.cloudIDMu.RLock()
	defer s.cloudIDMu.RUnlock()
	if s.cloudIDMap == nil {
		return 0
	}
	return s.cloudIDMap[localSongID]
}

// FindLocalSongIDByCloudProjectID resolves the local SQLite song ID mapped to a remote Kizuna Cloud project ID.
func (s *SyncClientService) FindLocalSongIDByCloudProjectID(cloudProjectID int) int {
	if cloudProjectID <= 0 {
		return 0
	}
	s.cloudIDMu.RLock()
	defer s.cloudIDMu.RUnlock()
	for localID, remoteID := range s.cloudIDMap {
		if remoteID == cloudProjectID {
			return localID
		}
	}
	return 0
}

// SetEventEmitter overrides the event emitter (useful for unit testing).
func (s *SyncClientService) SetEventEmitter(emitter EventEmitter) {
	s.eventEmitter = emitter
}

// SetRetryDelays overrides the retry backoff duration array (useful for fast unit testing).
func (s *SyncClientService) SetRetryDelays(delays []time.Duration) {
	s.retryDelays = delays
}

// isTemporaryNetworkError determines if an error should trigger an exponential backoff retry.
func isTemporaryNetworkError(err error, statusCode int) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
			return true
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return true
		}
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "broken pipe") ||
			strings.Contains(errStr, "eof") {
			return true
		}
	}

	// Transient HTTP status codes
	if statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		statusCode == http.StatusRequestTimeout {
		return true
	}

	return false
}

// doWithRetry executes an HTTP request operation with exponential backoff and progress tracking (Issue #23).
func (s *SyncClientService) doWithRetry(
	ctx context.Context,
	operation string,
	requestFactory func(attempt int, maxAttempts int) (*http.Request, *progressReader, error),
) (*http.Response, *progressReader, error) {
	maxAttempts := len(s.retryDelays)
	if maxAttempts == 0 {
		maxAttempts = 1
	}

	var lastErr error
	var lastStatus int

	for i := 0; i < maxAttempts; i++ {
		attempt := i + 1
		req, pr, err := requestFactory(attempt, maxAttempts)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build request (attempt %d): %w", attempt, err)
		}

		resp, err := s.client.Do(req)
		if err == nil && !isTemporaryNetworkError(nil, resp.StatusCode) {
			return resp, pr, nil
		}

		if resp != nil {
			lastStatus = resp.StatusCode
			_ = resp.Body.Close()
		}
		lastErr = err

		// Check if we should retry
		if i < maxAttempts-1 && isTemporaryNetworkError(err, lastStatus) {
			delay := s.retryDelays[i]
			reason := "server error"
			if lastErr != nil {
				reason = lastErr.Error()
			} else if lastStatus > 0 {
				reason = fmt.Sprintf("HTTP %d", lastStatus)
			}

			if s.eventEmitter != nil && ctx != nil {
				s.eventEmitter(ctx, "sync:retry", models.SyncRetryPayload{
					Operation:   operation,
					Attempt:     attempt,
					MaxAttempts: maxAttempts,
					WaitSeconds: int(delay.Seconds()),
					Reason:      reason,
				})
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, pr, ctx.Err()
			}
			continue
		}

		// Non-retryable error or exhausted retries
		break
	}

	if lastErr != nil {
		return nil, nil, fmt.Errorf("network request failed after %d attempts: %w", maxAttempts, lastErr)
	}
	return nil, nil, fmt.Errorf("network request rejected with temporary status %d after %d attempts", lastStatus, maxAttempts)
}

// SetEncryptionKey sets a custom 256-bit local encryption key (e.g. for key rotation or testing).
func (s *SyncClientService) SetEncryptionKey(key []byte) error {
	if len(key) != crypto.KeySize {
		return crypto.ErrInvalidKeySize
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]byte, crypto.KeySize)
	copy(copied, key)
	s.encryptionKey = copied
	return nil
}

// GetEncryptionKey returns a copy of the active 256-bit local E2EE key.
func (s *SyncClientService) GetEncryptionKey() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := make([]byte, len(s.encryptionKey))
	copy(copied, s.encryptionKey)
	return copied
}

// GetKeyFingerprint returns the short SHA-256 fingerprint of the local E2EE key.
func (s *SyncClientService) GetKeyFingerprint() string {
	return crypto.KeyFingerprint(s.GetEncryptionKey())
}

// PackageProject gathers song metadata, tracks, and audio versions into a sync payload,
// and attaches the AES-256-GCM encrypted bundle (SQLite + audio files).
func (s *SyncClientService) PackageProject(ctx context.Context, songID int, baseVersion int) (*models.CloudPushPayload, error) {
	if songID <= 0 {
		return nil, fmt.Errorf("invalid song ID: %d", songID)
	}

	song, err := s.songRepo.GetByID(ctx, songID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve song: %w", err)
	}

	tracks, err := s.trackRepo.GetBySongID(ctx, songID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tracks for song %d: %w", songID, err)
	}

	audioVersions, err := s.audioRepo.GetBySongID(ctx, songID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audio versions for song %d: %w", songID, err)
	}

	ciphertext, err := s.encryptBundle(song, tracks, audioVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt project bundle: %w", err)
	}

	cloudProjID := 0
	if baseVersion > 0 {
		if mappedID := s.GetCloudProjectID(songID); mappedID > 0 {
			cloudProjID = mappedID
		} else {
			cloudProjID = song.ID
		}
	}

	contentHash := models.ComputeProjectHash(
		song.Title,
		song.BPM,
		song.TimeSignature,
		song.KeySignature,
		tracks,
		audioVersions,
	)

	return &models.CloudPushPayload{
		ProjectID:        cloudProjID,
		Title:            song.Title,
		BPM:              song.BPM,
		TimeSignature:    song.TimeSignature,
		KeySignature:     song.KeySignature,
		BaseVersion:      baseVersion,
		ContentHash:      contentHash,
		Tracks:           tracks,
		AudioVersions:    audioVersions,
		EncryptedPayload: ciphertext,
		EncryptionAlgo:   crypto.AlgorithmAES256GCM,
		IsEncrypted:      true,
	}, nil
}

// PackageEncryptedProject creates a Zero-Knowledge E2EE payload where all SQLite composition data
// and physical .wav audio files are encrypted inside EncryptedPayload and stripped from plaintext fields.
func (s *SyncClientService) PackageEncryptedProject(
	ctx context.Context,
	songID int,
	remoteProjectID int,
	baseVersion int,
) (*models.CloudPushPayload, error) {
	rawPayload, err := s.PackageProject(ctx, songID, baseVersion)
	if err != nil {
		return nil, err
	}

	return &models.CloudPushPayload{
		ProjectID:        remoteProjectID,
		Title:            MaskedCloudTitle,
		BPM:              0,
		TimeSignature:    "4/4",
		KeySignature:     "",
		BaseVersion:      baseVersion,
		Tracks:           nil,
		AudioVersions:    nil,
		EncryptedPayload: rawPayload.EncryptedPayload,
		EncryptionAlgo:   crypto.AlgorithmAES256GCM,
		IsEncrypted:      true,
	}, nil
}

func (s *SyncClientService) encryptBundle(
	song *models.Song,
	tracks []models.Track,
	audioVersions []models.AudioVersion,
) (string, error) {
	var audioFiles []models.AudioFileBlob

	for i, tr := range tracks {
		if audioPath, ok := tr.GetAudioPath(); ok && strings.TrimSpace(audioPath) != "" {
			if data, err := os.ReadFile(audioPath); err == nil {
				audioFiles = append(audioFiles, models.AudioFileBlob{
					TrackIndex:   i,
					VersionIndex: -1,
					FileName:     filepath.Base(audioPath),
					OriginalPath: audioPath,
					Data:         data,
				})
			}
		}
	}

	for i, av := range audioVersions {
		if strings.TrimSpace(av.FilePath) != "" {
			if data, err := os.ReadFile(av.FilePath); err == nil {
				audioFiles = append(audioFiles, models.AudioFileBlob{
					TrackIndex:   -1,
					VersionIndex: i,
					FileName:     filepath.Base(av.FilePath),
					OriginalPath: av.FilePath,
					Data:         data,
				})
			}
		}
	}

	bundle := models.EncryptedProjectBundle{
		Title:         song.Title,
		BPM:           song.BPM,
		TimeSignature: song.TimeSignature,
		KeySignature:  song.KeySignature,
		Tracks:        tracks,
		AudioVersions: audioVersions,
		AudioFiles:    audioFiles,
	}

	bundleBytes, err := json.Marshal(bundle)
	if err != nil {
		return "", fmt.Errorf("failed to marshal project bundle: %w", err)
	}

	return crypto.Encrypt(bundleBytes, s.GetEncryptionKey())
}

// PushProjectToCloud packages the local song (SQLite + .wav audios), encrypts it E2EE with AES-256-GCM,
// and pushes the Zero-Knowledge payload to the Kizuna Cloud server with progress and retry backoff (Issues #13, #23).
func (s *SyncClientService) PushProjectToCloud(
	ctx context.Context,
	cloudURL string,
	token string,
	songID int,
	baseVersion int,
) (*models.CloudPushResult, error) {
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("authentication token is required to sync with Kizuna Cloud")
	}

	remoteProjectID := 0
	if baseVersion > 0 {
		s.mu.RLock()
		if mappedID, ok := s.cloudIDMap[songID]; ok {
			remoteProjectID = mappedID
		} else {
			remoteProjectID = songID
		}
		s.mu.RUnlock()
	}

	res, err := s.executeEncryptedPush(ctx, cloudURL, token, songID, remoteProjectID, baseVersion)
	if err != nil && remoteProjectID > 0 && strings.Contains(err.Error(), "cloud project not found") {
		// Self-heal if cloud server restarted and lost in-memory state
		res, err = s.executeEncryptedPush(ctx, cloudURL, token, songID, 0, 0)
	}
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cloudIDMap[songID] = res.ProjectID
	s.mu.Unlock()

	return res, nil
}

func (s *SyncClientService) executeEncryptedPush(
	ctx context.Context,
	cloudURL string,
	token string,
	songID int,
	remoteProjectID int,
	baseVersion int,
) (*models.CloudPushResult, error) {
	payload, err := s.PackageEncryptedProject(ctx, songID, remoteProjectID, baseVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to package encrypted project: %w", err)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize sync payload: %w", err)
	}

	totalBytes := int64(len(bodyBytes))
	endpoint := fmt.Sprintf("%s/api/v1/sync/push", strings.TrimRight(cloudURL, "/"))

	resp, pr, err := s.doWithRetry(ctx, "push", func(attempt, maxAttempts int) (*http.Request, *progressReader, error) {
		byteReader := bytes.NewReader(bodyBytes)
		reader := newProgressReader(ctx, byteReader, "push", totalBytes, attempt, maxAttempts, s.eventEmitter)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reader)
		if reqErr != nil {
			return nil, nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.ContentLength = totalBytes
		return req, reader, nil
	})

	if err != nil {
		if s.eventEmitter != nil && ctx != nil {
			s.eventEmitter(ctx, "sync:progress", models.SyncProgressPayload{
				Operation:   "push",
				TotalBytes:  totalBytes,
				Percentage:  0,
				Phase:       "failed",
				MaxAttempts: len(s.retryDelays),
			})
		}
		return nil, fmt.Errorf("failed to connect to Kizuna Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                    `json:"success"`
		Data    *models.CloudPushResult `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("invalid server response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("sync push rejected (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	if apiResp.Data == nil {
		return nil, fmt.Errorf("server returned empty data in response")
	}

	if apiResp.Data.ProjectID > 0 {
		s.SetCloudProjectID(songID, apiResp.Data.ProjectID)
	}

	if pr != nil {
		pr.emitProgress("completed", totalBytes, totalBytes, 100.0)
	}

	return apiResp.Data, nil
}

// DecryptCloudProject decrypts an E2EE CloudProject in-place using the client's local key
// and returns the unpacked EncryptedProjectBundle (including any embedded .wav audio files).
func (s *SyncClientService) DecryptCloudProject(cp *models.CloudProject) (*models.EncryptedProjectBundle, error) {
	if cp == nil {
		return nil, fmt.Errorf("cloud project cannot be nil")
	}

	if cp.EncryptedPayload == "" && !cp.IsEncrypted {
		return &models.EncryptedProjectBundle{
			Title:         cp.Title,
			BPM:           cp.BPM,
			TimeSignature: cp.TimeSignature,
			KeySignature:  cp.KeySignature,
			Tracks:        cp.Tracks,
			AudioVersions: cp.AudioVersions,
		}, nil
	}

	if strings.TrimSpace(cp.EncryptedPayload) == "" {
		return nil, fmt.Errorf("cloud project is marked encrypted but has an empty encrypted_payload")
	}

	decryptedBytes, err := crypto.Decrypt(cp.EncryptedPayload, s.GetEncryptionKey())
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt cloud project with local key: %w", err)
	}

	var bundle models.EncryptedProjectBundle
	if err := json.Unmarshal(decryptedBytes, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted project bundle: %w", err)
	}

	cp.Title = bundle.Title
	cp.BPM = bundle.BPM
	cp.TimeSignature = bundle.TimeSignature
	cp.KeySignature = bundle.KeySignature
	cp.Tracks = bundle.Tracks
	cp.AudioVersions = bundle.AudioVersions

	return &bundle, nil
}

// ImportCloudProject decrypts (if encrypted) and imports a pulled cloud project into local SQLite tables
// and writes any embedded .wav audio files to local disk (Issue #14).
// If this cloud project was previously imported locally, it updates the existing song in-place
// instead of creating duplicate records in SQLite.
func (s *SyncClientService) ImportCloudProject(ctx context.Context, cp *models.CloudProject) (int, error) {
	if cp == nil {
		return 0, fmt.Errorf("cloud project cannot be nil")
	}

	bundle, err := s.DecryptCloudProject(cp)
	if err != nil {
		return 0, err
	}

	title := strings.TrimSpace(cp.Title)
	if title == "" {
		title = "Imported Cloud Project"
	}

	bpm := cp.BPM
	if bpm <= 0 {
		bpm = 120
	}
	timeSig := cp.TimeSignature
	if timeSig == "" {
		timeSig = "4/4"
	}

	var songID int
	existingLocalID := s.FindLocalSongIDByCloudProjectID(cp.ID)
	if existingLocalID > 0 {
		// Verify existing song in SQLite
		if existingSong, err := s.songRepo.GetByID(ctx, existingLocalID); err == nil && existingSong != nil {
			existingSong.Title = title
			existingSong.BPM = bpm
			existingSong.TimeSignature = timeSig
			existingSong.KeySignature = cp.KeySignature
			if err := s.songRepo.Update(ctx, nil, existingSong); err != nil {
				return 0, fmt.Errorf("failed to update local song: %w", err)
			}
			songID = existingLocalID
			// Clean existing tracks before re-inserting
			_ = s.trackRepo.DeleteBySongIDExcept(ctx, nil, songID, nil)
		}
	}

	if songID == 0 {
		song := &models.Song{
			Title:         title,
			BPM:           bpm,
			TimeSignature: timeSig,
			KeySignature:  cp.KeySignature,
		}

		newSongID, err := s.songRepo.Create(ctx, nil, song)
		if err != nil {
			return 0, fmt.Errorf("failed to create local song: %w", err)
		}
		songID = newSongID
	}

	// Restore any embedded .wav files to local disk and remap paths for the imported song
	tracksToImport := make([]models.Track, len(cp.Tracks))
	copy(tracksToImport, cp.Tracks)

	audioVersionsToImport := make([]models.AudioVersion, len(cp.AudioVersions))
	copy(audioVersionsToImport, cp.AudioVersions)

	if bundle != nil && len(bundle.AudioFiles) > 0 {
		for _, blob := range bundle.AudioFiles {
			if len(blob.Data) == 0 {
				continue
			}
			restoredPath, err := writeRestoredAudioBlob(songID, blob)
			if err != nil {
				continue
			}
			if blob.TrackIndex >= 0 && blob.TrackIndex < len(tracksToImport) {
				tracksToImport[blob.TrackIndex].DataContent = models.AudioContentPrefix + restoredPath
			}
			if blob.VersionIndex >= 0 && blob.VersionIndex < len(audioVersionsToImport) {
				audioVersionsToImport[blob.VersionIndex].FilePath = restoredPath
			}
		}
	}

	for _, tr := range tracksToImport {
		instID := tr.InstrumentID
		if instID <= 0 {
			instID = 1
		}
		mode := tr.DisplayMode
		if mode == "" {
			mode = models.DisplayModeBoth
		}

		newTrack := &models.Track{
			SongID:       songID,
			InstrumentID: instID,
			Name:         tr.Name,
			DataContent:  tr.DataContent,
			DisplayMode:  mode,
			IsMuted:      tr.IsMuted,
		}
		_, err := s.trackRepo.Create(ctx, nil, newTrack)
		if err != nil {
			return 0, fmt.Errorf("failed to create local track '%s': %w", tr.Name, err)
		}
	}

	var existingTakes []models.AudioVersion
	if existingLocalID > 0 {
		existingTakes, _ = s.audioRepo.GetBySongID(ctx, songID)
	}

	for _, av := range audioVersionsToImport {
		alreadyExists := false
		for _, ex := range existingTakes {
			if (ex.FilePath != "" && ex.FilePath == av.FilePath) || (ex.VersionName != "" && ex.VersionName == av.VersionName) {
				alreadyExists = true
				break
			}
		}
		if alreadyExists {
			continue
		}
		newAV := &models.AudioVersion{
			SongID:      songID,
			VersionName: av.VersionName,
			FilePath:    av.FilePath,
			Notes:       av.Notes,
		}
		_, _ = s.audioRepo.Create(ctx, newAV)
	}

	if cp.ID > 0 {
		s.SetCloudProjectID(songID, cp.ID)
	}

	return songID, nil
}

func writeRestoredAudioBlob(songID int, blob models.AudioFileBlob) (string, error) {
	baseName := strings.TrimSuffix(blob.FileName, filepath.Ext(blob.FileName))
	if baseName == "" {
		baseName = "restored_take"
	}

	targetPath, err := audio.ResolveProjectAudioPath(songID, baseName, 0)
	if err == nil {
		if mkErr := os.MkdirAll(filepath.Dir(targetPath), 0o755); mkErr == nil {
			if wErr := os.WriteFile(targetPath, blob.Data, 0o644); wErr == nil {
				return targetPath, nil
			}
		}
	}

	// Fallback to temp directory if OS appdata directory is read-only
	fallbackDir := filepath.Join(os.TempDir(), "kizuna_restored_audio", fmt.Sprintf("song_%d", songID))
	if err := os.MkdirAll(fallbackDir, 0o755); err != nil {
		return "", err
	}
	fileName := blob.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("%s.wav", baseName)
	}
	fallbackPath := filepath.Join(fallbackDir, fileName)
	if err := os.WriteFile(fallbackPath, blob.Data, 0o644); err != nil {
		return "", err
	}
	return fallbackPath, nil
}

// PullProjectFromCloud downloads a project from Kizuna Cloud, decrypts its E2EE bundle using the local key,
// and unpacks SQLite records + .wav audio files into the local workspace with progress and retry resilience (Issues #14, #23).
func (s *SyncClientService) PullProjectFromCloud(
	ctx context.Context,
	cloudURL string,
	token string,
	projectID int,
) (*models.CloudProject, int, error) {
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}
	if strings.TrimSpace(token) == "" {
		return nil, 0, fmt.Errorf("authentication token is required to pull from Kizuna Cloud")
	}
	if projectID <= 0 {
		return nil, 0, fmt.Errorf("invalid cloud project ID: %d", projectID)
	}

	endpoint := fmt.Sprintf("%s/api/v1/sync/pull?project_id=%d", strings.TrimRight(cloudURL, "/"), projectID)

	resp, _, err := s.doWithRetry(ctx, "pull", func(attempt, maxAttempts int) (*http.Request, *progressReader, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if reqErr != nil {
			return nil, nil, reqErr
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return req, nil, nil
	})

	if err != nil {
		if s.eventEmitter != nil && ctx != nil {
			s.eventEmitter(ctx, "sync:progress", models.SyncProgressPayload{
				Operation:   "pull",
				Percentage:  0,
				Phase:       "failed",
				MaxAttempts: len(s.retryDelays),
			})
		}
		return nil, 0, fmt.Errorf("failed to connect to Kizuna Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	contentLength := resp.ContentLength
	reader := newProgressReader(ctx, resp.Body, "pull", contentLength, 1, len(s.retryDelays), s.eventEmitter)

	var apiResp struct {
		Success bool                 `json:"success"`
		Data    *models.CloudProject `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&apiResp); err != nil {
		return nil, 0, fmt.Errorf("invalid server response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("sync pull rejected (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			errMsg = apiResp.Error.Message
		}
		return nil, 0, fmt.Errorf("%s", errMsg)
	}

	if apiResp.Data == nil {
		return nil, 0, fmt.Errorf("server returned empty data for project %d", projectID)
	}

	if apiResp.Data.ID <= 0 && projectID > 0 {
		apiResp.Data.ID = projectID
	}

	reader.emitProgress("completed", reader.transferred, reader.transferred, 100.0)

	localSongID, err := s.ImportCloudProject(ctx, apiResp.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to import cloud project into local database: %w", err)
	}

	if localSongID > 0 && projectID > 0 {
		s.SetCloudProjectID(localSongID, projectID)
	}

	return apiResp.Data, localSongID, nil
}

// ListCloudProjects fetches the user's projects from Kizuna Cloud and decrypts their metadata locally.
func (s *SyncClientService) ListCloudProjects(
	ctx context.Context,
	cloudURL string,
	token string,
) ([]models.CloudProject, error) {
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("authentication token is required to list cloud projects")
	}

	endpoint := fmt.Sprintf("%s/api/v1/projects", strings.TrimRight(cloudURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list projects request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Kizuna Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                  `json:"success"`
		Data    []models.CloudProject `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("invalid server response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("list projects rejected (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	result := make([]models.CloudProject, 0, len(apiResp.Data))
	for _, cp := range apiResp.Data {
		projectCopy := cp
		_, _ = s.DecryptCloudProject(&projectCopy)
		result = append(result, projectCopy)
	}

	return result, nil
}

// ShareProject sends an invitation to Kizuna Cloud for a collaborator by email or username (Issue #15).
func (s *SyncClientService) ShareProject(
	ctx context.Context,
	cloudURL string,
	token string,
	cloudID string,
	collaboratorIdentifier string,
) error {
	cloudID = strings.TrimSpace(cloudID)
	collaboratorIdentifier = strings.TrimSpace(collaboratorIdentifier)

	if cloudID == "" {
		return errors.New("el identificador del proyecto remoto es inválido")
	}
	if collaboratorIdentifier == "" {
		return errors.New("debes ingresar el correo o nombre de usuario del colaborador")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("sesión expirada: inicia sesión en Kizuna Cloud para compartir")
	}
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}

	payload := map[string]string{
		"collaborator": collaboratorIdentifier,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error serializando petición: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/projects/%s/share", strings.TrimRight(cloudURL, "/"), cloudID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("error creando petición: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo conectar con Kizuna Cloud: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return errors.New("datos de invitación inválidos")
	case http.StatusUnauthorized:
		return errors.New("sesión no autorizada en Kizuna Cloud")
	case http.StatusForbidden:
		return errors.New("solo el propietario del proyecto puede invitar colaboradores")
	case http.StatusNotFound:
		return errors.New("Usuario no encontrado en Kizuna Cloud")
	case http.StatusConflict:
		return errors.New("el usuario ya es colaborador de este proyecto")
	default:
		return fmt.Errorf("error del servidor Kizuna Cloud (HTTP %d)", resp.StatusCode)
	}
}

// CheckoutProject requests an exclusive checkout lock on a cloud project before editing or syncing (Issue #16).
func (s *SyncClientService) CheckoutProject(
	ctx context.Context,
	cloudURL string,
	token string,
	projectID int,
) (*models.ProjectLock, error) {
	if projectID <= 0 {
		return nil, errors.New("el identificador del proyecto remoto es inválido")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("sesión expirada: inicia sesión en Kizuna Cloud para bloquear el proyecto")
	}
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}

	endpoint := fmt.Sprintf("%s/api/v1/projects/%d/checkout", strings.TrimRight(cloudURL, "/"), projectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición de check-out: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar con Kizuna Cloud: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                `json:"success"`
		Data    *models.ProjectLock `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("respuesta inválida del servidor (HTTP %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode == http.StatusConflict {
		msg := "project is currently checked out by another user"
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			msg = apiResp.Error.Message
		}
		return nil, errors.New(msg)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("sesión no autorizada en Kizuna Cloud")
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("error al solicitar candado (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			msg = apiResp.Error.Message
		}
		return nil, errors.New(msg)
	}

	return apiResp.Data, nil
}

// ReleaseProjectLock releases an active checkout lease on a cloud project (Issue #22).
func (s *SyncClientService) ReleaseProjectLock(
	ctx context.Context,
	cloudURL string,
	token string,
	projectID int,
) error {
	if projectID <= 0 {
		return errors.New("el identificador del proyecto remoto es inválido")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("sesión expirada: inicia sesión en Kizuna Cloud para liberar el candado")
	}
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}

	endpoint := fmt.Sprintf("%s/api/v1/projects/%d/release", strings.TrimRight(cloudURL, "/"), projectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creando petición de liberación: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo conectar con Kizuna Cloud: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return errors.New("sesión no autorizada en Kizuna Cloud")
	case http.StatusForbidden:
		return errors.New("no posees el candado activo de este proyecto")
	default:
		var apiResp struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiResp)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			return errors.New(apiResp.Error.Message)
		}
		return fmt.Errorf("error del servidor Kizuna Cloud al liberar candado (HTTP %d)", resp.StatusCode)
	}
}

// GetProjectLockStatus queries Kizuna Cloud to check if a project is currently locked (Issue #16).
func (s *SyncClientService) GetProjectLockStatus(
	ctx context.Context,
	cloudURL string,
	token string,
	projectID int,
) (*models.ProjectLock, error) {
	if projectID <= 0 {
		return nil, errors.New("el identificador del proyecto remoto es inválido")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("sesión expirada: inicia sesión en Kizuna Cloud")
	}
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}

	endpoint := fmt.Sprintf("%s/api/v1/projects/%d", strings.TrimRight(cloudURL, "/"), projectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando petición de estado de proyecto: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar con Kizuna Cloud: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                 `json:"success"`
		Data    *models.CloudProject `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("respuesta inválida del servidor (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("error al consultar estado del proyecto (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			msg = apiResp.Error.Message
		}
		return nil, errors.New(msg)
	}

	if apiResp.Data == nil {
		return nil, nil
	}

	return apiResp.Data.ActiveLock, nil
}
