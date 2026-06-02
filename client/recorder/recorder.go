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
	"encoding/hex"
	"fmt"
	"math"
	"runtime"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// Constants defining our audio capture configuration.
// Mono channel is chosen for vocal/instrument recording to optimize DSP, latency, and file size.
const (
	DefaultSampleRate    = 44100
	DefaultChannels      = 1
	DefaultBitsPerSample = 16 // S16 Format = 16-bit signed integer
)

var (
	sharedCtx         *malgo.AllocatedContext
	sharedCtxMu       sync.Mutex
	currentDriverName = "WASAPI" // Default driver name
	ctxErr            error
)

func init() {
	if runtime.GOOS == "darwin" {
		currentDriverName = "CoreAudio"
	} else if runtime.GOOS == "linux" {
		currentDriverName = "ALSA"
	}
}

// GetSharedContext returns a lazily-initialized singleton malgo context shared across playback/recording.
func GetSharedContext() (*malgo.AllocatedContext, error) {
	sharedCtxMu.Lock()
	defer sharedCtxMu.Unlock()

	if sharedCtx != nil {
		return sharedCtx, nil
	}

	backend := mapDriverNameToBackend(currentDriverName)
	sharedCtx, ctxErr = malgo.InitContext([]malgo.Backend{backend}, malgo.ContextConfig{}, nil)
	return sharedCtx, ctxErr
}

// TeardownSharedContext cleanly uninitializes and releases the shared malgo context during app shutdown.
func TeardownSharedContext() {
	sharedCtxMu.Lock()
	defer sharedCtxMu.Unlock()

	if sharedCtx != nil {
		_ = sharedCtx.Uninit()
		sharedCtx.Free()
		sharedCtx = nil
	}
}

// GetAvailableDrivers returns a list of audio drivers appropriate for the current OS.
func GetAvailableDrivers() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"WASAPI"}
	case "darwin":
		return []string{"CoreAudio"}
	case "linux":
		return []string{"ALSA", "PulseAudio", "JACK"}
	default:
		return []string{"WASAPI"}
	}
}

// GetCurrentDriver returns the name of the currently active audio driver.
func GetCurrentDriver() string {
	sharedCtxMu.Lock()
	defer sharedCtxMu.Unlock()
	return currentDriverName
}

// SetAudioDriver reinitializes the shared context with the newly selected audio driver backend.
func SetAudioDriver(driverName string) error {
	sharedCtxMu.Lock()
	defer sharedCtxMu.Unlock()

	if driverName == currentDriverName && sharedCtx != nil {
		return nil // Already using this driver
	}

	// 1. Uninitialize old context if it exists
	if sharedCtx != nil {
		_ = sharedCtx.Uninit()
		sharedCtx.Free()
		sharedCtx = nil
	}

	// 2. Set the new driver name
	currentDriverName = driverName

	// 3. Initialize the new context immediately to verify if it works
	backend := mapDriverNameToBackend(driverName)
	var err error
	sharedCtx, err = malgo.InitContext([]malgo.Backend{backend}, malgo.ContextConfig{}, nil)
	if err != nil {
		// Fallback to platform-default if selected backend fails
		fallbackDriver := "WASAPI"
		if runtime.GOOS == "darwin" {
			fallbackDriver = "CoreAudio"
		} else if runtime.GOOS == "linux" {
			fallbackDriver = "ALSA"
		}

		currentDriverName = fallbackDriver
		backendFallback := mapDriverNameToBackend(fallbackDriver)
		sharedCtx, _ = malgo.InitContext([]malgo.Backend{backendFallback}, malgo.ContextConfig{}, nil)

		return fmt.Errorf("failed to initialize driver %s: %w (fallback to %s)", driverName, err, fallbackDriver)
	}

	return nil
}

func mapDriverNameToBackend(name string) malgo.Backend {
	switch name {
	case "WASAPI":
		return malgo.BackendWasapi
	case "DirectSound":
		return malgo.BackendDsound
	case "WinMM":
		return malgo.BackendWinmm
	case "CoreAudio":
		return malgo.BackendCoreaudio
	case "ALSA":
		return malgo.BackendAlsa
	case "PulseAudio":
		return malgo.BackendPulseaudio
	case "JACK":
		return malgo.BackendJack
	default:
		return malgo.BackendNull
	}
}

// AudioDevice represents a physical audio hardware input device.
type AudioDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetInputDevices queries malgo to list all available physical microphone/capture devices.
func GetInputDevices() ([]AudioDevice, error) {
	ctx, err := GetSharedContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get shared context: %w", err)
	}

	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, fmt.Errorf("failed to get capture devices: %w", err)
	}

	list := make([]AudioDevice, 0, len(devices))
	for _, dev := range devices {
		list = append(list, AudioDevice{
			ID:   dev.ID.String(),
			Name: dev.Name(),
		})
	}
	return list, nil
}

// decodeDeviceID deserializes a hex string back into a malgo.DeviceID.
func decodeDeviceID(idStr string) (*malgo.DeviceID, error) {
	if idStr == "" || idStr == "default" {
		return nil, nil
	}
	bytes, err := hex.DecodeString(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid device hex ID: %w", err)
	}

	var malgoID malgo.DeviceID
	copy(malgoID[:], bytes)
	return &malgoID, nil
}

// AudioRecorder manages native audio capture using CGO miniaudio bindings (malgo).
// It handles thread-safe buffering, device state, and real-time audio levels calculation.
type AudioRecorder struct {
	sync.Mutex

	// Recording State
	isRecording  bool
	SongID       int
	TrackName    string
	capturedData []byte

	// malgo specific handles
	ctx    *malgo.AllocatedContext
	device *malgo.Device

	// Dynamic audio format determined during initialization fallback sequence
	openedChannels   int
	openedSampleRate int

	// Audio levels for VU meter
	currentLevel float64 // range [0.0, 1.0]
}

// NewAudioRecorder instantiates and returns a pointer to an AudioRecorder.
func NewAudioRecorder() *AudioRecorder {
	return &AudioRecorder{
		capturedData: make([]byte, 0, DefaultSampleRate*DefaultChannels*2*10), // preallocate 10s of mono 16-bit audio
	}
}

// StartRecording initializes the default or chosen audio input device and starts recording.
// It returns an error if recording is already in progress or if hardware fails.
func (r *AudioRecorder) StartRecording(songID int, trackName string, deviceID string) error {
	r.Lock()
	if r.isRecording {
		r.Unlock()
		return fmt.Errorf("audio recorder is already recording")
	}

	// Decode target capture hardware device ID
	malgoDeviceID, err := decodeDeviceID(deviceID)
	if err != nil {
		r.Unlock()
		return fmt.Errorf("failed to decode device ID: %w", err)
	}

	// Use runtime.Pinner to safely pin the Go-allocated device ID pointer.
	// This is required under Go's strict CGO rules since we pass the pointer
	// to C via deviceConfig.Capture.DeviceID.
	var pinner runtime.Pinner
	if malgoDeviceID != nil {
		pinner.Pin(malgoDeviceID)
		defer pinner.Unpin()
	}

	// 1. Reset state buffers under the lock
	r.SongID = songID
	r.TrackName = trackName
	r.capturedData = r.capturedData[:0]
	r.currentLevel = 0.0
	r.isRecording = true // Set isRecording to true under the lock so no other StartRecording can run
	r.Unlock()           // Release the lock before calling CGO device start to prevent deadlocks!

	// 2. Initialize the miniaudio Context using the shared singleton context
	ctx, err := GetSharedContext()
	if err != nil {
		r.Lock()
		r.isRecording = false
		r.Unlock()
		return fmt.Errorf("failed to get shared malgo context: %w", err)
	}

	// 3. Configure the fallback trial configurations to accept ANY audio device (consumer & professional):
	// - Channels = 0, SampleRate = 0 (Most stable native path: lets WASAPI/CoreAudio open the hardware native mix format directly, avoiding driver-side resampling hangs)
	// - Mono 44.1kHz (Standard preferred mono layout fallback)
	// - Stereo 44.1kHz (Standard consumer USB mic layout fallback)
	// - Stereo 48kHz (Standard professional sound cards / Behringer layout fallback)
	configsToTry := []struct {
		Channels   uint32
		SampleRate uint32
	}{
		{Channels: 0, SampleRate: 0},                                                // Device Hardware Default (Most stable, native)
		{Channels: uint32(DefaultChannels), SampleRate: uint32(DefaultSampleRate)}, // Mono 44.1kHz fallback
		{Channels: 2, SampleRate: 44100},                                            // Stereo 44.1kHz fallback
		{Channels: 2, SampleRate: 48000},                                            // Stereo 48kHz fallback
	}

	var device *malgo.Device
	var lastErr error

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

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	// 5. Try each device configuration in order until one succeeds
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
		return fmt.Errorf("failed to initialize capture device under any supported format: %w", lastErr)
	}

	// 6. Query the actual initialized format, channels, and sample rate from the active device!
	actualChannels := int(device.CaptureChannels())
	actualSampleRate := int(device.SampleRate())

	r.Lock()
	r.ctx = ctx
	r.device = device
	r.openedChannels = actualChannels
	r.openedSampleRate = actualSampleRate
	r.Unlock()

	// 7. Start Capture
	if err := device.Start(); err != nil {
		device.Uninit()
		r.Lock()
		r.device = nil
		r.ctx = nil
		r.isRecording = false
		r.Unlock()
		return fmt.Errorf("failed to start audio device: %w", err)
	}

	return nil
}

// StopRecording stops the audio capture, teardown device contexts, and flushes the PCM buffer.
// Returns the raw PCM byte slice, along with audio format properties.
func (r *AudioRecorder) StopRecording() (pcmData []byte, sampleRate, channels, bitsPerSample int, err error) {
	r.Lock()
	if !r.isRecording {
		r.Unlock()
		return nil, 0, 0, 0, fmt.Errorf("audio recorder is not recording")
	}

	// 1. Mark recording stopped to prevent callback append
	r.isRecording = false

	// 2. Extract device to stop it outside the lock to prevent CGO worker thread deadlock!
	dev := r.device
	r.device = nil
	r.ctx = nil

	// 3. Duplicate the captured data buffer to return it
	pcmOut := make([]byte, len(r.capturedData))
	copy(pcmOut, r.capturedData)
	openedSR := r.openedSampleRate
	openedCh := r.openedChannels

	// Clean up active levels
	r.currentLevel = 0.0
	r.Unlock() // Release mutex BEFORE calling blocking device.Stop() to allow concurrent worker callbacks to finish!

	// 4. Stop and release hardware device outside the lock
	if dev != nil {
		dev.Stop()
		dev.Uninit()
	}

	return pcmOut, openedSR, openedCh, DefaultBitsPerSample, nil
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
