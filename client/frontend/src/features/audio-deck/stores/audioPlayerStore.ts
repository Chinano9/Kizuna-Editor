/*
    Kizuna Editor - Audio Player Store (Feature: Audio Deck - Dev 4)
*/
import { writable } from "svelte/store";

export interface AudioPlayerState {
    isPlaying: boolean;
    isPaused: boolean;
    positionSeconds: number;
    durationSeconds: number;
    volume: number; // 0.0 - 1.0
    activeTakePath: string;
}

const initialState: AudioPlayerState = {
    isPlaying: false,
    isPaused: false,
    positionSeconds: 0,
    durationSeconds: 0,
    volume: 1.0,
    activeTakePath: "",
};

const { subscribe, set, update } = writable<AudioPlayerState>(initialState);

export const audioPlayerStore = {
    subscribe,
    setPlaying: (isPlaying: boolean, isPaused = false) =>
        update((s) => ({ ...s, isPlaying, isPaused })),
    setPosition: (pos: number) => update((s) => ({ ...s, positionSeconds: pos })),
    setDuration: (dur: number) => update((s) => ({ ...s, durationSeconds: dur })),
    setVolume: (vol: number) => update((s) => ({ ...s, volume: vol })),
    setActiveTake: (path: string) => update((s) => ({ ...s, activeTakePath: path })),
    reset: () => set(initialState),
};
