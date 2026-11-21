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

import "testing"

func TestComputeProjectHash_DeterminismAndSensitivity(t *testing.T) {
	tracks := []Track{
		{
			InstrumentID: InstrumentElectricGuitarID,
			Name:         "Lead Guitar",
			DisplayMode:  DisplayModeBoth,
			IsMuted:      false,
			DataContent:  ":4 5.3 7.3 | 5.2 8.2",
		},
	}
	takesA := []AudioVersion{
		{VersionName: "Take 1", FilePath: "/audio/take1.wav", Notes: "Warmup"},
		{VersionName: "Take 2", FilePath: "/audio/take2.wav", Notes: "Keeper"},
	}
	// Reversed order of audio takes should produce the exact same canonical hash
	takesB := []AudioVersion{
		{VersionName: "Take 2", FilePath: "/audio/take2.wav", Notes: "Keeper"},
		{VersionName: "Take 1", FilePath: "/audio/take1.wav", Notes: "Warmup"},
	}

	hash1 := ComputeProjectHash("Neon Drift", 120, "4/4", "Am", tracks, takesA)
	hash2 := ComputeProjectHash("  Neon Drift  ", 120, "4/4", "Am", tracks, takesB)

	if len(hash1) != 16 {
		t.Fatalf("expected 16-char hex hash, got %q (len=%d)", hash1, len(hash1))
	}
	if hash1 != hash2 {
		t.Fatalf("expected deterministic identical hashes, got %s vs %s", hash1, hash2)
	}

	// 1. Modifying title changes hash
	if h := ComputeProjectHash("Neon Drift v2", 120, "4/4", "Am", tracks, takesA); h == hash1 {
		t.Errorf("expected hash to change when title changes")
	}

	// 2. Modifying BPM changes hash
	if h := ComputeProjectHash("Neon Drift", 128, "4/4", "Am", tracks, takesA); h == hash1 {
		t.Errorf("expected hash to change when BPM changes")
	}

	// 3. Modifying AlphaTex note content changes hash
	modifiedTracks := []Track{
		{
			InstrumentID: InstrumentElectricGuitarID,
			Name:         "Lead Guitar",
			DisplayMode:  DisplayModeBoth,
			IsMuted:      false,
			DataContent:  ":4 5.3 7.3 | 5.2 9.2",
		},
	}
	if h := ComputeProjectHash("Neon Drift", 120, "4/4", "Am", modifiedTracks, takesA); h == hash1 {
		t.Errorf("expected hash to change when track AlphaTex content changes")
	}

	// 4. Recording a new audio take changes hash
	takesWithNewRecording := append([]AudioVersion{}, takesA...)
	takesWithNewRecording = append(takesWithNewRecording, AudioVersion{
		VersionName: "Take 3",
		FilePath:    "/audio/take3.wav",
		Notes:       "Final solo",
	})
	if h := ComputeProjectHash("Neon Drift", 120, "4/4", "Am", tracks, takesWithNewRecording); h == hash1 {
		t.Errorf("expected hash to change when a new audio take is recorded")
	}
}
