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

import "errors"

// Common domain errors for validation across models
var (
	ErrEmptyTitle          = errors.New("title cannot be empty")
	ErrEmptyArtist         = errors.New("artist cannot be empty")
	ErrEmptyName           = errors.New("name cannot be empty")
	ErrEmptyFilePath       = errors.New("file path cannot be empty")
	ErrInvalidBPM          = errors.New("BPM must be between 20 and 320")
	ErrInvalidSignature    = errors.New("time signature must be in format N/D (e.g. 4/4, 3/4, 6/8)")
	ErrInvalidDisplayMode  = errors.New("invalid display mode (allowed: BOTH, TAB, NOTATION, SCORE)")
	ErrInvalidClef         = errors.New("invalid clef (allowed: treble, bass, percussion, alto, tenor)")
	ErrInvalidSongID       = errors.New("song ID must be greater than zero")
	ErrInvalidInstrumentID = errors.New("instrument ID must be greater than zero")
)
