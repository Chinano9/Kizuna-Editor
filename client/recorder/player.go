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
	"sync"

	"github.com/gen2brain/malgo"
)

// AudioPlayer handles low-latency native Go audio playback using malgo Playback devices.
// It supports real-time volume scaling, seamless seeking, and play/pause controls.
type AudioPlayer struct {
	sync.Mutex

	// Playback State
	isPlaying      bool
	pcmData        []byte
	sampleRate     int
	channels       int
	bitsPerSample  int
	playbackOffset int     // current byte index in pcmData
	volume         float64 // gain multiplier in range [0.0, 1.0]

	// malgo Handles
	ctx    *malgo.AllocatedContext
	device *malgo.Device
}

// NewAudioPlayer instantiates and returns a pointer to a thread-safe AudioPlayer.
func NewAudioPlayer() *AudioPlayer {
	return &AudioPlayer{
		volume: 1.0, // Default to full volume
	}
}

// Play loads a WAV file from disk, initializes the native playback device, and starts streaming.
// If another file is currently playing, it automatically stops and releases it first.
func (p *AudioPlayer) Play(filePath string) error {
	// 1. Stop any active playback sessions cleanly outside the lock to prevent deadlocks!
	p.Stop()

	// 2. Decode the WAV file header and retrieve raw PCM frames
	pcm, sampleRate, channels, bitsPerSample, err := loadWavFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load WAV file: %w", err)
	}

	p.Lock()
	p.pcmData = pcm
	p.sampleRate = sampleRate
	p.channels = channels
	p.bitsPerSample = bitsPerSample
	p.playbackOffset = 0
	p.isPlaying = true
	p.Unlock() // Release the lock before CGO setup to avoid deadlocks!

	// 3. Initialize malgo context (using shared singleton context)
	ctx, err := GetSharedContext()
	if err != nil {
		p.Lock()
		p.isPlaying = false
		p.Unlock()
		return fmt.Errorf("failed to get shared malgo context: %w", err)
	}

	// 4. Configure playback device
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatS16
	deviceConfig.Playback.Channels = uint32(channels)
	deviceConfig.SampleRate = uint32(sampleRate)

	// 5. Implement thread-safe playback streaming callback
	onPlayFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		p.Lock()
		defer p.Unlock()

		if !p.isPlaying || len(p.pcmData) == 0 {
			// Output silence if paused or no data loaded
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
			return
		}

		bytesNeeded := len(pOutputSample)
		if p.playbackOffset >= len(p.pcmData) {
			// Reached end of audio file; stop playback
			p.isPlaying = false
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
			return
		}

		bytesAvailable := len(p.pcmData) - p.playbackOffset
		bytesToCopy := bytesNeeded
		if bytesToCopy > bytesAvailable {
			bytesToCopy = bytesAvailable
		}

		// Read raw PCM bytes
		src := p.pcmData[p.playbackOffset : p.playbackOffset+bytesToCopy]

		// Apply real-time volume scaling
		if p.volume >= 0.99 {
			copy(pOutputSample[0:bytesToCopy], src)
		} else {
			// FormatS16 represents 2 bytes per sample
			sampleCount := bytesToCopy / 2
			for i := 0; i < sampleCount; i++ {
				valBytes := src[i*2 : (i+1)*2]
				sampleVal := int16(binary.LittleEndian.Uint16(valBytes))

				// Apply volume fader multiplier
				scaledVal := float64(sampleVal) * p.volume

				// Safety clipping to prevent digital clipping/distortion
				if scaledVal > 32767.0 {
					scaledVal = 32767.0
				} else if scaledVal < -32768.0 {
					scaledVal = -32768.0
				}

				binary.LittleEndian.PutUint16(pOutputSample[i*2:(i+1)*2], uint16(int16(scaledVal)))
			}
		}

		// Fill remainder of the buffer with silence if we ran out of PCM frames mid-buffer
		if bytesToCopy < bytesNeeded {
			for i := bytesToCopy; i < bytesNeeded; i++ {
				pOutputSample[i] = 0
			}
			p.isPlaying = false // Mark end of playback
		}

		p.playbackOffset += bytesToCopy
	}

	// 6. Initialize Playback Device
	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onPlayFrames,
	}
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		p.Lock()
		p.isPlaying = false
		p.Unlock()
		return fmt.Errorf("failed to initialize playback device: %w", err)
	}

	p.Lock()
	p.ctx = ctx
	p.device = device
	p.Unlock()

	// 7. Start Playback Device thread
	if err := device.Start(); err != nil {
		device.Uninit()
		p.Lock()
		p.device = nil
		p.ctx = nil
		p.isPlaying = false
		p.Unlock()
		return fmt.Errorf("failed to start playback device: %w", err)
	}

	return nil
}

// Pause pauses or resumes the active audio streaming.
func (p *AudioPlayer) Pause() error {
	p.Lock()
	defer p.Unlock()

	if p.device == nil {
		return fmt.Errorf("no active playback session to pause")
	}

	p.isPlaying = !p.isPlaying
	return nil
}

// Seek shifts the current playback offset to the designated timestamp in seconds.
// It automatically aligns the byte offset to prevent splits in 16-bit block samples.
func (p *AudioPlayer) Seek(seconds float64) error {
	p.Lock()
	defer p.Unlock()

	if len(p.pcmData) == 0 {
		return fmt.Errorf("no audio loaded to seek")
	}

	bytesPerSample := p.bitsPerSample / 8
	blockAlign := p.channels * bytesPerSample

	// Calculate target offset in bytes
	targetByteOffset := int(seconds * float64(p.sampleRate) * float64(blockAlign))

	// Byte-align offset to block boundary
	targetByteOffset = (targetByteOffset / blockAlign) * blockAlign

	// Boundaries clamp
	if targetByteOffset < 0 {
		targetByteOffset = 0
	} else if targetByteOffset > len(p.pcmData) {
		targetByteOffset = len(p.pcmData)
	}

	p.playbackOffset = targetByteOffset
	return nil
}

// SetVolume updates the real-time gain multiplier in the range [0.0, 1.0].
func (p *AudioPlayer) SetVolume(vol float64) error {
	p.Lock()
	defer p.Unlock()

	if vol < 0.0 {
		vol = 0.0
	} else if vol > 1.0 {
		vol = 1.0
	}

	p.volume = vol
	return nil
}

// GetPosition calculates and returns the current playback progress in seconds.
func (p *AudioPlayer) GetPosition() float64 {
	p.Lock()
	defer p.Unlock()

	if len(p.pcmData) == 0 || p.sampleRate == 0 {
		return 0.0
	}

	bytesPerSample := p.bitsPerSample / 8
	blockAlign := p.channels * bytesPerSample

	return float64(p.playbackOffset) / float64(p.sampleRate*blockAlign)
}

// GetDuration returns the total duration of the loaded WAV file in seconds.
func (p *AudioPlayer) GetDuration() float64 {
	p.Lock()
	defer p.Unlock()

	if len(p.pcmData) == 0 || p.sampleRate == 0 {
		return 0.0
	}

	bytesPerSample := p.bitsPerSample / 8
	blockAlign := p.channels * bytesPerSample

	return float64(len(p.pcmData)) / float64(p.sampleRate*blockAlign)
}

// IsPlaying returns the current active play/stream state.
func (p *AudioPlayer) IsPlaying() bool {
	p.Lock()
	defer p.Unlock()
	return p.isPlaying && p.device != nil
}

// Stop terminates the streaming, tears down playback devices and releases context handles.
func (p *AudioPlayer) Stop() {
	p.Lock()
	dev := p.device
	p.device = nil
	p.isPlaying = false
	p.ctx = nil
	p.Unlock() // Unlock BEFORE calling synchronous device.Stop() to allow concurrent play callbacks to exit cleanly!

	if dev != nil {
		dev.Stop()
		dev.Uninit()
	}
}

// loadWavFile opens a WAV file, parses its RIFF/WAVE header, and returns the raw PCM data byte slice.
func loadWavFile(filePath string) (pcmData []byte, sampleRate, channels, bitsPerSample int, err error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("read WAV file: %w", err)
	}

	if len(data) < 44 {
		return nil, 0, 0, 0, fmt.Errorf("file too short to be a valid WAV")
	}

	// 1. Assert RIFF WAVE file descriptors
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, 0, 0, fmt.Errorf("invalid RIFF/WAVE file headers")
	}

	// 2. Assert uncompressed PCM audio format (1)
	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	if audioFormat != 1 {
		return nil, 0, 0, 0, fmt.Errorf("unsupported WAV audio format %d (only raw S16 PCM is supported)", audioFormat)
	}

	channels = int(binary.LittleEndian.Uint16(data[22:24]))
	sampleRate = int(binary.LittleEndian.Uint32(data[24:28]))
	bitsPerSample = int(binary.LittleEndian.Uint16(data[34:36]))

	// 3. Search for the "data" sub-chunk marker starting from byte 36
	// (Required since DAWs often prepend "LIST" or metadata chunks before raw PCM blocks)
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

	// Bound safeguard in case written size in header is corrupted
	if pcmStart+int(subChunk2Size) > len(data) {
		subChunk2Size = uint32(len(data) - pcmStart)
	}

	pcmData = data[pcmStart : pcmStart+int(subChunk2Size)]
	return pcmData, sampleRate, channels, bitsPerSample, nil
}
