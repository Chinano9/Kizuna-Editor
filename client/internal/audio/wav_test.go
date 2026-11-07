package audio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWavReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	wavFile := filepath.Join(tempDir, "test_output.wav")

	// Generate 1 second of dummy PCM 16-bit mono 44.1kHz audio
	sampleRate := 44100
	channels := 1
	bitsPerSample := 16
	pcmData := make([]byte, sampleRate*channels*(bitsPerSample/8))
	for i := range pcmData {
		pcmData[i] = byte(i % 256)
	}

	// 1. Write WAV file
	if err := WriteWavFile(wavFile, pcmData, sampleRate, channels, bitsPerSample); err != nil {
		t.Fatalf("failed to write WAV file: %v", err)
	}

	// Verify file was written
	stat, err := os.Stat(wavFile)
	if err != nil {
		t.Fatalf("failed to stat written WAV file: %v", err)
	}
	expectedSize := int64(44 + len(pcmData))
	if stat.Size() != expectedSize {
		t.Errorf("expected file size %d, got %d", expectedSize, stat.Size())
	}

	// 2. Read WAV file back
	readPCM, readSR, readCh, readBits, err := LoadWavFile(wavFile)
	if err != nil {
		t.Fatalf("failed to load WAV file: %v", err)
	}
	if readSR != sampleRate {
		t.Errorf("expected sample rate %d, got %d", sampleRate, readSR)
	}
	if readCh != channels {
		t.Errorf("expected channels %d, got %d", channels, readCh)
	}
	if readBits != bitsPerSample {
		t.Errorf("expected bits per sample %d, got %d", bitsPerSample, readBits)
	}
	if !bytes.Equal(pcmData, readPCM) {
		t.Errorf("loaded PCM data does not match original written PCM data")
	}
}

func TestCleanFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Solo:Guitar/Take#1?", "Solo_Guitar_Take#1_"},
		{"<Vocal> *Main* | 1", "_Vocal_ _Main_ _ 1"},
		{"Clean Name 123", "Clean Name 123"},
		{"///", "___"},
		{"", "recorded_track"},
	}

	for _, tc := range tests {
		actual := CleanFileName(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanFileName(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestResolveProjectAudioPath(t *testing.T) {
	path, err := ResolveProjectAudioPath(42, "Lead Vocal", 10)
	if err != nil {
		t.Fatalf("failed to resolve project audio path: %v", err)
	}
	if filepath.Ext(path) != ".wav" {
		t.Errorf("expected .wav extension, got %s", filepath.Ext(path))
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}

func TestResolveQuickIdeaAudioPath(t *testing.T) {
	path, err := ResolveQuickIdeaAudioPath()
	if err != nil {
		t.Fatalf("failed to resolve quick idea audio path: %v", err)
	}
	if filepath.Ext(path) != ".wav" {
		t.Errorf("expected .wav extension, got %s", filepath.Ext(path))
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
	if filepath.Base(filepath.Dir(path)) != "ideas" {
		t.Errorf("expected path under ideas directory, got %s", path)
	}
}
