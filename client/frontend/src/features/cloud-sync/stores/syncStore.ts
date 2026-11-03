/*
    Kizuna Editor - Cloud Sync & Auth Store (Feature: Cloud Sync - Dev 1 & 2)
*/
import { writable } from "svelte/store";

export interface SyncState {
    isAuthenticated: boolean;
    userEmail: string | null;
    token: string | null;
    isSyncing: boolean;
    lastSyncedAt: string | null;
    isLockedByOther: boolean;
    lockedBy: string | null;
}

const initialState: SyncState = {
    isAuthenticated: false,
    userEmail: null,
    token: null,
    isSyncing: false,
    lastSyncedAt: null,
    isLockedByOther: false,
    lockedBy: null,
};

const { subscribe, set, update } = writable<SyncState>(initialState);

export const syncStore = {
    subscribe,
    setAuth: (email: string, token: string) =>
        update((s) => ({ ...s, isAuthenticated: true, userEmail: email, token })),
    clearAuth: () => set(initialState),
    setSyncing: (isSyncing: boolean) => update((s) => ({ ...s, isSyncing })),
    setLock: (isLocked: boolean, lockedBy: string | null) =>
        update((s) => ({ ...s, isLockedByOther: isLocked, lockedBy })),
};
