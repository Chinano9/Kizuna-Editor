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

package models

import (
	"regexp"
	"strings"
)

var timeSignatureRegex = regexp.MustCompile(`^(\d+)/(\d+)$`)

// IsValidTimeSignature validates standard musical time signature format like "4/4", "3/4", "6/8", "12/8".
func IsValidTimeSignature(ts string) bool {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return false
	}
	matches := timeSignatureRegex.FindStringSubmatch(ts)
	if len(matches) != 3 {
		return false
	}
	// Denominator must be standard note value (1, 2, 4, 8, 16, 32)
	den := matches[2]
	switch den {
	case "1", "2", "4", "8", "16", "32":
		return true
	default:
		return false
	}
}

// IsValidClef checks if a clef identifier is recognized by the notation engine.
func IsValidClef(clef string) bool {
	switch strings.ToLower(strings.TrimSpace(clef)) {
	case "treble", "bass", "percussion", "alto", "tenor":
		return true
	default:
		return false
	}
}
