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
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/gen2brain/malgo"
)

// AudioDevice represents a physical audio hardware capture device.
type AudioDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	deviceCacheMu     sync.Mutex
	cachedDevices     []AudioDevice
	cachedDriverName  string
	hasDeviceCacheHit bool
)

// InvalidateDeviceCache clears the cached capture device list when switching audio drivers.
func InvalidateDeviceCache() {
	deviceCacheMu.Lock()
	defer deviceCacheMu.Unlock()
	cachedDevices = nil
	cachedDriverName = ""
	hasDeviceCacheHit = false
}

// GetInputDevices queries malgo to enumerate all available physical microphone/capture devices.
func GetInputDevices() ([]AudioDevice, error) {
	driver := GetCurrentDriver()

	deviceCacheMu.Lock()
	if hasDeviceCacheHit && cachedDriverName == driver {
		out := make([]AudioDevice, len(cachedDevices))
		copy(out, cachedDevices)
		deviceCacheMu.Unlock()
		return out, nil
	}
	deviceCacheMu.Unlock()

	ctx, err := GetSharedContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get shared audio context: %w", err)
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

	deviceCacheMu.Lock()
	cachedDevices = make([]AudioDevice, len(list))
	copy(cachedDevices, list)
	cachedDriverName = driver
	hasDeviceCacheHit = true
	deviceCacheMu.Unlock()

	return list, nil
}

// DecodeDeviceID deserializes a hex-encoded string back into a malgo.DeviceID pointer.
func DecodeDeviceID(idStr string) (*malgo.DeviceID, error) {
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
