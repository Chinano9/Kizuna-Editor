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
	"fmt"
	"os"
	"path/filepath"
)

// WriteWavFile encodes raw PCM audio data into a standard WAV file.
// It generates a standard 44-byte RIFF/WAVE header and writes it,
// followed by the raw PCM frames, directly to the specified file path.
//
// Parameters:
//   - filePath: Destination absolute file path (e.g. projects/{songID}/audio/{trackName}.wav)
//   - pcmData: Raw PCM bytes captured from the microphone device
//   - sampleRate: Sampling rate in Hz (e.g. 44100)
//   - channels: Number of channels (e.g. 1 for mono, 2 for stereo)
//   - bitsPerSample: Bit depth of each sample (e.g. 16 for malgo.FormatS16)
func WriteWavFile(filePath string, pcmData []byte, sampleRate, channels, bitsPerSample int) error {
	// 1. Ensure the parent directories exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directories: %w", err)
	}

	// 2. Create or overwrite the target file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create wav file: %w", err)
	}
	defer file.Close()

	// 3. Calculate sizes and rates for the WAV header
	subChunk2Size := uint32(len(pcmData))
	chunkSize := 36 + subChunk2Size
	byteRate := uint32(sampleRate * channels * (bitsPerSample / 8))
	blockAlign := uint16(channels * (bitsPerSample / 8))

	// 4. Construct the standard 44-byte WAV header
	// Format specifications:
	// - bytes 0-3: "RIFF" (Big Endian)
	// - bytes 4-7: Size of the entire file minus 8 bytes (Little Endian)
	// - bytes 8-11: "WAVE" (Big Endian)
	// - bytes 12-15: "fmt " (Big Endian)
	// - bytes 16-19: Length of format data above (16 for PCM, Little Endian)
	// - bytes 20-21: Audio format (1 for uncompressed PCM, Little Endian)
	// - bytes 22-23: Channels (1 for mono, 2 for stereo, Little Endian)
	// - bytes 24-27: Sample rate in Hz (Little Endian)
	// - bytes 28-31: Byte rate (SampleRate * NumChannels * BitsPerSample/8, Little Endian)
	// - bytes 32-33: Block align (NumChannels * BitsPerSample/8, Little Endian)
	// - bytes 34-35: Bits per sample (Little Endian)
	// - bytes 36-39: "data" (Big Endian)
	// - bytes 40-43: Size of raw PCM data (Little Endian)
	header := make([]byte, 44)

	// "RIFF" chunk descriptor
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], chunkSize)
	copy(header[8:12], []byte("WAVE"))

	// "fmt " sub-chunk
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16)                     // SubChunk1Size (16 for PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)                      // AudioFormat (1 = PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))       // NumChannels
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))     // SampleRate
	binary.LittleEndian.PutUint32(header[28:32], byteRate)               // ByteRate
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)             // BlockAlign
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))   // BitsPerSample

	// "data" sub-chunk
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], subChunk2Size)

	// 5. Write the header
	if _, err := file.Write(header); err != nil {
		return fmt.Errorf("failed to write wav header: %w", err)
	}

	// 6. Write the raw audio PCM data
	if _, err := file.Write(pcmData); err != nil {
		return fmt.Errorf("failed to write raw audio frames: %w", err)
	}

	return nil
}
