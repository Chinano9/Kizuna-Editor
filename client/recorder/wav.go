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

import "kizuna/client/internal/audio"

// WriteWavFile delegates to audio.WriteWavFile.
func WriteWavFile(filePath string, pcmData []byte, sampleRate, channels, bitsPerSample int) error {
	return audio.WriteWavFile(filePath, pcmData, sampleRate, channels, bitsPerSample)
}
