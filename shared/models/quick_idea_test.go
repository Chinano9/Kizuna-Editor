package models

import (
	"errors"
	"testing"
)

func TestNewQuickIdea(t *testing.T) {
	idea, err := NewQuickIdea("  C:/Kizuna/ideas/idea.wav  ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if idea.FilePath != "C:/Kizuna/ideas/idea.wav" {
		t.Errorf("expected trimmed file path, got %q", idea.FilePath)
	}
	if idea.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewQuickIdeaRequiresFilePath(t *testing.T) {
	_, err := NewQuickIdea("  ")
	if !errors.Is(err, ErrEmptyFilePath) {
		t.Fatalf("expected ErrEmptyFilePath, got %v", err)
	}
}
