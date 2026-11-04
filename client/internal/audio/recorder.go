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
	"math"
	"runtime"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// AudioRecorder manages native audio capture using miniaudio bindings (malgo).
type AudioRecorder struct {
	sync.Mutex

	isRecording  bool
	SongID       int
	TrackName    string
	capturedData []byte

	ctx    *malgo.AllocatedContext
	device *malgo.Device

	openedChannels   int
	openedSampleRate int
	currentLevel     float64 // normalized [0.0, 1.0]
}

// NewAudioRecorder instantiates and returns a thread-safe AudioRecorder.
func NewAudioRecorder() *AudioRecorder {
	return &AudioRecorder{
		capturedData: make([]byte, 0, DefaultSampleRate*DefaultChannels*2*10),
	}
}

// StartRecording initializes capture on the designated hardware device.
func (r *AudioRecorder) StartRecording(songID int, trackName string, deviceID string) error {
	r.Lock()
	if r.isRecording {
		r.Unlock()
		return fmt.Errorf("audio recorder is already recording")
	}

	malgoDeviceID, err := DecodeDeviceID(deviceID)
	if err != nil {
		r.Unlock()
		return fmt.Errorf("failed to decode device ID: %w", err)
	}

	var pinner runtime.Pinner
	if malgoDeviceID != nil {
		pinner.Pin(malgoDeviceID)
		defer pinner.Unpin()
	}

	r.SongID = songID
	r.TrackName = trackName
	r.capturedData = r.capturedData[:0]
	r.currentLevel = 0.0
	r.isRecording = true
	r.Unlock()

	ctx, err := GetSharedContext()
	if err != nil {
		r.Lock()
		r.isRecording = false
		r.Unlock()
		return fmt.Errorf("failed to get shared audio context: %w", err)
	}

	configsToTry := []struct {
		Channels   uint32
		SampleRate uint32
	}{
		{Channels: 0, SampleRate: 0},                                                // Hardware Native Default
		{Channels: uint32(DefaultChannels), SampleRate: uint32(DefaultSampleRate)}, // Mono 44.1kHz
		{Channels: 2, SampleRate: 44100},                                            // Stereo 44.1kHz
		{Channels: 2, SampleRate: 48000},                                            // Stereo 48kHz
	}

	var device *malgo.Device
	var lastErr error

	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		if len(pInputSamples) == 0 {
			return
		}

		r.Lock()
		defer r.Unlock()

		if !r.isRecording {
			return
		}

		r.capturedData = append(r.capturedData, pInputSamples...)

		// Peak amplitude calculation for VU visualization
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

		r.currentLevel = maxVal / 32768.0
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	for _, cfg := range configsToTry {
		deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
		deviceConfig.Capture.Format = malgo.FormatS16
		deviceConfig.Capture.Channels = cfg.Channels
		deviceConfig.Capture.DeviceID = unsafe.Pointer(malgoDeviceID)
		deviceConfig.SampleRate = cfg.SampleRate

		device, err = malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
		if err == nil {
			break
		}
		lastErr = err
	}

	if device == nil {
		r.Lock()
		r.isRecording = false
		r.Unlock()
		return fmt.Errorf("failed to initialize capture device: %w", lastErr)
	}

	actualChannels := int(device.CaptureChannels())
	actualSampleRate := int(device.SampleRate())

	r.Lock()
	r.ctx = ctx
	r.device = device
	r.openedChannels = actualChannels
	r.openedSampleRate = actualSampleRate
	r.Unlock()

	if err := device.Start(); err != nil {
		device.Uninit()
		r.Lock()
		r.device = nil
		r.ctx = nil
		r.isRecording = false
		r.Unlock()
		return fmt.Errorf("failed to start audio capture: %w", err)
	}

	return nil
}

// StopRecording halts audio capture and returns the recorded PCM byte buffer.
func (r *AudioRecorder) StopRecording() (pcmData []byte, sampleRate, channels, bitsPerSample int, err error) {
	r.Lock()
	if !r.isRecording {
		r.Unlock()
		return nil, 0, 0, 0, fmt.Errorf("audio recorder is not recording")
	}

	r.isRecording = false
	dev := r.device
	r.device = nil
	r.ctx = nil

	pcmOut := make([]byte, len(r.capturedData))
	copy(pcmOut, r.capturedData)
	openedSR := r.openedSampleRate
	openedCh := r.openedChannels
	r.currentLevel = 0.0
	r.Unlock()

	if dev != nil {
		dev.Stop()
		dev.Uninit()
	}

	return pcmOut, openedSR, openedCh, DefaultBitsPerSample, nil
}

// GetAudioLevels returns the current normalized peak amplitude [0.0, 1.0] with logarithmic release decay.
func (r *AudioRecorder) GetAudioLevels() float64 {
	r.Lock()
	defer r.Unlock()

	if !r.isRecording {
		return 0.0
	}

	level := r.currentLevel
	r.currentLevel = r.currentLevel * 0.85

	return level
}

// IsRecording returns true if recording is in progress.
func (r *AudioRecorder) IsRecording() bool {
	r.Lock()
	defer r.Unlock()
	return r.isRecording
}
