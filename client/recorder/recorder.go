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
	"kizuna/client/internal/audio"

	"github.com/gen2brain/malgo"
)

const (
	DefaultSampleRate    = audio.DefaultSampleRate
	DefaultChannels      = audio.DefaultChannels
	DefaultBitsPerSample = audio.DefaultBitsPerSample
)

// AudioDevice is an alias to audio.AudioDevice.
type AudioDevice = audio.AudioDevice

// AudioRecorder is an alias to the core audio.AudioRecorder implementation.
type AudioRecorder = audio.AudioRecorder

// NewAudioRecorder instantiates an AudioRecorder.
func NewAudioRecorder() *AudioRecorder {
	return audio.NewAudioRecorder()
}

// GetSharedContext returns the singleton malgo context.
func GetSharedContext() (*malgo.AllocatedContext, error) {
	return audio.GetSharedContext()
}

// TeardownSharedContext releases the shared malgo context.
func TeardownSharedContext() {
	audio.TeardownSharedContext()
}

// GetAvailableDrivers returns the supported drivers for the active OS.
func GetAvailableDrivers() []string {
	return audio.GetAvailableDrivers()
}

// GetCurrentDriver returns the active driver backend name.
func GetCurrentDriver() string {
	return audio.GetCurrentDriver()
}

// SetAudioDriver reinitializes the audio context with the specified backend.
func SetAudioDriver(driverName string) error {
	return audio.SetAudioDriver(driverName)
}

// GetInputDevices enumerates physical audio capture devices.
func GetInputDevices() ([]AudioDevice, error) {
	return audio.GetInputDevices()
}
