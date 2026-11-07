package repository

import (
	"context"
	"testing"

	"kizuna/shared/models"
)

func TestQuickIdeaRepositoryCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewQuickIdeaRepository(db)
	idea, err := models.NewQuickIdea("/audio/quick-idea.wav")
	if err != nil {
		t.Fatalf("failed to create quick idea model: %v", err)
	}

	id, err := repo.Create(context.Background(), idea)
	if err != nil {
		t.Fatalf("failed to persist quick idea: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive quick idea ID, got %d", id)
	}

	var filePath string
	if err := db.QueryRow("SELECT file_path FROM quick_ideas WHERE id = ?", id).Scan(&filePath); err != nil {
		t.Fatalf("failed to query persisted quick idea: %v", err)
	}
	if filePath != idea.FilePath {
		t.Errorf("expected file path %q, got %q", idea.FilePath, filePath)
	}
}

func TestQuickIdeaRepositoryRejectsInvalidInput(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewQuickIdeaRepository(db)
	if _, err := repo.Create(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil quick idea")
	}

	idea := &models.QuickIdea{FilePath: " "}
	if _, err := repo.Create(context.Background(), idea); err == nil {
		t.Fatal("expected error for empty quick idea file path")
	}
}
