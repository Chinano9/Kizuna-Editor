package models

import (
	"errors"
	"testing"
)

func TestNewTrack(t *testing.T) {
	t.Run("valid track creation", func(t *testing.T) {
		track, err := NewTrack(1, InstrumentElectricGuitarID, "Acoustic Solo", `\title "Intro"`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if track.Name != "Acoustic Solo" {
			t.Errorf("expected name 'Acoustic Solo', got %s", track.Name)
		}
		if track.DisplayMode != DisplayModeBoth {
			t.Errorf("expected default display mode 'BOTH', got %s", track.DisplayMode)
		}
		if track.IsAudioTrack() {
			t.Errorf("expected track not to be audio track")
		}
	})

	t.Run("empty name defaults to New Track", func(t *testing.T) {
		track, err := NewTrack(1, 1, "", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if track.Name != "New Track" {
			t.Errorf("expected default name 'New Track', got %s", track.Name)
		}
	})

	t.Run("invalid instrument ID error", func(t *testing.T) {
		_, err := NewTrack(1, 0, "Guitar", "")
		if !errors.Is(err, ErrInvalidInstrumentID) {
			t.Fatalf("expected ErrInvalidInstrumentID, got %v", err)
		}
	})

	t.Run("invalid song ID error", func(t *testing.T) {
		_, err := NewTrack(-1, 1, "Guitar", "")
		if !errors.Is(err, ErrInvalidSongID) {
			t.Fatalf("expected ErrInvalidSongID, got %v", err)
		}
	})
}

func TestTrackAudioMethods(t *testing.T) {
	t.Run("detects audio track and extracts path", func(t *testing.T) {
		track := Track{
			ID:           1,
			SongID:       1,
			InstrumentID: InstrumentVocalsID,
			Name:         "Main Vocals",
			DataContent:  "audio:/home/user/takes/vocal_take1.wav",
			DisplayMode:  DisplayModeBoth,
		}

		if !track.IsAudioTrack() {
			t.Errorf("expected IsAudioTrack to be true")
		}

		path, ok := track.GetAudioPath()
		if !ok {
			t.Errorf("expected GetAudioPath ok to be true")
		}
		if path != "/home/user/takes/vocal_take1.wav" {
			t.Errorf("expected path '/home/user/takes/vocal_take1.wav', got %s", path)
		}
	})

	t.Run("regular AlphaTex track returns false for audio", func(t *testing.T) {
		track := Track{
			DataContent: `\tempo 120 :4 3.5 5.5 7.5`,
		}
		if track.IsAudioTrack() {
			t.Errorf("expected IsAudioTrack to be false")
		}
		_, ok := track.GetAudioPath()
		if ok {
			t.Errorf("expected GetAudioPath to be false")
		}
	})
}
