package models

import (
	"errors"
	"testing"
)

func TestNewAudioVersion(t *testing.T) {
	t.Run("valid global audio take", func(t *testing.T) {
		av, err := NewAudioVersion(1, nil, "Main Chorus Idea", "/audio/song_1_take_1.wav", "First acoustic take")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !av.IsGlobal() {
			t.Errorf("expected take to be global when trackID is nil")
		}
		if av.VersionName != "Main Chorus Idea" {
			t.Errorf("expected version name 'Main Chorus Idea', got %s", av.VersionName)
		}
		if av.CreatedAt.IsZero() {
			t.Errorf("expected CreatedAt timestamp to be set")
		}
	})

	t.Run("valid track-specific take and link/unlink", func(t *testing.T) {
		trackID := 42
		av, err := NewAudioVersion(1, &trackID, "", "/audio/song_1_track_42.wav", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if av.IsGlobal() {
			t.Errorf("expected take not to be global")
		}
		if *av.TrackID != 42 {
			t.Errorf("expected track ID 42, got %d", *av.TrackID)
		}

		av.UnlinkTrack()
		if !av.IsGlobal() {
			t.Errorf("expected take to become global after UnlinkTrack")
		}

		av.LinkToTrack(99)
		if av.IsGlobal() || *av.TrackID != 99 {
			t.Errorf("expected take to be linked to track 99")
		}
	})

	t.Run("invalid song ID returns ErrInvalidSongID", func(t *testing.T) {
		_, err := NewAudioVersion(0, nil, "Take 1", "/path.wav", "")
		if !errors.Is(err, ErrInvalidSongID) {
			t.Fatalf("expected ErrInvalidSongID, got %v", err)
		}
	})

	t.Run("empty file path returns ErrEmptyFilePath", func(t *testing.T) {
		_, err := NewAudioVersion(1, nil, "Take 1", "   ", "")
		if !errors.Is(err, ErrEmptyFilePath) {
			t.Fatalf("expected ErrEmptyFilePath, got %v", err)
		}
	})
}
