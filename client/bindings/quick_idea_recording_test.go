package bindings

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"kizuna/client/internal/audio"
	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/client/internal/repository"
	"kizuna/client/internal/service"
)

type fakeRecorder struct {
	pcmData  []byte
	startErr error
	err      error
	stopped  bool
}

func (r *fakeRecorder) StartRecording(int, string, string) error { return nil }
func (r *fakeRecorder) StartQuickIdeaRecording(string) error     { return r.startErr }
func (r *fakeRecorder) GetAudioLevels() float64                  { return 0 }

func (r *fakeRecorder) StopRecordingWithInfo(expectedQuickIdea *bool) ([]byte, int, int, int, audio.RecordingInfo, error) {
	if expectedQuickIdea == nil || !*expectedQuickIdea {
		return nil, 0, 0, 0, audio.RecordingInfo{}, errors.New("stop request was not for a quick idea")
	}
	r.stopped = true
	if r.err != nil {
		return nil, 0, 0, 0, audio.RecordingInfo{}, r.err
	}
	return r.pcmData, 44100, 1, 16, audio.RecordingInfo{QuickIdea: true}, nil
}

func newQuickIdeaRecordingHandler(t *testing.T, pcmData []byte) (*AudioHandler, *sql.DB, *fakeRecorder) {
	t.Helper()

	db, err := database.Open(database.DefaultConfig(":memory:"))
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migrator.NewMigrator(db).Migrate(); err != nil {
		t.Fatalf("migrate in-memory database: %v", err)
	}

	rec := &fakeRecorder{pcmData: pcmData}
	quickIdeaService := service.NewQuickIdeaService(db, repository.NewQuickIdeaRepository(db))
	return NewAudioHandler(rec, audio.NewAudioPlayer(), nil, quickIdeaService), db, rec
}

func TestStopQuickIdeaRecordingWritesWAVAndRegistersIdea(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	pcmData := []byte{0x01, 0x02, 0x03, 0x04}
	handler, db, rec := newQuickIdeaRecordingHandler(t, pcmData)

	idea, err := handler.StopQuickIdeaRecording()
	if err != nil {
		t.Fatalf("stop quick idea recording: %v", err)
	}
	if !rec.stopped {
		t.Fatal("expected recorder to be stopped")
	}
	if idea.ID <= 0 {
		t.Fatalf("expected positive idea ID, got %d", idea.ID)
	}

	expectedDir := filepath.Join(appData, "Kizuna", "ideas")
	if filepath.Dir(idea.FilePath) != expectedDir {
		t.Errorf("expected WAV under %q, got %q", expectedDir, idea.FilePath)
	}
	wavData, sampleRate, channels, bitsPerSample, err := audio.LoadWavFile(idea.FilePath)
	if err != nil {
		t.Fatalf("load saved WAV: %v", err)
	}
	if !bytes.Equal(wavData, pcmData) || sampleRate != 44100 || channels != 1 || bitsPerSample != 16 {
		t.Errorf("saved WAV does not match captured PCM or expected format")
	}

	var storedPath string
	if err := db.QueryRowContext(context.Background(),
		"SELECT file_path FROM quick_ideas WHERE id = ?", idea.ID,
	).Scan(&storedPath); err != nil {
		t.Fatalf("query registered idea: %v", err)
	}
	if storedPath != idea.FilePath {
		t.Errorf("database path %q differs from returned path %q", storedPath, idea.FilePath)
	}
}

func TestStopQuickIdeaRecordingRejectsEmptyAudio(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	handler, db, rec := newQuickIdeaRecordingHandler(t, nil)

	if _, err := handler.StopQuickIdeaRecording(); err == nil {
		t.Fatal("expected empty audio to be rejected")
	}
	if !rec.stopped {
		t.Fatal("expected empty recording to be stopped")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM quick_ideas").Scan(&count); err != nil {
		t.Fatalf("count registered ideas: %v", err)
	}
	if count != 0 {
		t.Errorf("expected no registered ideas, got %d", count)
	}
	if _, err := os.Stat(filepath.Join(appData, "Kizuna", "ideas")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no ideas directory for empty recording, stat error: %v", err)
	}
}

func TestStopQuickIdeaRecordingRemovesWAVWhenPersistenceFails(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	handler, db, _ := newQuickIdeaRecordingHandler(t, []byte{0x01, 0x02})
	if err := db.Close(); err != nil {
		t.Fatalf("close database before persistence: %v", err)
	}

	if _, err := handler.StopQuickIdeaRecording(); err == nil {
		t.Fatal("expected closed database to cause persistence failure")
	}
	files, err := filepath.Glob(filepath.Join(appData, "Kizuna", "ideas", "*.wav"))
	if err != nil {
		t.Fatalf("list quick idea WAV files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected unregistered WAV to be removed, found %v", files)
	}
}

func TestStopQuickIdeaRecordingReportsWAVWriteFailure(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)
	handler, db, _ := newQuickIdeaRecordingHandler(t, []byte{0x01, 0x02})

	if err := os.WriteFile(filepath.Join(appData, "Kizuna"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create blocking file: %v", err)
	}
	if _, err := handler.StopQuickIdeaRecording(); err == nil {
		t.Fatal("expected WAV write failure")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM quick_ideas").Scan(&count); err != nil {
		t.Fatalf("count registered ideas: %v", err)
	}
	if count != 0 {
		t.Errorf("expected no registered ideas after WAV write failure, got %d", count)
	}
}

func TestStartQuickIdeaRecordingReturnsRecorderError(t *testing.T) {
	handler, _, rec := newQuickIdeaRecordingHandler(t, nil)
	rec.startErr = errors.New("microphone unavailable")

	err := handler.StartQuickIdeaRecording("default")
	if err == nil || err.Error() != rec.startErr.Error() {
		t.Fatalf("expected recorder error %q, got %v", rec.startErr, err)
	}
}

func TestStopQuickIdeaRecordingReturnsRecorderError(t *testing.T) {
	handler, _, rec := newQuickIdeaRecordingHandler(t, nil)
	rec.err = errors.New("capture device stopped unexpectedly")

	if _, err := handler.StopQuickIdeaRecording(); err == nil || err.Error() != rec.err.Error() {
		t.Fatalf("expected recorder error %q, got %v", rec.err, err)
	}
}

func TestGetQuickIdeasAddsWAVDuration(t *testing.T) {
	handler, _, _ := newQuickIdeaRecordingHandler(t, nil)
	wavPath := filepath.Join(t.TempDir(), "idea.wav")
	if err := audio.WriteWavFile(wavPath, make([]byte, 44100*2), 44100, 1, 16); err != nil {
		t.Fatalf("write WAV: %v", err)
	}
	if _, err := handler.quickIdeaService.RegisterQuickIdea(context.Background(), wavPath); err != nil {
		t.Fatalf("register quick idea: %v", err)
	}

	ideas, err := handler.GetQuickIdeas()
	if err != nil {
		t.Fatalf("get quick ideas: %v", err)
	}
	if len(ideas) != 1 {
		t.Fatalf("expected one quick idea, got %d", len(ideas))
	}
	if ideas[0].DurationSeconds == nil || *ideas[0].DurationSeconds != 1 {
		t.Errorf("expected 1 second duration, got %v", ideas[0].DurationSeconds)
	}
}

func TestPlayAudioReportsMissingQuickIdeaFile(t *testing.T) {
	handler, _, _ := newQuickIdeaRecordingHandler(t, nil)
	if err := handler.PlayAudio(filepath.Join(t.TempDir(), "missing.wav")); err == nil {
		t.Fatal("expected missing audio file to return an error")
	}
}
