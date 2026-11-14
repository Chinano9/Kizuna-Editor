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
	"database/sql"
	"encoding/json"
	"io"
	"kizuna/client/internal/crypto"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func setupSyncTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(database.DefaultConfig(":memory:"))
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	m := migrator.NewMigrator(db)
	if err := m.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	if err := database.SeedInstruments(db); err != nil {
		t.Fatalf("failed to seed instruments: %v", err)
	}

	return db
}

func TestSyncClientService_PackageProject(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)

	t.Run("Rejects invalid song ID", func(t *testing.T) {
		_, err := svc.PackageProject(ctx, 0, 0)
		if err == nil {
			t.Fatal("expected error for song ID 0")
		}
	})

	t.Run("Packages complete project snapshot with tracks and audio", func(t *testing.T) {
		song := &models.Song{
			Title:         "Climbing Up The Walls",
			BPM:           90,
			TimeSignature: "4/4",
			KeySignature:  "B minor",
		}
		songID, err := songRepo.Create(ctx, nil, song)
		if err != nil {
			t.Fatalf("failed to create song: %v", err)
		}

		track := &models.Track{
			SongID:       songID,
			InstrumentID: 1,
			Name:         "Lead Guitar",
			DataContent:  `\title "Climbing Up The Walls"\n.`,
			DisplayMode:  "BOTH",
		}
		if _, err := trackRepo.Create(ctx, nil, track); err != nil {
			t.Fatalf("failed to create track: %v", err)
		}

		audioV := &models.AudioVersion{
			SongID:      songID,
			VersionName: "Take 1",
			FilePath:    "/tmp/test_take1.wav",
		}
		if _, err := audioRepo.Create(ctx, audioV); err != nil {
			t.Fatalf("failed to create audio version: %v", err)
		}

		payload, err := svc.PackageProject(ctx, songID, 1)
		if err != nil {
			t.Fatalf("failed to package project: %v", err)
		}

		if payload.Title != "Climbing Up The Walls" {
			t.Errorf("expected title 'Climbing Up The Walls', got %s", payload.Title)
		}
		if payload.BPM != 90 {
			t.Errorf("expected BPM 90, got %d", payload.BPM)
		}
		if len(payload.Tracks) != 1 || payload.Tracks[0].Name != "Lead Guitar" {
			t.Errorf("expected 1 track named 'Lead Guitar', got %+v", payload.Tracks)
		}
		if len(payload.AudioVersions) != 1 || payload.AudioVersions[0].VersionName != "Take 1" {
			t.Errorf("expected 1 audio version named 'Take 1', got %+v", payload.AudioVersions)
		}
		if payload.BaseVersion != 1 {
			t.Errorf("expected base version 1, got %d", payload.BaseVersion)
		}
	})
}

func TestSyncClientService_ProgressReader(t *testing.T) {
	ctx := context.Background()
	data := make([]byte, 1024*100) // 100 KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	var emittedProgress []models.SyncProgressPayload
	var mu sync.Mutex

	mockEmitter := func(ctx context.Context, eventName string, args ...interface{}) {
		if eventName == "sync:progress" && len(args) > 0 {
			if p, ok := args[0].(models.SyncProgressPayload); ok {
				mu.Lock()
				emittedProgress = append(emittedProgress, p)
				mu.Unlock()
			}
		}
	}

	rawReader := bytes.NewReader(data)
	pr := newProgressReader(ctx, rawReader, "push", int64(len(data)), 1, 3, mockEmitter)
	pr.throttleDelay = 0 // Instant emission for unit test verification

	buf := make([]byte, 1024*10)
	var totalRead int64
	for {
		n, err := pr.Read(buf)
		totalRead += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
	}

	if totalRead != int64(len(data)) {
		t.Fatalf("expected to read %d bytes, read %d", len(data), totalRead)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(emittedProgress) == 0 {
		t.Fatalf("expected progress events to be emitted, got 0")
	}

	last := emittedProgress[len(emittedProgress)-1]
	if last.BytesTransferred != int64(len(data)) {
		t.Errorf("expected final bytes transferred %d, got %d", len(data), last.BytesTransferred)
	}
	if last.Percentage != 100.0 {
		t.Errorf("expected final percentage 100.0, got %f", last.Percentage)
	}
}

func TestSyncClientService_PushProjectToCloud_RetryRecovery(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	song := &models.Song{
		Title:         "Street Spirit",
		BPM:           102,
		TimeSignature: "4/4",
	}
	songID, err := songRepo.Create(ctx, nil, song)
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	var attempts int32
	var retryEvents []models.SyncRetryPayload
	var mu sync.Mutex

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		// Fail the first two requests with temporary 502 Bad Gateway
		if current < 3 {
			http.Error(w, "temporary gateway failure", http.StatusBadGateway)
			return
		}

		// Succeed on 3rd attempt
		var req models.CloudPushPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": models.CloudPushResult{
				ProjectID:  5001,
				NewVersion: 2,
				SyncedAt:   time.Now().UTC(),
				Status:     "synced",
			},
		})
	}))
	defer mockServer.Close()

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)
	// Fast delays for unit testing
	svc.SetRetryDelays([]time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond})
	svc.SetEventEmitter(func(ctx context.Context, eventName string, args ...interface{}) {
		if eventName == "sync:retry" && len(args) > 0 {
			if r, ok := args[0].(models.SyncRetryPayload); ok {
				mu.Lock()
				retryEvents = append(retryEvents, r)
				mu.Unlock()
			}
		}
	})

	result, err := svc.PushProjectToCloud(ctx, mockServer.URL, "valid.token", songID, 1)
	if err != nil {
		t.Fatalf("expected successful push after retries, got err: %v", err)
	}

	if result.ProjectID != 5001 || result.NewVersion != 2 {
		t.Errorf("unexpected push result: %+v", result)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(retryEvents) != 2 {
		t.Errorf("expected 2 retry events, got %d", len(retryEvents))
	}
	if len(retryEvents) > 0 && retryEvents[0].Attempt != 1 {
		t.Errorf("expected first retry attempt to be 1, got %d", retryEvents[0].Attempt)
	}
}

func TestSyncClientService_PushProjectToCloud_MaxRetriesExceeded(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	song := &models.Song{Title: "Creep", BPM: 92, TimeSignature: "4/4"}
	songID, _ := songRepo.Create(ctx, nil, song)

	var attempts int32
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, "server down", http.StatusServiceUnavailable)
	}))
	defer mockServer.Close()

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)
	svc.SetRetryDelays([]time.Duration{5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond})

	_, err := svc.PushProjectToCloud(ctx, mockServer.URL, "valid.token", songID, 0)
	if err == nil {
		t.Fatal("expected error after exhausting all retries")
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts before giving up, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestSyncClientService_PushProjectToCloud_NonRetryableError(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	song := &models.Song{Title: "Karma Police", BPM: 75, TimeSignature: "4/4"}
	songID, _ := songRepo.Create(ctx, nil, song)

	var attempts int32
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		// 401 Unauthorized should NOT be retried
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    401,
				"message": "unauthorized token",
			},
		})
	}))
	defer mockServer.Close()

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)
	svc.SetRetryDelays([]time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond})

	_, err := svc.PushProjectToCloud(ctx, mockServer.URL, "invalid.token", songID, 0)
	if err == nil {
		t.Fatal("expected error for 401 unauthorized")
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected only 1 attempt for non-retryable 401, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestSyncClientService_PullProjectFromCloud(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)

	t.Run("Rejects invalid project ID or token", func(t *testing.T) {
		_, _, err := svc.PullProjectFromCloud(ctx, "http://localhost:8080", "", 10)
		if err == nil {
			t.Fatal("expected error with empty token")
		}

		_, _, err = svc.PullProjectFromCloud(ctx, "http://localhost:8080", "tok", 0)
		if err == nil {
			t.Fatal("expected error with 0 project ID")
		}
	})

	t.Run("Successfully pulls and imports project from cloud with progress", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/sync/pull" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if r.URL.Query().Get("project_id") != "1001" {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": models.CloudProject{
					ID:            1001,
					Title:         "Highway Star Remaster",
					BPM:           175,
					TimeSignature: "4/4",
					KeySignature:  "G",
					Version:       3,
					Tracks: []models.Track{
						{
							InstrumentID: 1,
							Name:         "Guitar Riff",
							DataContent:  "g.4 g.4 bb.4 c.4",
						},
					},
				},
			})
		}))
		defer mockServer.Close()

		var progressCount int
		svc.SetEventEmitter(func(ctx context.Context, eventName string, args ...interface{}) {
			if eventName == "sync:progress" {
				progressCount++
			}
		})

		cloudProject, songID, err := svc.PullProjectFromCloud(ctx, mockServer.URL, "pull.jwt.token", 1001)
		if err != nil {
			t.Fatalf("unexpected error pulling project: %v", err)
		}
		if cloudProject == nil || cloudProject.Title != "Highway Star Remaster" {
			t.Errorf("unexpected cloud project: %+v", cloudProject)
		}
		if songID <= 0 {
			t.Fatalf("expected valid imported song ID, got %d", songID)
		}

		if importedSong, err := songRepo.GetByID(ctx, songID); err != nil || importedSong.Title != "Highway Star Remaster" {
			t.Fatalf("failed to query imported song: %v", err)
		}

		if progressCount == 0 {
			t.Errorf("expected progress events to be emitted on pull")
		}
	})
}

func TestSyncClientService_ZeroKnowledgeE2EE_PushAndPullWithAudio(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)
	localKey := svc.GetEncryptionKey()
	if len(localKey) != crypto.KeySize {
		t.Fatalf("expected %d-byte local key, got %d", crypto.KeySize, len(localKey))
	}

	// 1. Create a local song with tracks and a physical .wav file on disk
	song := &models.Song{
		Title:         "Zero Knowledge Symphony",
		BPM:           128,
		TimeSignature: "4/4",
		KeySignature:  "E minor",
	}
	songID, err := songRepo.Create(ctx, nil, song)
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	_, err = trackRepo.Create(ctx, nil, &models.Track{
		SongID:       songID,
		InstrumentID: 1,
		Name:         "Guitarra Principal",
		DataContent:  `\tempo 128 (0.1 2.2).4 (3.1 5.2).4 |`,
		DisplayMode:  models.DisplayModeBoth,
	})
	if err != nil {
		t.Fatalf("failed to create track: %v", err)
	}

	wavContent := []byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x44\xac\x00\x00\x88\x58\x01\x00\x02\x00\x10\x00data\x04\x00\x00\x00\x11\x22\x33\x44")
	originalWavPath := filepath.Join(t.TempDir(), "solo_take.wav")
	if err := os.WriteFile(originalWavPath, wavContent, 0o644); err != nil {
		t.Fatalf("failed to write test wav: %v", err)
	}

	_, err = audioRepo.Create(ctx, &models.AudioVersion{
		SongID:      songID,
		VersionName: "Solo Take 1",
		FilePath:    originalWavPath,
		Notes:       "Recorded with condenser mic",
	})
	if err != nil {
		t.Fatalf("failed to create audio version: %v", err)
	}

	// 2. Mock Zero-Knowledge Cloud Server that stores whateverPushRequest it receives
	var storedOnServer models.CloudProject
	var rawPushWireBytes []byte

	mockCloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sync/push":
			rawBody, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			rawPushWireBytes = rawBody

			var req models.CloudPushPayload
			if err := json.Unmarshal(rawBody, &req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			storedOnServer = models.CloudProject{
				ID:               2025,
				OwnerID:          "usr_zk",
				Title:            req.Title,
				BPM:              req.BPM,
				TimeSignature:    req.TimeSignature,
				KeySignature:     req.KeySignature,
				Version:          1,
				Tracks:           req.Tracks,
				AudioVersions:    req.AudioVersions,
				EncryptedPayload: req.EncryptedPayload,
				EncryptionAlgo:   req.EncryptionAlgo,
				IsEncrypted:      req.IsEncrypted,
				CreatedAt:        time.Now().UTC(),
				UpdatedAt:        time.Now().UTC(),
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": models.CloudPushResult{
					ProjectID:  2025,
					NewVersion: 1,
					SyncedAt:   time.Now().UTC(),
					Status:     "created",
				},
			})

		case "/api/v1/sync/pull":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    storedOnServer,
			})

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer mockCloud.Close()

	// 3. Push project to cloud (E2EE)
	pushRes, err := svc.PushProjectToCloud(ctx, mockCloud.URL, "zk.jwt.token", songID, 0)
	if err != nil {
		t.Fatalf("E2EE push failed: %v", err)
	}
	if pushRes.ProjectID != 2025 {
		t.Fatalf("expected cloud project ID 2025, got %d", pushRes.ProjectID)
	}

	// Verify Zero-Knowledge guarantee on HTTP wire and server storage:
	// Neither the raw HTTP body nor the server's stored record may contain plaintext composition data!
	rawWireStr := string(rawPushWireBytes)
	for _, secret := range []string{"Zero Knowledge Symphony", "Guitarra Principal", "(0.1 2.2).4", "Solo Take 1", "Recorded with condenser mic"} {
		if strings.Contains(rawWireStr, secret) {
			t.Errorf("Zero-Knowledge violation: HTTP push payload exposed plaintext %q", secret)
		}
	}
	if !storedOnServer.IsEncrypted || storedOnServer.EncryptionAlgo != crypto.AlgorithmAES256GCM || storedOnServer.EncryptedPayload == "" {
		t.Fatalf("expected server to store AES-256-GCM encrypted payload, got %+v", storedOnServer)
	}
	if len(storedOnServer.Tracks) != 0 || len(storedOnServer.AudioVersions) != 0 {
		t.Fatalf("expected server plaintext Tracks and AudioVersions to be empty, got %d tracks and %d audio versions", len(storedOnServer.Tracks), len(storedOnServer.AudioVersions))
	}

	// 4. Verify that without the user's local key, pulling fails to decrypt
	attackerSvc := NewSyncClientService(songRepo, trackRepo, audioRepo)
	attackerKey, _ := crypto.GenerateKey()
	_ = attackerSvc.SetEncryptionKey(attackerKey)
	if _, _, err := attackerSvc.PullProjectFromCloud(ctx, mockCloud.URL, "zk.jwt.token", 2025); err == nil {
		t.Fatalf("expected pull with wrong local key to fail decryption")
	}

	// 5. Pull project with the rightful local key into a fresh database
	dbDest := setupSyncTestDB(t)
	defer dbDest.Close()

	destSongRepo := repository.NewSongRepository(dbDest)
	destTrackRepo := repository.NewTrackRepository(dbDest)
	destAudioRepo := repository.NewAudioRepository(dbDest)
	destSvc := NewSyncClientService(destSongRepo, destTrackRepo, destAudioRepo)
	_ = destSvc.SetEncryptionKey(localKey)

	pulledProject, importedSongID, err := destSvc.PullProjectFromCloud(ctx, mockCloud.URL, "zk.jwt.token", 2025)
	if err != nil {
		t.Fatalf("E2EE pull and decrypt failed: %v", err)
	}
	if pulledProject.Title != "Zero Knowledge Symphony" {
		t.Errorf("expected decrypted title 'Zero Knowledge Symphony', got %q", pulledProject.Title)
	}

	importedSong, err := destSongRepo.GetByID(ctx, importedSongID)
	if err != nil {
		t.Fatalf("failed to fetch imported song: %v", err)
	}
	if importedSong.Title != "Zero Knowledge Symphony" || importedSong.BPM != 128 || importedSong.KeySignature != "E minor" {
		t.Errorf("imported song metadata mismatch: %+v", importedSong)
	}

	importedTracks, err := destTrackRepo.GetBySongID(ctx, importedSongID)
	if err != nil || len(importedTracks) != 1 {
		t.Fatalf("expected 1 imported track, got %d (err=%v)", len(importedTracks), err)
	}
	if importedTracks[0].Name != "Guitarra Principal" || importedTracks[0].DataContent != `\tempo 128 (0.1 2.2).4 (3.1 5.2).4 |` {
		t.Errorf("imported track mismatch: %+v", importedTracks[0])
	}

	importedAudios, err := destAudioRepo.GetBySongID(ctx, importedSongID)
	if err != nil || len(importedAudios) != 1 {
		t.Fatalf("expected 1 imported audio version, got %d (err=%v)", len(importedAudios), err)
	}
	if importedAudios[0].VersionName != "Solo Take 1" {
		t.Errorf("expected audio version 'Solo Take 1', got %q", importedAudios[0].VersionName)
	}

	// Verify the restored .wav file on disk matches the original bytes
	restoredWavBytes, err := os.ReadFile(importedAudios[0].FilePath)
	if err != nil {
		t.Fatalf("failed to read restored wav file at %s: %v", importedAudios[0].FilePath, err)
	}
	if !bytes.Equal(restoredWavBytes, wavContent) {
		t.Errorf("restored wav file bytes do not match original wav content")
	}
}
