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
	"fmt"
	"runtime"
	"sync"

	"github.com/gen2brain/malgo"
)

// Constants defining standard audio capture and playback configurations.
const (
	DefaultSampleRate    = 44100
	DefaultChannels      = 1
	DefaultBitsPerSample = 16 // 16-bit signed integer (S16)
)

var (
	sharedCtx         *malgo.AllocatedContext
	sharedCtxMu       sync.Mutex
	currentDriverName = "WASAPI"
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

	backend := MapDriverNameToBackend(currentDriverName)
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

// GetAvailableDrivers returns a list of audio drivers appropriate for the current operating system.
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

// GetCurrentDriver returns the name of the currently active audio driver backend.
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

	// 1. Uninitialize previous context if active
	if sharedCtx != nil {
		_ = sharedCtx.Uninit()
		sharedCtx.Free()
		sharedCtx = nil
	}

	// 2. Set new driver name
	currentDriverName = driverName

	// 3. Initialize new context to test validity
	backend := MapDriverNameToBackend(driverName)
	var err error
	sharedCtx, err = malgo.InitContext([]malgo.Backend{backend}, malgo.ContextConfig{}, nil)
	if err != nil {
		// Fallback to platform default driver if selected backend fails
		fallbackDriver := "WASAPI"
		if runtime.GOOS == "darwin" {
			fallbackDriver = "CoreAudio"
		} else if runtime.GOOS == "linux" {
			fallbackDriver = "ALSA"
		}

		currentDriverName = fallbackDriver
		backendFallback := MapDriverNameToBackend(fallbackDriver)
		sharedCtx, _ = malgo.InitContext([]malgo.Backend{backendFallback}, malgo.ContextConfig{}, nil)

		return fmt.Errorf("failed to initialize driver %s: %w (fallback to %s)", driverName, err, fallbackDriver)
	}

	return nil
}

// MapDriverNameToBackend converts a user-friendly driver name into a malgo.Backend enum.
func MapDriverNameToBackend(name string) malgo.Backend {
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
