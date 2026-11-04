package main

import (
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/shared/models"
	"testing"
)

func setupTestDBManager(t *testing.T) *DBManager {
	cfg := database.DefaultConfig(":memory:")
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	m := migrator.NewMigrator(db)
	if err := m.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	if err := database.SeedInstruments(db); err != nil {
		t.Fatalf("failed to seed instruments: %v", err)
	}

	return NewDBManagerWithDB(db)
}

func TestDBManager_SaveQuickIdeaAndGetSong(t *testing.T) {
	mgr := setupTestDBManager(t)
	defer mgr.Close()

	// 1. Create quick idea
	id, err := mgr.SaveQuickIdea(0, "Riff in D minor", `\tempo 130 5.3 7.3 8.3`)
	if err != nil {
		t.Fatalf("failed to save quick idea: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}

	// 2. Retrieve song
	song, err := mgr.GetSong(int(id))
	if err != nil {
		t.Fatalf("failed to get song: %v", err)
	}
	if song.Title != "Riff in D minor" {
		t.Errorf("expected title 'Riff in D minor', got '%s'", song.Title)
	}
	if len(song.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(song.Tracks))
	}
	if song.Tracks[0].DataContent != `\tempo 130 5.3 7.3 8.3` {
		t.Errorf("track content mismatch: %s", song.Tracks[0].DataContent)
	}

	// 3. Update quick idea
	updatedID, err := mgr.SaveQuickIdea(int(id), "Updated Riff in D minor", `\tempo 140 5.3 7.3 8.3`)
	if err != nil {
		t.Fatalf("failed to update quick idea: %v", err)
	}
	if updatedID != id {
		t.Errorf("expected same ID %d, got %d", id, updatedID)
	}

	updatedSong, err := mgr.GetSong(int(id))
	if err != nil {
		t.Fatalf("failed to get updated song: %v", err)
	}
	if updatedSong.Title != "Updated Riff in D minor" {
		t.Errorf("expected updated title, got '%s'", updatedSong.Title)
	}
}

func TestDBManager_TracksCRUDAndCascade(t *testing.T) {
	mgr := setupTestDBManager(t)
	defer mgr.Close()

	// 1. Save new song
	newSong := &models.Song{
		Title:         "Cascade Test",
		BPM:           120,
		TimeSignature: "4/4",
	}
	savedSong, err := mgr.SaveSong(newSong)
	if err != nil {
		t.Fatalf("failed to save song: %v", err)
	}

	// 2. Add track
	track, err := mgr.AddTrack(savedSong.ID, "Acoustic Rythm")
	if err != nil {
		t.Fatalf("failed to add track: %v", err)
	}
	if track.Name != "Acoustic Rythm" {
		t.Errorf("expected track name 'Acoustic Rythm', got '%s'", track.Name)
	}

	// 3. Update track
	track.Name = "Acoustic Rhythm (Updated)"
	track.IsMuted = true
	if err := mgr.UpdateTrack(track); err != nil {
		t.Fatalf("failed to update track: %v", err)
	}

	// 4. Add audio version linked to track
	av, err := mgr.AddAudioVersion(savedSong.ID, &track.ID, "Take 1", "/path/to/take1.wav", "Clean mic")
	if err != nil {
		t.Fatalf("failed to add audio version: %v", err)
	}
	if av.ID <= 0 {
		t.Fatalf("expected audio version to have positive ID")
	}

	// 5. Test CASCADE delete: deleting track should cascade delete its audio_version
	if err := mgr.DeleteTrack(track.ID); err != nil {
		t.Fatalf("failed to delete track: %v", err)
	}

	versions, err := mgr.GetAudioVersionsForTrack(track.ID)
	if err != nil {
		t.Fatalf("failed to get audio versions: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 audio versions after cascade delete, got %d", len(versions))
	}
}

func TestDBManager_GetInstruments(t *testing.T) {
	mgr := setupTestDBManager(t)
	defer mgr.Close()

	instruments, err := mgr.GetInstruments()
	if err != nil {
		t.Fatalf("failed to get instruments: %v", err)
	}
	if len(instruments) != 5 {
		t.Fatalf("expected 5 seeded instruments, got %d", len(instruments))
	}
}
