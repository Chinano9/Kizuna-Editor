/*
    Kizuna Editor - Reactive Debounced AutoSave Store
    Automatically persists song and track edits (lyrics, chords, AlphaTex) to SQLite via Wails
    after a pause in user typing (default: 1000ms) without requiring manual Ctrl+S.
*/
import { writable, get } from "svelte/store";
import type { Song } from "@shared/types/wails";
import { song, songId } from "./activeSongStore.ts";

export type AutoSaveStatus = "idle" | "pending" | "saving" | "saved" | "error";

export interface AutoSaveState {
    status: AutoSaveStatus;
    lastSavedAt: string | null;
    errorMessage: string | null;
    debounceMs: number;
}

export const AUTOSAVE_DEBOUNCE_MS = 1000;

export type SongPersistFn = (songToSave: Song) => Promise<Song>;

async function defaultWailsPersist(songToSave: Song): Promise<Song> {
    if (typeof window !== "undefined") {
        const wailsApp = (window as any).go?.main?.App;
        if (wailsApp && typeof wailsApp.SaveSong === "function") {
            return await wailsApp.SaveSong(songToSave);
        }
        const wailsSongHandler = (window as any).go?.bindings?.SongHandler;
        if (wailsSongHandler && typeof wailsSongHandler.SaveSong === "function") {
            return await wailsSongHandler.SaveSong(songToSave);
        }
    }
    // Fallback for non-Wails test/browser environments
    return songToSave;
}

const initialState: AutoSaveState = {
    status: "idle",
    lastSavedAt: null,
    errorMessage: null,
    debounceMs: AUTOSAVE_DEBOUNCE_MS,
};

const internalStore = writable<AutoSaveState>(initialState);

let debounceTimer: ReturnType<typeof setTimeout> | null = null;
let isSavingInFlight = false;
let hasPendingWhileSaving = false;
let customPersistFn: SongPersistFn | null = null;

function clearTimer() {
    if (debounceTimer !== null) {
        clearTimeout(debounceTimer);
        debounceTimer = null;
    }
}

export const autoSaveStore = {
    subscribe: internalStore.subscribe,

    /**
     * Configure a custom persistence function (useful for unit tests or custom Wails bindings)
     */
    setPersistHandler: (fn: SongPersistFn | null) => {
        customPersistFn = fn;
    },

    /**
     * Schedule a debounced autosave after `delayMs` (defaults to 1000ms).
     * Every keystroke resets the timer so saving only happens after the user pauses typing.
     */
    scheduleSave: (delayMs?: number) => {
        const currentSong = get(song);
        if (!currentSong) return;

        clearTimer();

        const effectiveDelay =
            typeof delayMs === "number" ? delayMs : get(internalStore).debounceMs;

        if (isSavingInFlight) {
            hasPendingWhileSaving = true;
        }

        internalStore.update((s) => ({
            ...s,
            status: "pending",
            errorMessage: null,
        }));

        debounceTimer = setTimeout(() => {
            debounceTimer = null;
            void autoSaveStore.flush();
        }, effectiveDelay);
    },

    /**
     * Immediately persist any current or pending song/track state to SQLite via Go.
     */
    flush: async (): Promise<Song | null> => {
        clearTimer();

        const currentSong = get(song);
        if (!currentSong) {
            internalStore.update((s) => ({ ...s, status: "idle" }));
            return null;
        }

        if (isSavingInFlight) {
            hasPendingWhileSaving = true;
            return currentSong;
        }

        isSavingInFlight = true;
        hasPendingWhileSaving = false;

        internalStore.update((s) => ({
            ...s,
            status: "saving",
            errorMessage: null,
        }));

        const persist = customPersistFn ?? defaultWailsPersist;

        try {
            const savedSong = await persist(currentSong);

            if (savedSong) {
                const liveSong = get(song);
                if (liveSong) {
                    // Synchronize assigned SQLite IDs without clobbering text typed while save was in-flight
                    const idChanged = liveSong.id === 0 && savedSong.id > 0;
                    if (savedSong.id > 0) {
                        liveSong.id = savedSong.id;
                        songId.set(savedSong.id);
                    }
                    if (liveSong.tracks && savedSong.tracks) {
                        for (let i = 0; i < liveSong.tracks.length; i++) {
                            const savedTrack = savedSong.tracks[i];
                            if (savedTrack && savedTrack.id > 0) {
                                liveSong.tracks[i].id = savedTrack.id;
                                liveSong.tracks[i].song_id = savedSong.id;
                            }
                        }
                    }
                    if (idChanged) {
                        song.set(liveSong);
                    }
                }
            }

            const nowIso = new Date().toISOString();
            internalStore.update((s) => ({
                ...s,
                status: hasPendingWhileSaving ? "pending" : "saved",
                lastSavedAt: nowIso,
                errorMessage: null,
            }));

            isSavingInFlight = false;

            if (hasPendingWhileSaving) {
                hasPendingWhileSaving = false;
                autoSaveStore.scheduleSave();
            }

            return savedSong;
        } catch (err) {
            isSavingInFlight = false;
            const msg = err instanceof Error ? err.message : "Error al autoguardar";
            internalStore.update((s) => ({
                ...s,
                status: "error",
                errorMessage: msg,
            }));
            throw err;
        }
    },

    /**
     * Flush only if there are pending debounced changes waiting to be saved.
     */
    flushIfPending: async (): Promise<void> => {
        const state = get(internalStore);
        if (debounceTimer !== null || state.status === "pending") {
            try {
                await autoSaveStore.flush();
            } catch {
                // Handled in store state
            }
        }
    },

    /**
     * Cancel any pending timer and reset autosave status.
     */
    reset: () => {
        clearTimer();
        isSavingInFlight = false;
        hasPendingWhileSaving = false;
        internalStore.set(initialState);
    },
};
