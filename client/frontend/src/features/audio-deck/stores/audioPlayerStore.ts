/*
    Kizuna Editor - Audio Player Store (Feature: Audio Deck - Dev 4 / Issue #25 Sync)
*/
import { writable } from "svelte/store";

export type PlaybackMode = "take" | "synth";

export interface AudioPlayerState {
    isPlaying: boolean;
    isPaused: boolean;
    positionSeconds: number;
    durationSeconds: number;
    volume: number; // 0.0 - 1.0
    activeTakePath: string;
    playbackMode: PlaybackMode;
    currentBar: number;
    currentBeat: number;
    totalBars: number;
    bpm: number;
}

const initialState: AudioPlayerState = {
    isPlaying: false,
    isPaused: false,
    positionSeconds: 0,
    durationSeconds: 0,
    volume: 0.8,
    activeTakePath: "",
    playbackMode: "take",
    currentBar: 0,
    currentBeat: 1,
    totalBars: 1,
    bpm: 120,
};

const { subscribe, set, update } = writable<AudioPlayerState>(initialState);

let seekHandler: ((seconds: number) => void | Promise<void>) | null = null;
let togglePlayHandler: (() => void | Promise<void>) | null = null;

export const audioPlayerStore = {
    subscribe,
    setPlaying: (isPlaying: boolean, isPaused = false) =>
        update((s) => ({ ...s, isPlaying, isPaused })),
    setPosition: (pos: number) => update((s) => ({ ...s, positionSeconds: pos })),
    setDuration: (dur: number) => update((s) => ({ ...s, durationSeconds: dur })),
    setVolume: (vol: number) => update((s) => ({ ...s, volume: vol })),
    setActiveTake: (path: string) => update((s) => ({ ...s, activeTakePath: path })),
    setPlaybackMode: (playbackMode: PlaybackMode) =>
        update((s) => ({ ...s, playbackMode })),
    setScoreSync: (currentBar: number, currentBeat: number, totalBars: number, bpm: number) =>
        update((s) => ({
            ...s,
            currentBar,
            currentBeat,
            totalBars: Math.max(1, totalBars),
            bpm: bpm > 0 ? bpm : s.bpm,
        })),
    registerSeekHandler: (fn: ((seconds: number) => void | Promise<void>) | null) => {
        seekHandler = fn;
    },
    requestSeek: async (seconds: number) => {
        const clamped = Math.max(0, seconds);
        update((s) => ({ ...s, positionSeconds: clamped }));
        if (seekHandler) {
            await seekHandler(clamped);
        }
    },
    registerTogglePlayHandler: (fn: (() => void | Promise<void>) | null) => {
        togglePlayHandler = fn;
    },
    requestTogglePlay: async () => {
        if (togglePlayHandler) {
            await togglePlayHandler();
        }
    },
    reset: () => {
        seekHandler = null;
        togglePlayHandler = null;
        set(initialState);
    },
};
