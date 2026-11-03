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

	"github.com/gen2brain/malgo"
)

// AudioDevice represents a physical audio hardware capture device.
type AudioDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetInputDevices queries malgo to enumerate all available physical microphone/capture devices.
func GetInputDevices() ([]AudioDevice, error) {
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
