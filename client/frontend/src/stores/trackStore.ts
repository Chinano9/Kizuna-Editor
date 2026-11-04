/*
    Kizuna Editor - Track Store
    Stores the index of the currently active track in the editor with bounds protection.
*/
import { writable } from "svelte/store";

const internalStore = writable<number>(0);

export const activeTrackIndex = {
    subscribe: internalStore.subscribe,
    set: (index: number) => {
        internalStore.set(Math.max(0, index));
    },
    update: (fn: (curr: number) => number) => {
        internalStore.update((curr) => Math.max(0, fn(curr)));
    },
    clamp: (totalTracks: number) => {
        internalStore.update((curr) => {
            if (totalTracks <= 0) return 0;
            return Math.min(Math.max(0, curr), totalTracks - 1);
        });
    },
};
