import { writable } from 'svelte/store';

/**
 * projectStore.ts
 * Global state management for the active editing session.
 * Handles metadata, AlphaTex source content, and editor preferences.
 */

// Unique identifier for the song in the database (0 = New/Unsaved)
export const songId = writable<number>(0);

// Metadata: Project Title
export const songTitle = writable<string>("My New Idea");

// The core content: AlphaTex markup (Lyrics + Chords + Tabs)
export const trackSource = writable<string>(`
\\title "Draft"
\\tempo 120
.
:4 0.6 3.6 5.6
`.trim());

// Preference: Enable/Disable automatic bar line injection in the preview
export const autoBar = writable<boolean>(true);
