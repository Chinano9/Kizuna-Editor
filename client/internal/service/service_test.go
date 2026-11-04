package service

import (
	"context"
	"database/sql"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"testing"
)

func setupTestServices(t *testing.T) (*sql.DB, *SongService, *TrackService, *AlbumService, *AudioService) {
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

	songService := NewSongService(db, songRepo, trackRepo)
	trackService := NewTrackService(trackRepo)
	albumService := NewAlbumService(albumRepo, songRepo)
	audioService := NewAudioService(audioRepo)

	return db, songService, trackService, albumService, audioService
}

func TestSongService_SaveQuickIdeaAndSaveSong(t *testing.T) {
	db, songService, _, _, _ := setupTestServices(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Save new quick idea
	songID, err := songService.SaveQuickIdea(ctx, 0, "Acoustic Melody", `\tempo 110 :4 1.1 3.1 5.1`)
	if err != nil {
		t.Fatalf("failed to save quick idea: %v", err)
	}
	if songID <= 0 {
		t.Fatalf("expected positive song ID, got %d", songID)
	}

	// 2. Fetch full song
	song, err := songService.GetSong(ctx, int(songID))
	if err != nil {
		t.Fatalf("failed to get song: %v", err)
	}
	if song.Title != "Acoustic Melody" {
		t.Errorf("expected title 'Acoustic Melody', got '%s'", song.Title)
	}
	if len(song.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(song.Tracks))
	}
	if song.Tracks[0].Name != "Lead Guitar" {
		t.Errorf("expected track name 'Lead Guitar', got '%s'", song.Tracks[0].Name)
	}

	// 3. Save full song object with an added track and modifying title
	song.Title = "Acoustic Melody (Extended)"
	song.AddTrack(models.Track{
		InstrumentID: models.InstrumentBassID,
		Name:         "Bass Line",
		DataContent:  `:4 1.3 3.3`,
		DisplayMode:  models.DisplayModeBoth,
	})

	savedSong, err := songService.SaveSong(ctx, song)
	if err != nil {
		t.Fatalf("failed to save song with tracks: %v", err)
	}
	if savedSong.Title != "Acoustic Melody (Extended)" {
		t.Errorf("expected updated title, got '%s'", savedSong.Title)
	}
	if len(savedSong.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(savedSong.Tracks))
	}
	if savedSong.Tracks[1].ID <= 0 {
		t.Errorf("expected new track to receive an assigned DB ID")
	}

	// 4. Test deleting a track during SaveSong
	// Keep only the first track
	savedSong.Tracks = savedSong.Tracks[:1]
	updatedSong, err := songService.SaveSong(ctx, savedSong)
	if err != nil {
		t.Fatalf("failed to update song with deleted track: %v", err)
	}
	if len(updatedSong.Tracks) != 1 {
		t.Errorf("expected 1 track after deletion, got %d", len(updatedSong.Tracks))
	}
}

func TestTrackService(t *testing.T) {
	db, songService, trackService, _, _ := setupTestServices(t)
	defer db.Close()

	ctx := context.Background()

	sID, err := songService.SaveQuickIdea(ctx, 0, "Test Track Song", "")
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	// Add track
	newTrack, err := trackService.AddTrack(ctx, int(sID), "Second Guitar")
	if err != nil {
		t.Fatalf("failed to add track: %v", err)
	}
	if newTrack.Name != "Second Guitar" {
		t.Errorf("expected name 'Second Guitar', got '%s'", newTrack.Name)
	}

	// Add audio track
	audioTrack, err := trackService.AddAudioTrack(ctx, int(sID), "Lead Vocal", "/takes/take1.wav")
	if err != nil {
		t.Fatalf("failed to add audio track: %v", err)
	}
	if !audioTrack.IsAudioTrack() {
		t.Errorf("expected track to be audio track")
	}
	audioPath, ok := audioTrack.GetAudioPath()
	if !ok || audioPath != "/takes/take1.wav" {
		t.Errorf("expected audio path '/takes/take1.wav', got '%s'", audioPath)
	}

	// Update track
	newTrack.Name = "Rhythm Guitar"
	if err := trackService.UpdateTrack(ctx, newTrack); err != nil {
		t.Fatalf("failed to update track: %v", err)
	}

	// Delete track
	if err := trackService.DeleteTrack(ctx, newTrack.ID); err != nil {
		t.Fatalf("failed to delete track: %v", err)
	}
}

func TestAlbumService_Issues(t *testing.T) {
	db, songService, _, albumService, _ := setupTestServices(t)
	defer db.Close()

	ctx := context.Background()

	// Issue #6: Create empty album
	album, err := albumService.CreateAlbum(ctx, "Debut Album", "Rock Band", "First official release")
	if err != nil {
		t.Fatalf("failed to create album: %v", err)
	}
	if album.ID <= 0 {
		t.Fatalf("expected valid album ID, got %d", album.ID)
	}

	// Create song
	sID, err := songService.SaveQuickIdea(ctx, 0, "Track One", "")
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	// Issue #7: Assign song to album
	if err := albumService.AssignSongToAlbum(ctx, int(sID), album.ID); err != nil {
		t.Fatalf("failed to assign song to album: %v", err)
	}

	// Issue #8: List albums and get album songs
	albums, err := albumService.ListAlbums(ctx)
	if err != nil || len(albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albums))
	}

	albumSongs, err := albumService.GetAlbumSongs(ctx, album.ID)
	if err != nil || len(albumSongs) != 1 {
		t.Fatalf("expected 1 song in album, got %d", len(albumSongs))
	}
	if albumSongs[0].Title != "Track One" {
		t.Errorf("expected song title 'Track One', got '%s'", albumSongs[0].Title)
	}

	// Issue #18: Update album
	album.Title = "Debut Album (Deluxe)"
	if err := albumService.UpdateAlbum(ctx, album); err != nil {
		t.Fatalf("failed to update album: %v", err)
	}

	// Issue #17: Remove song from album
	if err := albumService.RemoveSongFromAlbum(ctx, int(sID)); err != nil {
		t.Fatalf("failed to remove song from album: %v", err)
	}

	albumSongsAfter, err := albumService.GetAlbumSongs(ctx, album.ID)
	if err != nil || len(albumSongsAfter) != 0 {
		t.Errorf("expected 0 songs in album after removal, got %d", len(albumSongsAfter))
	}
}

func TestAudioService(t *testing.T) {
	db, songService, _, _, audioService := setupTestServices(t)
	defer db.Close()

	ctx := context.Background()

	sID, err := songService.SaveQuickIdea(ctx, 0, "Vocal Jam", "")
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	// Register global audio version
	av, err := audioService.RegisterAudioVersion(ctx, int(sID), nil, "Rough Idea", "/takes/rough.wav", "Voice memo")
	if err != nil {
		t.Fatalf("failed to register audio version: %v", err)
	}
	if av.ID <= 0 || !av.IsGlobal() {
		t.Errorf("expected valid global audio version")
	}

	takes, err := audioService.GetAudioVersionsForSong(ctx, int(sID))
	if err != nil || len(takes) != 1 {
		t.Fatalf("expected 1 take for song, got %d", len(takes))
	}

	if err := audioService.DeleteAudioVersion(ctx, av.ID); err != nil {
		t.Fatalf("failed to delete audio version: %v", err)
	}
}
