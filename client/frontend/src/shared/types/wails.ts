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

import { models as WailsModels, audio as WailsAudio } from "@wails/go/models";

// Tipos y clases de dominio principales exportados limpiamente
export const Song = WailsModels.Song;
export type Song = WailsModels.Song;

export const Track = WailsModels.Track;
export type Track = WailsModels.Track;

export const Instrument = WailsModels.Instrument;
export type Instrument = WailsModels.Instrument;

export const AudioVersion = WailsModels.AudioVersion;
export type AudioVersion = WailsModels.AudioVersion;

export const AudioDevice = WailsAudio.AudioDevice;
export type AudioDevice = WailsAudio.AudioDevice;

// Constructores y fábricas auxiliares
export const SongModel = WailsModels.Song;
export const TrackModel = WailsModels.Track;
export const InstrumentModel = WailsModels.Instrument;
export const AudioVersionModel = WailsModels.AudioVersion;
export const AudioDeviceModel = WailsAudio.AudioDevice;
