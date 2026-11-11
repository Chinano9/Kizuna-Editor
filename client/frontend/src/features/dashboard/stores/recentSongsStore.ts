/*
    Kizuna Editor - Recent Songs Store (Feature: Dashboard)
*/
import { writable } from "svelte/store";
import type { Song } from "@shared/types/wails";
import { GetRecentSongs } from "@wails/go/main/App";

const { subscribe, set } = writable<Song[]>([]);

let loaded = false;

export const recentSongsStore = {
    subscribe,
    load: async () => {
        if (loaded) return;
        try {
            const songs = await GetRecentSongs();
            set(songs || []);
            loaded = true;
        } catch (err) {
            console.error("Failed to load recent songs:", err);
            set([]);
        }
    },
    refresh: async () => {
        loaded = false;
        await recentSongsStore.load();
    },
};
