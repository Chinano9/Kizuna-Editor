package models

import (
	"errors"
	"testing"
)

func TestNewSong(t *testing.T) {
	t.Run("valid song with defaults", func(t *testing.T) {
		song, err := NewSong("Comfortably Numb", 0, "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if song.Title != "Comfortably Numb" {
			t.Errorf("expected title 'Comfortably Numb', got %s", song.Title)
		}
		if song.BPM != 120 {
			t.Errorf("expected default BPM 120, got %d", song.BPM)
		}
		if song.TimeSignature != "4/4" {
			t.Errorf("expected default time signature '4/4', got %s", song.TimeSignature)
		}
		if song.CreatedAt.IsZero() || song.UpdatedAt.IsZero() {
			t.Errorf("expected non-zero timestamps")
		}
	})

	t.Run("empty title error", func(t *testing.T) {
		_, err := NewSong("", 120, "4/4")
		if !errors.Is(err, ErrEmptyTitle) {
			t.Fatalf("expected ErrEmptyTitle, got %v", err)
		}
	})

	t.Run("invalid BPM error", func(t *testing.T) {
		_, err := NewSong("Fast Track", 500, "4/4")
		if !errors.Is(err, ErrInvalidBPM) {
			t.Fatalf("expected ErrInvalidBPM, got %v", err)
		}
		_, errLow := NewSong("Slow Track", 10, "4/4")
		if !errors.Is(errLow, ErrInvalidBPM) {
			t.Fatalf("expected ErrInvalidBPM, got %v", errLow)
		}
	})

	t.Run("invalid time signature error", func(t *testing.T) {
		_, err := NewSong("Weird Track", 120, "invalid-sig")
		if !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature, got %v", err)
		}
	})
}

func TestSongTrackManagement(t *testing.T) {
	song, err := NewSong("Test Project", 120, "4/4")
	if err != nil {
		t.Fatalf("failed to create song: %v", err)
	}

	track1 := Track{ID: 101, SongID: 1, Name: "Lead Guitar"}
	track2 := Track{ID: 102, SongID: 1, Name: "Bass"}

	song.AddTrack(track1)
	song.AddTrack(track2)

	if len(song.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(song.Tracks))
	}

	found := song.GetTrack(101)
	if found == nil || found.Name != "Lead Guitar" {
		t.Errorf("expected to find track 101 with name 'Lead Guitar'")
	}

	notFound := song.GetTrack(999)
	if notFound != nil {
		t.Errorf("expected nil for non-existent track ID")
	}

	removed := song.RemoveTrack(101)
	if !removed {
		t.Errorf("expected RemoveTrack to return true")
	}
	if len(song.Tracks) != 1 {
		t.Fatalf("expected 1 track remaining, got %d", len(song.Tracks))
	}
	if song.Tracks[0].ID != 102 {
		t.Errorf("expected remaining track to be 102, got %d", song.Tracks[0].ID)
	}
}
