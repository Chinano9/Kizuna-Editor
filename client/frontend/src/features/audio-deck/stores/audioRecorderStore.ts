/*
    Kizuna Editor - Audio Recorder Store (Feature: Audio Deck - Dev 4)
*/
import { writable } from "svelte/store";
import type { AudioDevice } from "@shared/types/wails";

export interface AudioRecorderState {
    isRecording: boolean;
    secondsElapsed: number;
    currentPeakLevel: number; // 0.0 - 1.0
    selectedDriver: string;
    selectedDeviceID: string;
    availableDrivers: string[];
    availableDevices: AudioDevice[];
}

const initialState: AudioRecorderState = {
    isRecording: false,
    secondsElapsed: 0,
    currentPeakLevel: 0,
    selectedDriver: "WASAPI",
    selectedDeviceID: "default",
    availableDrivers: [],
    availableDevices: [],
};

const { subscribe, set, update } = writable<AudioRecorderState>(initialState);

export const audioRecorderStore = {
    subscribe,
    setRecording: (isRecording: boolean) => update((s) => ({ ...s, isRecording })),
    setElapsed: (seconds: number) => update((s) => ({ ...s, secondsElapsed: seconds })),
    setPeakLevel: (level: number) => update((s) => ({ ...s, currentPeakLevel: level })),
    setDriver: (driver: string) => update((s) => ({ ...s, selectedDriver: driver })),
    setDeviceID: (id: string) => update((s) => ({ ...s, selectedDeviceID: id })),
    setAvailableDrivers: (drivers: string[]) =>
        update((s) => ({ ...s, availableDrivers: drivers })),
    setAvailableDevices: (devices: AudioDevice[]) =>
        update((s) => ({ ...s, availableDevices: devices })),
    reset: () => set(initialState),
};
