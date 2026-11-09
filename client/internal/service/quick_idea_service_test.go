package service

import (
	"context"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"os"
	"path/filepath"
	"testing"
)

func TestQuickIdeaServiceDeleteQuickIdeaRemovesFileAndRecord(t *testing.T) {
	db, _, _, _, _ := setupTestServices(t)
	defer db.Close()

	filePath := filepath.Join(t.TempDir(), "idea.wav")
	if err := os.WriteFile(filePath, []byte("wave data"), 0o644); err != nil {
		t.Fatalf("write WAV: %v", err)
	}
	repo := repository.NewQuickIdeaRepository(db)
	idea := &models.QuickIdea{FilePath: filePath}
	ideaID, err := repo.Create(context.Background(), idea)
	if err != nil {
		t.Fatalf("create quick idea: %v", err)
	}

	svc := NewQuickIdeaService(db, repo)
	if err := svc.DeleteQuickIdea(context.Background(), ideaID); err != nil {
		t.Fatalf("delete quick idea: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected WAV to be deleted, stat error: %v", err)
	}
	if _, err := repo.GetByID(context.Background(), nil, ideaID); err == nil {
		t.Fatal("expected quick idea record to be deleted")
	}
	entries, err := os.ReadDir(filepath.Dir(filePath))
	if err != nil {
		t.Fatalf("read WAV directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected staging file cleanup, found %d files", len(entries))
	}
}

func TestQuickIdeaServiceDeleteQuickIdeaRemovesRecordWhenFileIsMissing(t *testing.T) {
	db, _, _, _, _ := setupTestServices(t)
	defer db.Close()

	repo := repository.NewQuickIdeaRepository(db)
	idea := &models.QuickIdea{FilePath: filepath.Join(t.TempDir(), "missing.wav")}
	ideaID, err := repo.Create(context.Background(), idea)
	if err != nil {
		t.Fatalf("create quick idea: %v", err)
	}

	if err := NewQuickIdeaService(db, repo).DeleteQuickIdea(context.Background(), ideaID); err != nil {
		t.Fatalf("delete quick idea with missing WAV: %v", err)
	}
	if _, err := repo.GetByID(context.Background(), nil, ideaID); err == nil {
		t.Fatal("expected stale quick idea record to be deleted")
	}
}

func TestQuickIdeaServiceDeleteQuickIdeaRestoresFileWhenDatabaseDeleteFails(t *testing.T) {
	db, _, _, _, _ := setupTestServices(t)
	defer db.Close()

	filePath := filepath.Join(t.TempDir(), "idea.wav")
	originalContents := []byte("wave data")
	if err := os.WriteFile(filePath, originalContents, 0o644); err != nil {
		t.Fatalf("write WAV: %v", err)
	}
	repo := repository.NewQuickIdeaRepository(db)
	idea := &models.QuickIdea{FilePath: filePath}
	ideaID, err := repo.Create(context.Background(), idea)
	if err != nil {
		t.Fatalf("create quick idea: %v", err)
	}

	failingRepo := failingQuickIdeaDeleteRepository{QuickIdeaRepository: repo}
	err = NewQuickIdeaService(db, failingRepo).DeleteQuickIdea(context.Background(), ideaID)
	if err == nil {
		t.Fatal("expected delete failure")
	}
	contents, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected original WAV to be restored: %v", err)
	}
	if string(contents) != string(originalContents) {
		t.Fatalf("unexpected restored WAV contents %q", contents)
	}
	if _, err := repo.GetByID(context.Background(), nil, ideaID); err != nil {
		t.Fatalf("expected quick idea record to remain: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(filePath))
	if err != nil {
		t.Fatalf("read WAV directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected no staging file after rollback, found %d files", len(entries))
	}
}

func TestQuickIdeaServiceDeleteQuickIdeaRejectsNonRegularAudioPath(t *testing.T) {
	db, _, _, _, _ := setupTestServices(t)
	defer db.Close()

	repo := repository.NewQuickIdeaRepository(db)
	idea := &models.QuickIdea{FilePath: t.TempDir()}
	ideaID, err := repo.Create(context.Background(), idea)
	if err != nil {
		t.Fatalf("create quick idea: %v", err)
	}

	err = NewQuickIdeaService(db, repo).DeleteQuickIdea(context.Background(), ideaID)
	if err == nil {
		t.Fatal("expected non-regular audio path to fail")
	}
	if _, err := repo.GetByID(context.Background(), nil, ideaID); err != nil {
		t.Fatalf("expected idea record to remain after invalid path: %v", err)
	}
}
