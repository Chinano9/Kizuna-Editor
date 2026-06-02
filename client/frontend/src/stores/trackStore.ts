import { writable } from "svelte/store";

/**
 * Stores the index of the currently active track in the editor.
 */
export const activeTrackIndex = writable<number>(0);
