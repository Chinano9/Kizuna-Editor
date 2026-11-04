package main

import (
	"context"
	"kizuna/client/bindings"
	"kizuna/client/internal/audio"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"testing"
)

func setupTestApp(t *testing.T) *App {
	dbConn, err := database.Open(database.DefaultConfig(":memory:"))
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	m := migrator.NewMigrator(dbConn)
	if err := m.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	if err := database.SeedInstruments(dbConn); err != nil {
		t.Fatalf("failed to seed instruments: %v", err)
	}

	db := NewDBManagerWithDB(dbConn)
	audioRec := audio.NewAudioRecorder()
	audioPlay := audio.NewAudioPlayer()

	songHandler := bindings.NewSongHandler(db.SongService())
	trackHandler := bindings.NewTrackHandler(db.TrackService(), db.InstrumentRepository())
	albumHandler := bindings.NewAlbumHandler(db.AlbumService())
	audioHandler := bindings.NewAudioHandler(audioRec, audioPlay, db.AudioService())

	app := &App{
		db:           db,
		audioRec:     audioRec,
		audioPlay:    audioPlay,
		songHandler:  songHandler,
		trackHandler: trackHandler,
		albumHandler: albumHandler,
		audioHandler: audioHandler,
	}
	app.startup(context.Background())
	return app
}

func TestApp_LifecycleAndDelegation(t *testing.T) {
	app := setupTestApp(t)
	defer app.shutdown(context.Background())

	// 1. Song operations
	id, err := app.SaveQuickIdea(0, "App Test Song", `\tempo 120 1.1 2.1`)
	if err != nil {
		t.Fatalf("failed to save quick idea: %v", err)
	}

	song, err := app.GetSong(int(id))
	if err != nil {
		t.Fatalf("failed to get song: %v", err)
	}
	if song.Title != "App Test Song" {
		t.Errorf("expected title 'App Test Song', got %s", song.Title)
	}

	// 2. Track operations
	newTrack, err := app.AddTrack(int(id), "Overdrive Guitar")
	if err != nil {
		t.Fatalf("failed to add track: %v", err)
	}
	if newTrack.Name != "Overdrive Guitar" {
		t.Errorf("expected track name 'Overdrive Guitar', got %s", newTrack.Name)
	}

	newTrack.Name = "Clean Guitar"
	if err := app.UpdateTrack(newTrack); err != nil {
		t.Fatalf("failed to update track: %v", err)
	}

	instruments, err := app.GetInstruments()
	if err != nil || len(instruments) != 5 {
		t.Fatalf("expected 5 instruments, got %d", len(instruments))
	}

	// 3. Album operations (Dev 3 - Issues #6, #7, #8, #17, #18)
	album, err := app.CreateAlbum("Studio Session", "The Band", "Live in studio")
	if err != nil {
		t.Fatalf("failed to create album: %v", err)
	}

	if err := app.AssignSongToAlbum(int(id), album.ID); err != nil {
		t.Fatalf("failed to assign song to album: %v", err)
	}

	albumSongs, err := app.GetAlbumSongs(album.ID)
	if err != nil || len(albumSongs) != 1 {
		t.Fatalf("expected 1 song in album, got %d", len(albumSongs))
	}

	albums, err := app.ListAlbums()
	if err != nil || len(albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albums))
	}

	if err := app.RemoveSongFromAlbum(int(id)); err != nil {
		t.Fatalf("failed to remove song from album: %v", err)
	}

	// 4. Audio operations
	drivers := app.GetAvailableDrivers()
	if len(drivers) == 0 {
		t.Errorf("expected audio driver list")
	}

	currDriver := app.GetCurrentDriver()
	if currDriver == "" {
		t.Errorf("expected current driver name")
	}

	level := app.GetAudioLevels()
	if level != 0.0 {
		t.Errorf("expected 0.0 level when inactive")
	}

	// 5. Cleanup track
	if err := app.DeleteTrack(newTrack.ID); err != nil {
		t.Fatalf("failed to delete track: %v", err)
	}
}
