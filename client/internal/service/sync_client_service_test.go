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
	"context"
	"database/sql"
	"encoding/json"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"net/http"
	"net/http/httptest"
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
			SongID:        songID,
			InstrumentID:  1,
			Name:          "Lead Guitar",
			DataContent:   `\title "Climbing Up The Walls"\n.`,
			DisplayMode:   "BOTH",
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

func TestSyncClientService_PushProjectToCloud(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	song := &models.Song{
		Title:         "Paranoid Android",
		BPM:           82,
		TimeSignature: "4/4",
	}
	songID, err := songRepo.Create(ctx, nil, song)
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	t.Run("Requires authentication token", func(t *testing.T) {
		svc := NewSyncClientService(songRepo, trackRepo, audioRepo)
		_, err := svc.PushProjectToCloud(ctx, "http://localhost:8080", "", songID, 0)
		if err == nil {
			t.Fatal("expected error when token is empty")
		}
	})

	t.Run("Successfully pushes project to cloud endpoint", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/sync/push" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if r.Header.Get("Authorization") != "Bearer valid.jwt.token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			var req models.CloudPushPayload
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": models.CloudPushResult{
					ProjectID:  9999,
					NewVersion: 1,
					SyncedAt:   time.Now().UTC(),
					Status:     "created",
				},
			})
		}))
		defer mockServer.Close()

		svc := NewSyncClientService(songRepo, trackRepo, audioRepo)
		result, err := svc.PushProjectToCloud(ctx, mockServer.URL, "valid.jwt.token", songID, 0)
		if err != nil {
			t.Fatalf("unexpected error pushing project: %v", err)
		}

		if result.ProjectID != 9999 || result.NewVersion != 1 || result.Status != "created" {
			t.Errorf("unexpected push result: %+v", result)
		}
	})
}

func TestSyncClientService_ImportCloudProject(t *testing.T) {
	db := setupSyncTestDB(t)
	defer db.Close()

	ctx := context.Background()
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)

	svc := NewSyncClientService(songRepo, trackRepo, audioRepo)

	t.Run("Rejects nil cloud project", func(t *testing.T) {
		_, err := svc.ImportCloudProject(ctx, nil)
		if err == nil {
			t.Fatal("expected error for nil project")
		}
	})

	t.Run("Successfully imports cloud project with tracks and audio into local database", func(t *testing.T) {
		cp := &models.CloudProject{
			ID:            555,
			Title:         "Starlight Serenade",
			BPM:           140,
			TimeSignature: "3/4",
			KeySignature:  "D#",
			Version:       2,
			Tracks: []models.Track{
				{
					InstrumentID: 1,
					Name:         "Lead Guitar",
					DataContent:  "c.4 d.4 e.4",
					DisplayMode:  "BOTH",
					IsMuted:      false,
				},
				{
					InstrumentID: 2,
					Name:         "Bass",
					DataContent:  "c.2 g.2",
					DisplayMode:  "STANDARD",
					IsMuted:      true,
				},
			},
			AudioVersions: []models.AudioVersion{
				{
					VersionName: "Demo Take 1",
					FilePath:    "/tmp/demo_take_1.wav",
					Notes:       "Acoustic demo take",
				},
			},
		}

		songID, err := svc.ImportCloudProject(ctx, cp)
		if err != nil {
			t.Fatalf("unexpected error importing cloud project: %v", err)
		}
		if songID <= 0 {
			t.Fatalf("expected positive song ID, got %d", songID)
		}

		// Verify song was inserted
		savedSong, err := songRepo.GetByID(ctx, songID)
		if err != nil {
			t.Fatalf("failed to retrieve imported song: %v", err)
		}
		if savedSong.Title != "Starlight Serenade" || savedSong.BPM != 140 || savedSong.TimeSignature != "3/4" {
			t.Errorf("imported song mismatch: %+v", savedSong)
		}

		// Verify tracks
		tracks, err := trackRepo.GetBySongID(ctx, songID)
		if err != nil {
			t.Fatalf("failed to retrieve tracks: %v", err)
		}
		if len(tracks) != 2 {
			t.Fatalf("expected 2 tracks, got %d", len(tracks))
		}
		if tracks[0].Name != "Lead Guitar" || tracks[1].Name != "Bass" {
			t.Errorf("unexpected tracks: %+v", tracks)
		}

		// Verify audio versions
		audioList, err := audioRepo.GetBySongID(ctx, songID)
		if err != nil {
			t.Fatalf("failed to retrieve audio: %v", err)
		}
		if len(audioList) != 1 || audioList[0].VersionName != "Demo Take 1" {
			t.Errorf("unexpected audio versions: %+v", audioList)
		}
	})
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

	t.Run("Successfully pulls and imports project from cloud", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/sync/pull" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if r.URL.Query().Get("project_id") != "1001" {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}
			if r.Header.Get("Authorization") != "Bearer pull.jwt.token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
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

		importedSong, err := songRepo.GetByID(ctx, songID)
		if err != nil || importedSong.Title != "Highway Star Remaster" {
			t.Fatalf("failed to query imported song: %v", err)
		}
	})
}
