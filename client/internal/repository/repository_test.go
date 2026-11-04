package repository

import (
	"context"
	"database/sql"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/shared/models"
	"testing"
)

func setupTestDB(t *testing.T) *sql.DB {
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

func TestRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	songRepo := NewSongRepository(db)
	trackRepo := NewTrackRepository(db)
	albumRepo := NewAlbumRepository(db)
	instRepo := NewInstrumentRepository(db)
	audioRepo := NewAudioRepository(db)

	t.Run("AlbumRepository CRUD", func(t *testing.T) {
		album := &models.Album{
			Title:       "Abbey Road",
			Artist:      "The Beatles",
			Description: "11th studio album",
		}
		albumID, err := albumRepo.Create(ctx, album)
		if err != nil {
			t.Fatalf("failed to create album: %v", err)
		}
		if albumID <= 0 {
			t.Fatalf("expected positive album ID, got %d", albumID)
		}

		fetched, err := albumRepo.GetByID(ctx, albumID)
		if err != nil {
			t.Fatalf("failed to fetch album: %v", err)
		}
		if fetched.Title != "Abbey Road" || fetched.Artist != "The Beatles" {
			t.Errorf("mismatch in fetched album: %+v", fetched)
		}

		fetched.Title = "Abbey Road (Remastered)"
		if err := albumRepo.Update(ctx, fetched); err != nil {
			t.Fatalf("failed to update album: %v", err)
		}

		all, err := albumRepo.GetAll(ctx)
		if err != nil {
			t.Fatalf("failed to list albums: %v", err)
		}
		if len(all) != 1 || all[0].Title != "Abbey Road (Remastered)" {
			t.Errorf("unexpected albums list: %+v", all)
		}

		if err := albumRepo.Delete(ctx, albumID); err != nil {
			t.Fatalf("failed to delete album: %v", err)
		}
		_, err = albumRepo.GetByID(ctx, albumID)
		if err == nil {
			t.Errorf("expected error when fetching deleted album")
		}
	})

	t.Run("SongRepository and TrackRepository", func(t *testing.T) {
		song := &models.Song{
			Title:         "Come Together",
			BPM:           84,
			TimeSignature: "4/4",
			KeySignature:  "D minor",
		}
		songID, err := songRepo.Create(ctx, nil, song)
		if err != nil {
			t.Fatalf("failed to create song: %v", err)
		}

		// Add tracks
		t1 := &models.Track{
			SongID:       songID,
			InstrumentID: models.InstrumentBassID,
			Name:         "Iconic Bass Riff",
			DataContent:  ":4 5.3 7.3",
		}
		t1ID, err := trackRepo.Create(ctx, nil, t1)
		if err != nil {
			t.Fatalf("failed to create track 1: %v", err)
		}

		t2 := &models.Track{
			SongID:       songID,
			InstrumentID: models.InstrumentElectricGuitarID,
			Name:         "Guitar Slide",
			DataContent:  ":4 7.2 9.2",
		}
		t2ID, err := trackRepo.Create(ctx, nil, t2)
		if err != nil {
			t.Fatalf("failed to create track 2: %v", err)
		}

		tracks, err := trackRepo.GetBySongID(ctx, songID)
		if err != nil {
			t.Fatalf("failed to get tracks: %v", err)
		}
		if len(tracks) != 2 {
			t.Fatalf("expected 2 tracks, got %d", len(tracks))
		}

		// Delete track 2 using DeleteBySongIDExcept keepIDs=[t1ID]
		if err := trackRepo.DeleteBySongIDExcept(ctx, nil, songID, []int{t1ID}); err != nil {
			t.Fatalf("failed to delete tracks except keepIDs: %v", err)
		}

		remaining, err := trackRepo.GetBySongID(ctx, songID)
		if err != nil {
			t.Fatalf("failed to get tracks: %v", err)
		}
		if len(remaining) != 1 || remaining[0].ID != t1ID {
			t.Errorf("expected only track %d remaining, got %+v", t1ID, remaining)
		}

		_ = t2ID
	})

	t.Run("InstrumentRepository", func(t *testing.T) {
		instruments, err := instRepo.GetAll(ctx)
		if err != nil {
			t.Fatalf("failed to get instruments: %v", err)
		}
		if len(instruments) != 5 {
			t.Errorf("expected 5 instruments, got %d", len(instruments))
		}

		inst1, err := instRepo.GetByID(ctx, models.InstrumentElectricGuitarID)
		if err != nil {
			t.Fatalf("failed to get instrument 1: %v", err)
		}
		if inst1.Name != "Electric Guitar" {
			t.Errorf("expected Electric Guitar, got %s", inst1.Name)
		}
	})

	t.Run("AudioRepository", func(t *testing.T) {
		// Create a song first
		sID, err := songRepo.Create(ctx, nil, &models.Song{Title: "Vocal Idea", BPM: 100})
		if err != nil {
			t.Fatalf("failed to create song: %v", err)
		}

		av := &models.AudioVersion{
			SongID:      sID,
			TrackID:     nil, // Global
			VersionName: "Take 1",
			FilePath:    "/audio/vocal.wav",
			Notes:       "Acoustic room",
		}

		avID, err := audioRepo.Create(ctx, av)
		if err != nil {
			t.Fatalf("failed to create audio version: %v", err)
		}

		fetchedAV, err := audioRepo.GetByID(ctx, avID)
		if err != nil {
			t.Fatalf("failed to get audio version: %v", err)
		}
		if !fetchedAV.IsGlobal() {
			t.Errorf("expected audio version to be global")
		}

		takes, err := audioRepo.GetBySongID(ctx, sID)
		if err != nil {
			t.Fatalf("failed to query takes by song: %v", err)
		}
		if len(takes) != 1 {
			t.Errorf("expected 1 take, got %d", len(takes))
		}
	})
}
