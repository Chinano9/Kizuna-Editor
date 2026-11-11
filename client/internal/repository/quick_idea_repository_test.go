package repository

import (
	"context"
	"testing"
	"time"

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

func TestQuickIdeaRepositoryListOrdersNewestFirst(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewQuickIdeaRepository(db)
	ctx := context.Background()
	older := &models.QuickIdea{FilePath: "/audio/older.wav", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	newer := &models.QuickIdea{FilePath: "/audio/newer.wav", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}
	olderID, err := repo.Create(ctx, older)
	if err != nil {
		t.Fatalf("create older quick idea: %v", err)
	}
	older.ID = olderID
	newerID, err := repo.Create(ctx, newer)
	if err != nil {
		t.Fatalf("create newer quick idea: %v", err)
	}
	newer.ID = newerID

	ideas, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list quick ideas: %v", err)
	}
	if len(ideas) != 2 {
		t.Fatalf("expected 2 quick ideas, got %d", len(ideas))
	}
	if ideas[0].ID != newer.ID || ideas[1].ID != older.ID {
		t.Errorf("expected newest idea first, got IDs %d then %d", ideas[0].ID, ideas[1].ID)
	}
}

func TestQuickIdeaRepositoryListReturnsNonNilEmptySlice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ideas, err := NewQuickIdeaRepository(db).List(context.Background())
	if err != nil {
		t.Fatalf("list empty quick ideas: %v", err)
	}
	if ideas == nil || len(ideas) != 0 {
		t.Fatalf("expected non-nil empty slice, got %#v", ideas)
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
