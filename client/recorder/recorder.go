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
	"math"
	"sync"

	"github.com/gen2brain/malgo"
)

// Constants defining our audio capture configuration.
// Mono channel is chosen for vocal/instrument recording to optimize DSP, latency, and file size.
const (
	DefaultSampleRate    = 44100
	DefaultChannels      = 1
	DefaultBitsPerSample = 16 // S16 Format = 16-bit signed integer
)

// AudioRecorder manages native audio capture using CGO miniaudio bindings (malgo).
// It handles thread-safe buffering, device state, and real-time audio levels calculation.
type AudioRecorder struct {
	sync.Mutex

	// Recording State
	isRecording  bool
	songID       int
	trackName    string
	capturedData []byte

	// malgo specific handles
	ctx    *malgo.AllocatedContext
	device *malgo.Device

	// Audio levels for VU meter
	currentLevel float64 // range [0.0, 1.0]
}

// NewAudioRecorder instantiates and returns a pointer to an AudioRecorder.
func NewAudioRecorder() *AudioRecorder {
	return &AudioRecorder{
		capturedData: make([]byte, 0, DefaultSampleRate*DefaultChannels*2*10), // preallocate 10s of mono 16-bit audio
	}
}

// StartRecording initializes the default audio input device and starts recording.
// It returns an error if recording is already in progress or if hardware fails.
func (r *AudioRecorder) StartRecording(songID int, trackName string) error {
	r.Lock()
	defer r.Unlock()

	if r.isRecording {
		return fmt.Errorf("audio recorder is already recording")
	}

	// 1. Reset state buffers
	r.songID = songID
	r.trackName = trackName
	r.capturedData = r.capturedData[:0]
	r.currentLevel = 0.0

	// 2. Initialize the miniaudio Context
	// Creating it on-demand ensures microphone descriptors are cleanly freed when idle.
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize malgo context: %w", err)
	}
	r.ctx = ctx

	// 3. Configure the capture device
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = DefaultChannels
	deviceConfig.SampleRate = DefaultSampleRate

	// 4. Implement thread-safe Capture Data Callback
	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		if len(pInputSamples) == 0 {
			return
		}

		r.Lock()
		defer r.Unlock()

		// Safeguard in case callback is fired after calling Stop()
		if !r.isRecording {
			return
		}

		// Append the raw PCM bytes to our buffer
		r.capturedData = append(r.capturedData, pInputSamples...)

		// Calculate Peak Amplitude in real-time for VU visualizer.
		// Since format is FormatS16, each sample is 2 bytes (16-bit signed).
		var maxVal float64
		sampleCount := len(pInputSamples) / 2

		for i := 0; i < sampleCount; i++ {
			valBytes := pInputSamples[i*2 : (i+1)*2]
			val := int16(binary.LittleEndian.Uint16(valBytes))
			absVal := math.Abs(float64(val))
			if absVal > maxVal {
				maxVal = absVal
			}
		}

		// Normalize level to range [0.0, 1.0] (32768.0 is max signed 16-bit int)
		r.currentLevel = maxVal / 32768.0
	}

	// 5. Initialize the Capture Device
	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}
	device, err := malgo.InitDevice(r.ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		r.ctx.Uninit()
		r.ctx.Free()
		r.ctx = nil
		return fmt.Errorf("failed to initialize capture device: %w", err)
	}
	r.device = device

	// 6. Start Capture
	if err := r.device.Start(); err != nil {
		r.device.Uninit()
		r.ctx.Uninit()
		r.ctx.Free()
		r.device = nil
		r.ctx = nil
		return fmt.Errorf("failed to start audio device: %w", err)
	}

	r.isRecording = true
	return nil
}

// StopRecording stops the audio capture, teardown device contexts, and flushes the PCM buffer.
// Returns the raw PCM byte slice, along with audio format properties.
func (r *AudioRecorder) StopRecording() (pcmData []byte, sampleRate, channels, bitsPerSample int, err error) {
	r.Lock()
	defer r.Unlock()

	if !r.isRecording {
		return nil, 0, 0, 0, fmt.Errorf("audio recorder is not recording")
	}

	// 1. Mark recording stopped to prevent callback append
	r.isRecording = false

	// 2. Stop and release hardware device
	if r.device != nil {
		r.device.Stop()
		r.device.Uninit()
		r.device = nil
	}

	// 3. Teardown CGO Context safely to avoid memory leaks
	if r.ctx != nil {
		r.ctx.Uninit()
		r.ctx.Free()
		r.ctx = nil
	}

	// 4. Duplicate the captured data buffer to return it
	pcmOut := make([]byte, len(r.capturedData))
	copy(pcmOut, r.capturedData)

	// Clean up active levels
	r.currentLevel = 0.0

	return pcmOut, DefaultSampleRate, DefaultChannels, DefaultBitsPerSample, nil
}

// GetAudioLevels returns the current real-time normalized audio level [0.0, 1.0].
// It implements a natural logarithmic decay (0.85 multiplier) so the frontend VU meter
// transitions smoothly and organically instead of dropping instantly to zero between levels query.
func (r *AudioRecorder) GetAudioLevels() float64 {
	r.Lock()
	defer r.Unlock()

	if !r.isRecording {
		return 0.0
	}

	level := r.currentLevel
	// Decay level gradually to create a smooth release transition in the UI
	r.currentLevel = r.currentLevel * 0.85

	return level
}

// IsRecording returns the current active recording state.
func (r *AudioRecorder) IsRecording() bool {
	r.Lock()
	defer r.Unlock()
	return r.isRecording
}
