package service

import (
	"context"
	"database/sql"
	"fmt"
	"kizuna/client/internal/repository"
	"os"
	"path/filepath"
	"testing"

	"kizuna/shared/models"
)

func TestQuickIdeaConversionServiceConvertToSong(t *testing.T) {
	db, _, _, _, _ := setupTestServices(t)
	defer db.Close()
	appData := filepath.Join(t.TempDir(), "appdata")
	t.Setenv("APPDATA", appData)

	ideaRepo := repository.NewQuickIdeaRepository(db)
	sourcePath := filepath.Join(t.TempDir(), "idea.wav")
	if err := os.WriteFile(sourcePath, []byte("wave data"), 0o644); err != nil {
		t.Fatalf("write source audio: %v", err)
	}
	idea := &models.QuickIdea{FilePath: sourcePath}
	ideaID, err := ideaRepo.Create(context.Background(), idea)
	if err != nil {
		t.Fatalf("create quick idea: %v", err)
	}

	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)
	svc := NewQuickIdeaConversionService(db, songRepo, trackRepo, audioRepo, ideaRepo)
	songID, err := svc.ConvertToSong(context.Background(), ideaID, "  First Song  ")
	if err != nil {
		t.Fatalf("convert quick idea: %v", err)
	}
	if songID <= 0 {
		t.Fatalf("expected positive song ID, got %d", songID)
	}

	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected original idea audio to be moved, stat error: %v", err)
	}
	if _, err := ideaRepo.GetByID(context.Background(), nil, ideaID); err == nil {
		t.Fatal("expected converted idea to be removed from inbox")
	}

	song, err := songRepo.GetByID(context.Background(), songID)
	if err != nil {
		t.Fatalf("get converted song: %v", err)
	}
	if song.Title != "First Song" || song.BPM != 120 || song.TimeSignature != "4/4" {
		t.Fatalf("unexpected converted song metadata: %+v", song)
	}

	tracks, err := trackRepo.GetBySongID(context.Background(), songID)
	if err != nil {
		t.Fatalf("get converted song tracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected default notation and imported audio tracks, got %+v", tracks)
	}
	if tracks[0].DataContent != "\\title \"New Song\"\n." {
		t.Fatalf("unexpected default track content %q", tracks[0].DataContent)
	}
	if !tracks[1].IsAudioTrack() || tracks[1].Name != "Quick Idea Audio" {
		t.Fatalf("expected imported audio track after the default track, got %+v", tracks[1])
	}
	audioPath, ok := tracks[1].GetAudioPath()
	if !ok {
		t.Fatal("imported track does not contain an audio path")
	}
	if filepath.Dir(audioPath) != filepath.Join(appData, "Kizuna", "projects", fmt.Sprint(songID), "audio") {
		t.Fatalf("audio path is outside the project audio directory: %q", audioPath)
	}
	contents, err := os.ReadFile(audioPath)
	if err != nil {
		t.Fatalf("read imported audio: %v", err)
	}
	if string(contents) != "wave data" {
		t.Fatalf("unexpected imported audio contents %q", contents)
	}
	audioVersions, err := repository.NewAudioRepository(db).GetBySongID(context.Background(), songID)
	if err != nil {
		t.Fatalf("get converted song audio versions: %v", err)
	}
	if len(audioVersions) != 1 || audioVersions[0].FilePath != audioPath {
		t.Fatalf("expected imported WAV registered as a song audio version, got %+v", audioVersions)
	}
	if audioVersions[0].TrackID == nil || *audioVersions[0].TrackID != tracks[1].ID {
		t.Fatalf("expected audio version associated with imported track %d, got %+v", tracks[1].ID, audioVersions[0].TrackID)
	}
}

func TestQuickIdeaConversionServiceFailurePreservesIdeaAndCleansProject(t *testing.T) {
	db, _, _, _, _ := setupTestServices(t)
	defer db.Close()
	appData := filepath.Join(t.TempDir(), "appdata")
	t.Setenv("APPDATA", appData)

	baseRepo := repository.NewQuickIdeaRepository(db)
	sourcePath := filepath.Join(t.TempDir(), "idea.wav")
	if err := os.WriteFile(sourcePath, []byte("wave data"), 0o644); err != nil {
		t.Fatalf("write source audio: %v", err)
	}
	ideaID, err := baseRepo.Create(context.Background(), &models.QuickIdea{FilePath: sourcePath})
	if err != nil {
		t.Fatalf("create quick idea: %v", err)
	}

	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	audioRepo := repository.NewAudioRepository(db)
	svc := NewQuickIdeaConversionService(db, songRepo, trackRepo, audioRepo, failingQuickIdeaDeleteRepository{QuickIdeaRepository: baseRepo})
	if _, err := svc.ConvertToSong(context.Background(), ideaID, "Will Fail"); err == nil {
		t.Fatal("expected conversion error")
	}

	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("expected original quick idea audio to remain: %v", err)
	}
	if _, err := baseRepo.GetByID(context.Background(), nil, ideaID); err != nil {
		t.Fatalf("expected quick idea record to remain: %v", err)
	}
	var songCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&songCount); err != nil {
		t.Fatalf("count songs: %v", err)
	}
	if songCount != 0 {
		t.Fatalf("expected rolled-back song, found %d", songCount)
	}
	var audioVersionCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM audio_versions").Scan(&audioVersionCount); err != nil {
		t.Fatalf("count audio versions: %v", err)
	}
	if audioVersionCount != 0 {
		t.Fatalf("expected rolled-back audio version, found %d", audioVersionCount)
	}
	projectsDir := filepath.Join(appData, "Kizuna", "projects")
	if entries, err := os.ReadDir(projectsDir); err == nil && len(entries) != 0 {
		t.Fatalf("expected incomplete project files to be cleaned, found %d entries", len(entries))
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read projects directory: %v", err)
	}
}

type failingQuickIdeaDeleteRepository struct {
	repository.QuickIdeaRepository
}

func (r failingQuickIdeaDeleteRepository) Delete(context.Context, *sql.Tx, int) error {
	return fmt.Errorf("simulated quick idea delete failure")
}
