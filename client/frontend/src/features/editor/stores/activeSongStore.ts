/*
    Kizuna Editor - Active Song Store (Feature: Editor - Dev 5)
    Centralized, transactional, immutable state management for the active editing session.
*/
import { writable, derived } from "svelte/store";
import type { Song, Track } from "@shared/types/wails";

// Internal writable for the complete song object
const songInternal = writable<Song | null>(null);

// Unique identifier for the song in the database (0 = New/Unsaved)
export const songId = writable<number>(0);

// Preference: Enable/Disable automatic bar line injection
export const autoBar = writable<boolean>(true);

// Export reactive store for the full song object
export const song = {
    subscribe: songInternal.subscribe,
    set: (val: Song | null) => {
        songInternal.set(val);
        if (val) {
            songId.set(val.id);
        }
    },
    update: (fn: (curr: Song | null) => Song | null) => {
        songInternal.update((curr) => {
            const next = fn(curr);
            if (next) songId.set(next.id);
            return next;
        });
    },
};

export const songTitle = derived(songInternal, ($song) =>
    $song ? $song.title : "My New Idea"
);

export const tracks = derived(songInternal, ($song) =>
    $song && $song.tracks ? $song.tracks : []
);

/**
 * Encapsulated immutable state modification helpers
 */
export const activeSongStore = {
    subscribe: songInternal.subscribe,
    setSong: (newSong: Song | null) => {
        song.set(newSong);
    },
    setTitle: (title: string) => {
        songInternal.update((curr) => {
            if (!curr) return null;
            curr.title = title;
            curr.updated_at = new Date().toISOString();
            return curr;
        });
    },
    addTrack: (track: Track) => {
        songInternal.update((curr) => {
            if (!curr) return null;
            const currentTracks = curr.tracks || [];
            curr.tracks = [...currentTracks, track];
            curr.updated_at = new Date().toISOString();
            return curr;
        });
    },
    updateTrack: (updatedTrack: Track) => {
        songInternal.update((curr) => {
            if (!curr || !curr.tracks) return curr;
            curr.tracks = curr.tracks.map((t) =>
                t.id === updatedTrack.id ? updatedTrack : t
            );
            curr.updated_at = new Date().toISOString();
            return curr;
        });
    },
    removeTrack: (index: number) => {
        songInternal.update((curr) => {
            if (!curr || !curr.tracks) return curr;
            curr.tracks = curr.tracks.filter((_, idx) => idx !== index);
            curr.updated_at = new Date().toISOString();
            return curr;
        });
    },
    createNewSong: (): Song => {
        const defaultTrack: Track = {
            id: 0,
            song_id: 0,
            instrument_id: 1, // Default to electric guitar
            name: "Track 1",
            data_content: '\\title "New Song"\n.',
            display_mode: "BOTH",
            is_muted: false,
            created_at: new Date().toISOString(),
            convertValues: (a: any) => a,
        };
        const defaultSong: Song = {
            id: 0,
            title: "My New Idea",
            tracks: [defaultTrack],
            bpm: 120,
            time_signature: "4/4",
            key_signature: "",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            convertValues: (a: any) => a,
        };
        songInternal.set(defaultSong);
        songId.set(0);
        return defaultSong;
    },
    reset: () => {
        songInternal.set(null);
        songId.set(0);
    },
};
