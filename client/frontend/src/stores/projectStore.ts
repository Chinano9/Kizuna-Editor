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

import { writable } from "svelte/store";
import type { main } from "../../../wailsjs/go/models";

/**
 * projectStore.ts
 * Global state management for the active editing session.
 * Handles metadata, AlphaTex source content, and editor preferences.
 */

// Unique identifier for the song in the database (0 = New/Unsaved)
export const songId = writable<number>(0);

// Metadata: Project Title
export const songTitle = writable<string>("My New Idea");

// Preference: Enable/Disable automatic bar line injection in the preview
export const autoBar = writable<boolean>(true);

// Holds the full song object, including all its tracks.
export const song = writable<main.Song | null>(null);
