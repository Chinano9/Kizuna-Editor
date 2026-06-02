/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package recorder

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteWavFile generates a 1-second 440Hz sine wave (standard A4 reference pitch),
// encodes it to WAV, reads it back, and validates that every header byte complies
// exactly with the official RIFF/WAVE file specifications.
func TestWriteWavFile(t *testing.T) {
	// 1. Setup Audio parameters
	sampleRate := DefaultSampleRate
	channels := DefaultChannels
	bitsPerSample := DefaultBitsPerSample
	frequency := 440.0 // 440 Hz
	durationSeconds := 1

	numSamples := sampleRate * durationSeconds
	pcmData := make([]byte, numSamples*2) // 2 bytes per sample (S16 PCM)

	// Generate 440Hz sine wave
	for i := 0; i < numSamples; i++ {
		tVal := float64(i) / float64(sampleRate)
		sineSample := math.Sin(2.0 * math.Pi * frequency * tVal)
		
		// Scale sine range [-1.0, 1.0] to signed 16-bit integer range [-32767, 32767]
		sampleInt16 := int16(sineSample * 32767.0)
		binary.LittleEndian.PutUint16(pcmData[i*2:(i+1)*2], uint16(sampleInt16))
	}

	// 2. Prepare temporary directories
	tempDir, err := os.MkdirTemp("", "kizuna_wav_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "sine_wave_440hz.wav")

	// 3. Write target WAV file
	err = WriteWavFile(testFilePath, pcmData, sampleRate, channels, bitsPerSample)
	if err != nil {
		t.Fatalf("WriteWavFile returned error: %v", err)
	}

	// 4. Verify file was created on disk
	info, err := os.Stat(testFilePath)
	if err != nil {
		t.Fatalf("failed to stat generated WAV file: %v", err)
	}

	expectedSize := int64(44 + len(pcmData))
	if info.Size() != expectedSize {
		t.Errorf("expected file size %d bytes, got %d", expectedSize, info.Size())
	}

	// 5. Read back WAV file and parse header bytes for absolute compliance
	data, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("failed to read written WAV file: %v", err)
	}

	// RIFF descriptor [0:4]
	if string(data[0:4]) != "RIFF" {
		t.Errorf("expected 'RIFF' ChunkID, got %q", data[0:4])
	}

	// Chunk Size [4:8]
	chunkSize := binary.LittleEndian.Uint32(data[4:8])
	if chunkSize != uint32(expectedSize-8) {
		t.Errorf("expected chunkSize %d, got %d", expectedSize-8, chunkSize)
	}

	// WAVE format [8:12]
	if string(data[8:12]) != "WAVE" {
		t.Errorf("expected format 'WAVE', got %q", data[8:12])
	}

	// SubChunk1 ID [12:16]
	if string(data[12:16]) != "fmt " {
		t.Errorf("expected subchunk 'fmt ', got %q", data[12:16])
	}

	// SubChunk1 Size [16:20] (Must be 16 for PCM)
	subChunk1Size := binary.LittleEndian.Uint32(data[16:20])
	if subChunk1Size != 16 {
		t.Errorf("expected SubChunk1Size 16, got %d", subChunk1Size)
	}

	// Audio Format [20:22] (Must be 1 for uncompressed PCM)
	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	if audioFormat != 1 {
		t.Errorf("expected audio format 1 (uncompressed PCM), got %d", audioFormat)
	}

	// Number of channels [22:24]
	numChannels := binary.LittleEndian.Uint16(data[22:24])
	if int(numChannels) != channels {
		t.Errorf("expected channels %d, got %d", channels, numChannels)
	}

	// Sample Rate [24:28]
	sampleRateHeader := binary.LittleEndian.Uint32(data[24:28])
	if int(sampleRateHeader) != sampleRate {
		t.Errorf("expected sample rate %d, got %d", sampleRate, sampleRateHeader)
	}

	// Byte Rate [28:32] (SampleRate * Channels * BitsPerSample/8)
	expectedByteRate := uint32(sampleRate * channels * (bitsPerSample / 8))
	byteRateHeader := binary.LittleEndian.Uint32(data[28:32])
	if byteRateHeader != expectedByteRate {
		t.Errorf("expected byte rate %d, got %d", expectedByteRate, byteRateHeader)
	}

	// Block Align [32:34] (Channels * BitsPerSample/8)
	expectedBlockAlign := uint16(channels * (bitsPerSample / 8))
	blockAlignHeader := binary.LittleEndian.Uint16(data[32:34])
	if blockAlignHeader != expectedBlockAlign {
		t.Errorf("expected block align %d, got %d", expectedBlockAlign, blockAlignHeader)
	}

	// Bits per sample [34:36]
	bitsHeader := binary.LittleEndian.Uint16(data[34:36])
	if int(bitsHeader) != bitsPerSample {
		t.Errorf("expected bits per sample %d, got %d", bitsPerSample, bitsHeader)
	}

	// SubChunk2 ID [36:40]
	if string(data[36:40]) != "data" {
		t.Errorf("expected SubChunk2ID 'data', got %q", data[36:40])
	}

	// SubChunk2 Size [40:44]
	subChunk2Size := binary.LittleEndian.Uint32(data[40:44])
	if int(subChunk2Size) != len(pcmData) {
		t.Errorf("expected SubChunk2Size %d, got %d", len(pcmData), subChunk2Size)
	}
}

func TestGetInputDevices(t *testing.T) {
	devices, err := GetInputDevices()
	if err != nil {
		t.Logf("GetInputDevices returned error (expected if no input hardware or drivers): %v", err)
		return
	}
	t.Logf("Found %d input devices", len(devices))
	for _, dev := range devices {
		t.Logf("Device - ID: %s, Name: %s", dev.ID, dev.Name)
	}
}

