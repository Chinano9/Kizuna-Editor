/*
    Kizuna Editor - Active Song Store (Feature: Editor - Dev 5)
    Centralized, transactional, immutable state management for the active editing session.
*/
import { writable, derived, get } from "svelte/store";
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
            return {
                ...curr,
                title,
                updated_at: new Date().toISOString(),
            } as Song;
        });
    },
    addTrack: (track: Track) => {
        songInternal.update((curr) => {
            if (!curr) return null;
            const currentTracks = curr.tracks || [];
            return {
                ...curr,
                tracks: [...currentTracks, track],
                updated_at: new Date().toISOString(),
            } as Song;
        });
    },
    updateTrack: (updatedTrack: Track) => {
        songInternal.update((curr) => {
            if (!curr || !curr.tracks) return curr;
            const newTracks = curr.tracks.map((t) =>
                t.id === updatedTrack.id ? updatedTrack : t
            );
            return {
                ...curr,
                tracks: newTracks,
                updated_at: new Date().toISOString(),
            } as Song;
        });
    },
    removeTrack: (index: number) => {
        songInternal.update((curr) => {
            if (!curr || !curr.tracks) return curr;
            const newTracks = curr.tracks.filter((_, idx) => idx !== index);
            return {
                ...curr,
                tracks: newTracks,
                updated_at: new Date().toISOString(),
            } as Song;
        });
    },
    reset: () => {
        songInternal.set(null);
        songId.set(0);
    },
};
