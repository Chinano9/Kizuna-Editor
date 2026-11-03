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
	"sync"

	"github.com/gen2brain/malgo"
)

// AudioPlayer handles low-latency native audio playback using malgo devices.
// It supports thread-safe volume scaling, seeking, and playback state tracking.
type AudioPlayer struct {
	sync.Mutex

	// Playback state
	isPlaying      bool
	pcmData        []byte
	sampleRate     int
	channels       int
	bitsPerSample  int
	playbackOffset int     // current byte index in pcmData
	volume         float64 // gain multiplier in range [0.0, 1.0]

	// malgo handles
	ctx    *malgo.AllocatedContext
	device *malgo.Device
}

// NewAudioPlayer instantiates and returns a thread-safe AudioPlayer.
func NewAudioPlayer() *AudioPlayer {
	return &AudioPlayer{
		volume: 1.0, // Default to full volume
	}
}

// Play loads a WAV file from disk, initializes the native playback device, and begins streaming.
func (p *AudioPlayer) Play(filePath string) error {
	// 1. Stop any active playback session cleanly outside the lock
	p.Stop()

	// 2. Decode the WAV file header and retrieve raw PCM frames
	pcm, sampleRate, channels, bitsPerSample, err := LoadWavFile(filePath)
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
	p.Unlock()

	// 3. Initialize malgo context
	ctx, err := GetSharedContext()
	if err != nil {
		p.Lock()
		p.isPlaying = false
		p.Unlock()
		return fmt.Errorf("failed to get shared audio context: %w", err)
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
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
			return
		}

		bytesNeeded := len(pOutputSample)
		if p.playbackOffset >= len(p.pcmData) {
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

		src := p.pcmData[p.playbackOffset : p.playbackOffset+bytesToCopy]

		// Real-time volume scaling
		if p.volume >= 0.99 {
			copy(pOutputSample[0:bytesToCopy], src)
		} else {
			sampleCount := bytesToCopy / 2
			for i := 0; i < sampleCount; i++ {
				valBytes := src[i*2 : (i+1)*2]
				sampleVal := int16(binary.LittleEndian.Uint16(valBytes))
				scaledVal := float64(sampleVal) * p.volume

				// Safety clipping
				if scaledVal > 32767.0 {
					scaledVal = 32767.0
				} else if scaledVal < -32768.0 {
					scaledVal = -32768.0
				}

				binary.LittleEndian.PutUint16(pOutputSample[i*2:(i+1)*2], uint16(int16(scaledVal)))
			}
		}

		// Silence remainder if audio ended mid-buffer
		if bytesToCopy < bytesNeeded {
			for i := bytesToCopy; i < bytesNeeded; i++ {
				pOutputSample[i] = 0
			}
			p.isPlaying = false
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

	// 7. Start playback thread
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

// Pause toggles play/pause state.
func (p *AudioPlayer) Pause() error {
	p.Lock()
	defer p.Unlock()

	if p.device == nil {
		return fmt.Errorf("no active playback session to pause")
	}

	p.isPlaying = !p.isPlaying
	return nil
}

// Seek shifts the playback progress to the requested second offset.
func (p *AudioPlayer) Seek(seconds float64) error {
	p.Lock()
	defer p.Unlock()

	if len(p.pcmData) == 0 {
		return fmt.Errorf("no audio loaded to seek")
	}

	bytesPerSample := p.bitsPerSample / 8
	blockAlign := p.channels * bytesPerSample

	targetByteOffset := int(seconds * float64(p.sampleRate) * float64(blockAlign))
	targetByteOffset = (targetByteOffset / blockAlign) * blockAlign

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

// GetPosition returns the current playback progress in seconds.
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

// GetDuration returns the total duration of the loaded audio in seconds.
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

// IsPlaying returns the active streaming status.
func (p *AudioPlayer) IsPlaying() bool {
	p.Lock()
	defer p.Unlock()
	return p.isPlaying && p.device != nil
}

// Stop cleanly terminates playback streaming and releases device resources.
func (p *AudioPlayer) Stop() {
	p.Lock()
	dev := p.device
	p.device = nil
	p.isPlaying = false
	p.ctx = nil
	p.Unlock()

	if dev != nil {
		dev.Stop()
		dev.Uninit()
	}
}
