package models

import (
	"errors"
	"testing"
)

func TestNewInstrument(t *testing.T) {
	t.Run("valid instrument creation", func(t *testing.T) {
		inst, err := NewInstrument(1, "Electric Guitar", "String", "treble")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if inst.Name != "Electric Guitar" {
			t.Errorf("expected name 'Electric Guitar', got %s", inst.Name)
		}
		if inst.DefaultClef != "treble" {
			t.Errorf("expected default clef 'treble', got %s", inst.DefaultClef)
		}
	})

	t.Run("empty clef defaults to treble", func(t *testing.T) {
		inst, err := NewInstrument(2, "Bass Guitar", "String", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if inst.DefaultClef != "treble" {
			t.Errorf("expected clef to default to 'treble', got %s", inst.DefaultClef)
		}
	})

	t.Run("empty name returns ErrEmptyName", func(t *testing.T) {
		_, err := NewInstrument(3, "", "Keys", "treble")
		if !errors.Is(err, ErrEmptyName) {
			t.Fatalf("expected ErrEmptyName, got %v", err)
		}
	})

	t.Run("invalid clef returns ErrInvalidClef", func(t *testing.T) {
		_, err := NewInstrument(4, "Sax", "Woodwind", "unsupported_clef")
		if !errors.Is(err, ErrInvalidClef) {
			t.Fatalf("expected ErrInvalidClef, got %v", err)
		}
	})
}
