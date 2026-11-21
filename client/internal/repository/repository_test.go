package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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

	t.Run("Issue #21: Track and Audio changes update Song updated_at and ContentHash", func(t *testing.T) {
		sID, err := songRepo.Create(ctx, nil, &models.Song{
			Title:         "Session Hash Test",
			BPM:           120,
			TimeSignature: "4/4",
			KeySignature:  "Em",
		})
		if err != nil {
			t.Fatalf("failed to create song: %v", err)
		}

		// Force an older timestamp on songs.updated_at to verify touch on audio/track creation
		_, err = db.ExecContext(ctx, `UPDATE songs SET updated_at = '2025-01-01 00:00:00' WHERE id = ?`, sID)
		if err != nil {
			t.Fatalf("failed to backdate song updated_at: %v", err)
		}

		songBefore, err := songRepo.GetByID(ctx, sID)
		if err != nil {
			t.Fatalf("failed to get song before changes: %v", err)
		}
		if len(songBefore.ContentHash) != 16 {
			t.Fatalf("expected 16-char ContentHash, got %q", songBefore.ContentHash)
		}
		initialHash := songBefore.ContentHash
		initialUpdatedAt := songBefore.UpdatedAt

		// 1. Adding a track updates songs.updated_at and changes ContentHash
		trackID, err := trackRepo.Create(ctx, nil, &models.Track{
			SongID:       sID,
			InstrumentID: models.InstrumentElectricGuitarID,
			Name:         "Rhythm Guitar",
			DataContent:  ":4 2.5 2.4",
		})
		if err != nil {
			t.Fatalf("failed to create track: %v", err)
		}

		songAfterTrack, err := songRepo.GetByID(ctx, sID)
		if err != nil {
			t.Fatalf("failed to get song after track creation: %v", err)
		}
		if !songAfterTrack.UpdatedAt.After(initialUpdatedAt) {
			t.Errorf("expected songs.updated_at to advance after adding track: before=%v, after=%v", initialUpdatedAt, songAfterTrack.UpdatedAt)
		}
		if songAfterTrack.ContentHash == initialHash {
			t.Errorf("expected ContentHash to change after adding track")
		}
		hashAfterTrack := songAfterTrack.ContentHash

		// 2. Recording an audio take updates songs.updated_at and changes ContentHash
		_, _ = db.ExecContext(ctx, `UPDATE songs SET updated_at = '2025-01-02 00:00:00' WHERE id = ?`, sID)
		songMid, _ := songRepo.GetByID(ctx, sID)

		_, err = audioRepo.Create(ctx, &models.AudioVersion{
			SongID:      sID,
			TrackID:     &trackID,
			VersionName: "Take 1",
			FilePath:    "/audio/session_take1.wav",
			Notes:       "Live room",
		})
		if err != nil {
			t.Fatalf("failed to create audio take: %v", err)
		}

		songAfterAudio, err := songRepo.GetByID(ctx, sID)
		if err != nil {
			t.Fatalf("failed to get song after audio creation: %v", err)
		}
		if !songAfterAudio.UpdatedAt.After(songMid.UpdatedAt) {
			t.Errorf("expected songs.updated_at to advance after recording audio take: before=%v, after=%v", songMid.UpdatedAt, songAfterAudio.UpdatedAt)
		}
		if songAfterAudio.ContentHash == hashAfterTrack {
			t.Errorf("expected ContentHash to change after recording audio take")
		}

		// 3. GetRecent includes the computed ContentHash for the dashboard list
		recent, err := songRepo.GetRecent(ctx, 10)
		if err != nil {
			t.Fatalf("failed to get recent songs: %v", err)
		}
		var foundInRecent bool
		for _, rs := range recent {
			if rs.ID == sID {
				foundInRecent = true
				if rs.ContentHash != songAfterAudio.ContentHash {
					t.Errorf("expected GetRecent ContentHash %q to match GetByID %q", rs.ContentHash, songAfterAudio.ContentHash)
				}
			}
		}
		if !foundInRecent {
			t.Errorf("expected song %d in GetRecent results", sID)
		}
	})

	t.Run("Empty Query Non-Nil Slice and JSON Array Marshalling", func(t *testing.T) {
		// Verify fresh DB without songs produces non-nil empty slices that serialize to "[]"
		freshDB, err := database.Open(database.DefaultConfig(":memory:"))
		if err != nil {
			t.Fatalf("failed to open in-memory db: %v", err)
		}
		defer freshDB.Close()

		m := migrator.NewMigrator(freshDB)
		if err := m.Migrate(); err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}

		freshSongRepo := NewSongRepository(freshDB)
		freshAlbumRepo := NewAlbumRepository(freshDB)
		freshTrackRepo := NewTrackRepository(freshDB)
		freshAudioRepo := NewAudioRepository(freshDB)

		// 1. GetRecent on empty songs table
		recentSongs, err := freshSongRepo.GetRecent(ctx, 10)
		if err != nil {
			t.Fatalf("unexpected error querying recent songs on empty DB: %v", err)
		}
		if recentSongs == nil {
			t.Fatalf("expected non-nil slice from GetRecent on empty DB, got nil")
		}
		songBytes, err := json.Marshal(recentSongs)
		if err != nil {
			t.Fatalf("failed to marshal empty songs slice: %v", err)
		}
		if string(songBytes) != "[]" {
			t.Fatalf("expected JSON '[]' for empty songs, got '%s'", string(songBytes))
		}

		// 2. GetByAlbumID on empty songs table
		albumSongs, err := freshSongRepo.GetByAlbumID(ctx, 999)
		if err != nil || albumSongs == nil {
			t.Fatalf("expected non-nil slice for empty album songs, got err=%v, res=%v", err, albumSongs)
		}
		if b, _ := json.Marshal(albumSongs); string(b) != "[]" {
			t.Fatalf("expected JSON '[]', got '%s'", string(b))
		}

		// 3. GetAll on empty albums table
		emptyAlbums, err := freshAlbumRepo.GetAll(ctx)
		if err != nil || emptyAlbums == nil {
			t.Fatalf("expected non-nil slice for empty albums, got err=%v, res=%v", err, emptyAlbums)
		}
		if b, _ := json.Marshal(emptyAlbums); string(b) != "[]" {
			t.Fatalf("expected JSON '[]', got '%s'", string(b))
		}

		// 4. GetBySongID on empty tracks table
		emptyTracks, err := freshTrackRepo.GetBySongID(ctx, 999)
		if err != nil || emptyTracks == nil {
			t.Fatalf("expected non-nil slice for empty tracks, got err=%v, res=%v", err, emptyTracks)
		}
		if b, _ := json.Marshal(emptyTracks); string(b) != "[]" {
			t.Fatalf("expected JSON '[]', got '%s'", string(b))
		}

		// 5. GetBySongID on empty audio_versions table
		emptyAudio, err := freshAudioRepo.GetBySongID(ctx, 999)
		if err != nil || emptyAudio == nil {
			t.Fatalf("expected non-nil slice for empty audio takes, got err=%v, res=%v", err, emptyAudio)
		}
		if b, _ := json.Marshal(emptyAudio); string(b) != "[]" {
			t.Fatalf("expected JSON '[]', got '%s'", string(b))
		}
	})
}
