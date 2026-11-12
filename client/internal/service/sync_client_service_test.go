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
