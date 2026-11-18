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
	"fmt"
	"kizuna/client/internal/audio"
	"kizuna/client/internal/crypto"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const MaskedCloudTitle = "[E2EE] Encrypted Project"

// SyncClientService coordinates local project packaging, E2EE encryption/decryption, and cloud synchronization (Issues #13, #14).
type SyncClientService struct {
	songRepo      repository.SongRepository
	trackRepo     repository.TrackRepository
	audioRepo     repository.AudioRepository
	client        *http.Client
	mu            sync.RWMutex
	encryptionKey []byte
	cloudIDMap    map[int]int
}

// NewSyncClientService instantiates a SyncClientService with HTTP client timeout and local E2EE key.
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
	}
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

	return &models.CloudPushPayload{
		ProjectID:        song.ID,
		Title:            song.Title,
		BPM:              song.BPM,
		TimeSignature:    song.TimeSignature,
		KeySignature:     song.KeySignature,
		BaseVersion:      baseVersion,
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
// and pushes the Zero-Knowledge payload to the Kizuna Cloud server.
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

	endpoint := fmt.Sprintf("%s/api/v1/sync/push", strings.TrimRight(cloudURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
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

	song := &models.Song{
		Title:         title,
		BPM:           cp.BPM,
		TimeSignature: cp.TimeSignature,
		KeySignature:  cp.KeySignature,
	}
	if song.BPM <= 0 {
		song.BPM = 120
	}
	if song.TimeSignature == "" {
		song.TimeSignature = "4/4"
	}

	songID, err := s.songRepo.Create(ctx, nil, song)
	if err != nil {
		return 0, fmt.Errorf("failed to create local song: %w", err)
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

	for _, av := range audioVersionsToImport {
		newAV := &models.AudioVersion{
			SongID:      songID,
			VersionName: av.VersionName,
			FilePath:    av.FilePath,
			Notes:       av.Notes,
		}
		_, _ = s.audioRepo.Create(ctx, newAV)
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
// and unpacks SQLite records + .wav audio files into the local workspace (Issue #14).
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create pull request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to connect to Kizuna Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                 `json:"success"`
		Data    *models.CloudProject `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
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

	localSongID, err := s.ImportCloudProject(ctx, apiResp.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to import cloud project into local database: %w", err)
	}

	s.mu.Lock()
	s.cloudIDMap[localSongID] = projectID
	s.mu.Unlock()

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
