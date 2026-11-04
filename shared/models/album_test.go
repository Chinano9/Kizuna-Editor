package models

import (
	"errors"
	"testing"
)

func TestNewAlbum(t *testing.T) {
	t.Run("valid album creation", func(t *testing.T) {
		album, err := NewAlbum("The Dark Side of the Moon", "Pink Floyd", "Classic progressive rock")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if album.Title != "The Dark Side of the Moon" {
			t.Errorf("expected title 'The Dark Side of the Moon', got %s", album.Title)
		}
		if album.Artist != "Pink Floyd" {
			t.Errorf("expected artist 'Pink Floyd', got %s", album.Artist)
		}
		if album.CreatedAt.IsZero() {
			t.Errorf("expected CreatedAt timestamp to be set")
		}
	})

	t.Run("empty title returns ErrEmptyTitle", func(t *testing.T) {
		_, err := NewAlbum("   ", "Pink Floyd", "")
		if !errors.Is(err, ErrEmptyTitle) {
			t.Fatalf("expected ErrEmptyTitle, got %v", err)
		}
	})
}
