package bindings

import (
	"kizuna/client/internal/audio"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/client/internal/repository"
	"kizuna/client/internal/service"
	"testing"
)

func setupTestBindings(t *testing.T) (*SongHandler, *TrackHandler, *AlbumHandler, *AudioHandler, func()) {
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

	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	albumRepo := repository.NewAlbumRepository(db)
	audioRepo := repository.NewAudioRepository(db)
	quickIdeaRepo := repository.NewQuickIdeaRepository(db)
	instRepo := repository.NewInstrumentRepository(db)

	songService := service.NewSongService(db, songRepo, trackRepo)
	trackService := service.NewTrackService(trackRepo)
	albumService := service.NewAlbumService(albumRepo, songRepo)
	audioService := service.NewAudioService(audioRepo)
	quickIdeaService := service.NewQuickIdeaService(quickIdeaRepo)

	audioRec := audio.NewAudioRecorder()
	audioPlay := audio.NewAudioPlayer()

	songHandler := NewSongHandler(songService)
	trackHandler := NewTrackHandler(trackService, instRepo)
	albumHandler := NewAlbumHandler(albumService)
	audioHandler := NewAudioHandler(audioRec, audioPlay, audioService, quickIdeaService)

	cleanup := func() {
		_ = db.Close()
	}

	return songHandler, trackHandler, albumHandler, audioHandler, cleanup
}

func TestSongAndTrackHandlers(t *testing.T) {
	songH, trackH, _, _, cleanup := setupTestBindings(t)
	defer cleanup()

	// 1. SaveQuickIdea
	songID, err := songH.SaveQuickIdea(0, "Bindings Song", `\tempo 120 1.1 2.1`)
	if err != nil {
		t.Fatalf("failed to save quick idea: %v", err)
	}
	if songID <= 0 {
		t.Fatalf("expected positive ID, got %d", songID)
	}

	// 2. GetSong
	song, err := songH.GetSong(int(songID))
	if err != nil {
		t.Fatalf("failed to get song: %v", err)
	}
	if song.Title != "Bindings Song" || len(song.Tracks) != 1 {
		t.Errorf("song mismatch: %+v", song)
	}

	// 3. AddTrack
	newTrack, err := trackH.AddTrack(int(songID), "Rhythm Section")
	if err != nil {
		t.Fatalf("failed to add track: %v", err)
	}
	if newTrack.Name != "Rhythm Section" {
		t.Errorf("expected track name 'Rhythm Section', got %s", newTrack.Name)
	}

	// 4. UpdateTrack
	newTrack.IsMuted = true
	if err := trackH.UpdateTrack(newTrack); err != nil {
		t.Fatalf("failed to update track: %v", err)
	}

	// 5. DeleteTrack
	if err := trackH.DeleteTrack(newTrack.ID); err != nil {
		t.Fatalf("failed to delete track: %v", err)
	}

	// 6. GetInstruments
	instruments, err := trackH.GetInstruments()
	if err != nil {
		t.Fatalf("failed to get instruments: %v", err)
	}
	if len(instruments) != 5 {
		t.Errorf("expected 5 instruments, got %d", len(instruments))
	}

	// 7. GetRecentSongs
	recents, err := songH.GetRecentSongs()
	if err != nil {
		t.Fatalf("failed to get recent songs: %v", err)
	}
	if len(recents) != 1 {
		t.Errorf("expected 1 recent song, got %d", len(recents))
	}
}

func TestAlbumHandler(t *testing.T) {
	songH, _, albumH, _, cleanup := setupTestBindings(t)
	defer cleanup()

	// 1. CreateAlbum
	album, err := albumH.CreateAlbum("Greatest Hits", "Artist A", "Compilation")
	if err != nil {
		t.Fatalf("failed to create album: %v", err)
	}

	// 2. ListAlbums
	albums, err := albumH.ListAlbums()
	if err != nil || len(albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albums))
	}

	// 3. Assign song to album
	sID, _ := songH.SaveQuickIdea(0, "Hit Single", "")
	if err := albumH.AssignSongToAlbum(int(sID), album.ID); err != nil {
		t.Fatalf("failed to assign song to album: %v", err)
	}

	// 4. GetAlbumSongs
	songs, err := albumH.GetAlbumSongs(album.ID)
	if err != nil || len(songs) != 1 {
		t.Fatalf("expected 1 song in album, got %d", len(songs))
	}

	// 5. RemoveSongFromAlbum
	if err := albumH.RemoveSongFromAlbum(int(sID)); err != nil {
		t.Fatalf("failed to remove song from album: %v", err)
	}

	songsAfter, _ := albumH.GetAlbumSongs(album.ID)
	if len(songsAfter) != 0 {
		t.Errorf("expected 0 songs after removal, got %d", len(songsAfter))
	}
}

func TestAudioHandlerBasics(t *testing.T) {
	_, _, _, audioH, cleanup := setupTestBindings(t)
	defer cleanup()

	drivers := audioH.GetAvailableDrivers()
	if len(drivers) == 0 {
		t.Errorf("expected at least 1 available audio driver")
	}

	currDriver := audioH.GetCurrentDriver()
	if currDriver == "" {
		t.Errorf("expected non-empty current driver name")
	}

	level := audioH.GetAudioLevels()
	if level != 0.0 {
		t.Errorf("expected level 0.0 when not recording, got %f", level)
	}
}
