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

package audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// WriteWavFile encodes raw PCM audio data into a standard WAV file with a 44-byte RIFF/WAVE header.
func WriteWavFile(filePath string, pcmData []byte, sampleRate, channels, bitsPerSample int) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("could not create directories: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create wav file: %w", err)
	}
	defer file.Close()

	subChunk2Size := uint32(len(pcmData))
	chunkSize := 36 + subChunk2Size
	byteRate := uint32(sampleRate * channels * (bitsPerSample / 8))
	blockAlign := uint16(channels * (bitsPerSample / 8))

	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], chunkSize)
	copy(header[8:12], []byte("WAVE"))

	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16)                     // SubChunk1Size (16 for PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)                      // AudioFormat (1 = PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))       // NumChannels
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))     // SampleRate
	binary.LittleEndian.PutUint32(header[28:32], byteRate)               // ByteRate
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)             // BlockAlign
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))   // BitsPerSample

	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], subChunk2Size)

	if _, err := file.Write(header); err != nil {
		return fmt.Errorf("failed to write wav header: %w", err)
	}

	if _, err := file.Write(pcmData); err != nil {
		return fmt.Errorf("failed to write raw audio frames: %w", err)
	}

	return nil
}

// LoadWavFile opens a WAV file, parses its RIFF/WAVE header, and returns the raw PCM data and audio format specs.
func LoadWavFile(filePath string) (pcmData []byte, sampleRate, channels, bitsPerSample int, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("read WAV file: %w", err)
	}

	if len(data) < 44 {
		return nil, 0, 0, 0, fmt.Errorf("file too short to be a valid WAV")
	}

	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, 0, 0, fmt.Errorf("invalid RIFF/WAVE file headers")
	}

	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	if audioFormat != 1 {
		return nil, 0, 0, 0, fmt.Errorf("unsupported WAV audio format %d (only raw S16 PCM is supported)", audioFormat)
	}

	channels = int(binary.LittleEndian.Uint16(data[22:24]))
	sampleRate = int(binary.LittleEndian.Uint32(data[24:28]))
	bitsPerSample = int(binary.LittleEndian.Uint16(data[34:36]))

	// Search for the "data" sub-chunk marker starting from byte 36
	dataMarkerIndex := -1
	for i := 36; i < len(data)-4; i++ {
		if string(data[i:i+4]) == "data" {
			dataMarkerIndex = i
			break
		}
	}

	if dataMarkerIndex == -1 {
		return nil, 0, 0, 0, fmt.Errorf("data sub-chunk marker not found")
	}

	subChunk2Size := binary.LittleEndian.Uint32(data[dataMarkerIndex+4 : dataMarkerIndex+8])
	pcmStart := dataMarkerIndex + 8

	if pcmStart+int(subChunk2Size) > len(data) {
		subChunk2Size = uint32(len(data) - pcmStart)
	}

	pcmData = data[pcmStart : pcmStart+int(subChunk2Size)]
	return pcmData, sampleRate, channels, bitsPerSample, nil
}

// ResolveProjectAudioPath determines the absolute OS-appropriate project path to save recorded .wav files.
func ResolveProjectAudioPath(songID int, trackName string, trackID int) (string, error) {
	safeTrackName := CleanFileName(trackName)
	timestamp := time.Now().Format("20060102_150405")

	var filename string
	if trackID > 0 {
		filename = fmt.Sprintf("track_%d_%s_%s.wav", trackID, safeTrackName, timestamp)
	} else {
		filename = fmt.Sprintf("global_%s_%s.wav", safeTrackName, timestamp)
	}

	baseDir, err := getAudioBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "Kizuna", "projects", fmt.Sprintf("%d", songID), "audio", filename), nil
}

// CleanFileName sanitizes names for filesystem safety by replacing invalid characters with underscores.
func CleanFileName(name string) string {
	var result []rune
	for _, r := range name {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}
	}
	cleaned := string(result)
	if cleaned == "" {
		cleaned = "recorded_track"
	}
	return cleaned
}

func getAudioBaseDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			return appdata, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming"), nil

	case "darwin":
		configDir, err := os.UserConfigDir()
		if err == nil && configDir != "" {
			return configDir, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil

	default: // Linux / Unix
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			return xdg, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}
