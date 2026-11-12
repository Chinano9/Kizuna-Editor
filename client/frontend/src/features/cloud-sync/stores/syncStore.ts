/*
    Kizuna Editor - Cloud Sync & Auth Store (Feature: Cloud Sync - Dev 1 & 2)
    Provides reactive session state, JWT token management, auto-persistence, and cloud backup (Issue #12, #13, #20).
*/
import { writable, get } from "svelte/store";
import type { AuthUser, LoginPayload, RegisterPayload } from "../types/auth.ts";
import type { CloudPushPayload, CloudPushResult } from "../types/sync.ts";
import { authApi, authStorage, DEFAULT_CLOUD_URL } from "../services/authApi.ts";
import { syncApi } from "../services/syncApi.ts";
import type { Song, AudioVersion } from "@shared/types/wails";

export interface CloudProjectMeta {
    cloudProjectId: number;
    version: number;
    lastSyncedAt: string;
}

export interface SyncState {
    isAuthenticated: boolean;
    user: AuthUser | null;
    userEmail: string | null;
    token: string | null;
    cloudUrl: string;
    isSyncing: boolean;
    lastSyncedAt: string | null;
    isLockedByOther: boolean;
    lockedBy: string | null;
    authError: string | null;
    cloudProjectMappings: Record<number, CloudProjectMeta>;
}

const STORAGE_KEYS = {
    MAPPINGS: "kizuna_cloud_project_mappings",
} as const;

function loadStoredMappings(): Record<number, CloudProjectMeta> {
    try {
        if (typeof window !== "undefined" && window.localStorage) {
            const raw = localStorage.getItem(STORAGE_KEYS.MAPPINGS);
            if (raw) return JSON.parse(raw);
        }
    } catch {
        // Ignore storage errors
    }
    return {};
}

function saveStoredMappings(mappings: Record<number, CloudProjectMeta>): void {
    try {
        if (typeof window !== "undefined" && window.localStorage) {
            localStorage.setItem(STORAGE_KEYS.MAPPINGS, JSON.stringify(mappings));
        }
    } catch {
        // Ignore storage errors
    }
}

// Rehydrate state from storage if previously logged in
const persistedToken = authStorage.getToken();
const persistedUser = authStorage.getUser();
const persistedUrl = authStorage.getCloudUrl();
const persistedMappings = loadStoredMappings();

const initialState: SyncState = {
    isAuthenticated: Boolean(persistedToken),
    user: persistedUser,
    userEmail: persistedUser ? persistedUser.email : null,
    token: persistedToken,
    cloudUrl: persistedUrl || DEFAULT_CLOUD_URL,
    isSyncing: false,
    lastSyncedAt: null,
    isLockedByOther: false,
    lockedBy: null,
    authError: null,
    cloudProjectMappings: persistedMappings,
};

const internalStore = writable<SyncState>(initialState);

export const syncStore = {
    subscribe: internalStore.subscribe,

    /**
     * Store active session and persist token + user profile to storage
     */
    setAuth: (userOrEmail: AuthUser | string, token: string) => {
        let user: AuthUser;
        if (typeof userOrEmail === "string") {
            user = { id: "", username: userOrEmail.split("@")[0], email: userOrEmail };
        } else {
            user = userOrEmail;
        }

        authStorage.saveSession(token, user);
        internalStore.update((s) => ({
            ...s,
            isAuthenticated: true,
            user,
            userEmail: user.email,
            token,
            authError: null,
        }));
    },

    /**
     * Terminate active session and purge storage (Issue #20)
     */
    clearAuth: () => {
        authStorage.clearSession();
        internalStore.update((s) => ({
            ...s,
            isAuthenticated: false,
            user: null,
            userEmail: null,
            token: null,
            authError: null,
        }));
    },

    /**
     * Change target Kizuna Cloud server URL
     */
    setCloudUrl: (url: string) => {
        authStorage.setCloudUrl(url);
        internalStore.update((s) => ({ ...s, cloudUrl: url }));
    },

    /**
     * Authenticate with email/password and persist session (Issue #12)
     */
    login: async (credentials: LoginPayload): Promise<void> => {
        const state = get(internalStore);
        try {
            const res = await authApi.login(credentials, state.cloudUrl);
            syncStore.setAuth(res.user, res.token);
        } catch (err) {
            const message = err instanceof Error ? err.message : "Authentication failed";
            internalStore.update((s) => ({ ...s, authError: message }));
            throw err;
        }
    },

    /**
     * Register a new account on Kizuna Cloud and log in immediately (Issue #12)
     */
    register: async (credentials: RegisterPayload): Promise<void> => {
        const state = get(internalStore);
        try {
            const res = await authApi.register(credentials, state.cloudUrl);
            syncStore.setAuth(res.user, res.token);
        } catch (err) {
            const message = err instanceof Error ? err.message : "Registration failed";
            internalStore.update((s) => ({ ...s, authError: message }));
            throw err;
        }
    },

    /**
     * Validate current token against the cloud server; auto-logout on 401 (Issue #20)
     */
    checkSession: async (): Promise<boolean> => {
        const state = get(internalStore);
        if (!state.token) return false;

        try {
            const user = await authApi.getMe(state.token, state.cloudUrl);
            internalStore.update((s) => ({
                ...s,
                isAuthenticated: true,
                user,
                userEmail: user.email,
            }));
            return true;
        } catch (err) {
            // If token expired or unauthorized, clear session
            syncStore.clearAuth();
            return false;
        }
    },

    /**
     * Backup a complete local song (metadata, tracks, audio versions) to Kizuna Cloud (Issue #13).
     * Handles initial backup (project_id = 0) and incremental versioning (project_id = cloudID).
     */
    pushProject: async (
        song: Song,
        audioVersions: AudioVersion[] = []
    ): Promise<CloudPushResult> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token) {
            throw new Error("Debes iniciar sesión en Kizuna Cloud para respaldar tu proyecto.");
        }

        internalStore.update((s) => ({ ...s, isSyncing: true }));

        const mapping = state.cloudProjectMappings[song.id];
        let remoteCloudId = mapping ? mapping.cloudProjectId : 0;
        let baseVersion = mapping ? mapping.version : 0;

        const buildPayload = (projectId: number, version: number): CloudPushPayload => ({
            project_id: projectId,
            title: song.title,
            bpm: song.bpm,
            time_signature: song.time_signature,
            key_signature: song.key_signature,
            base_version: version,
            tracks: song.tracks || [],
            audio_versions: audioVersions,
        });

        try {
            let result: CloudPushResult;
            try {
                result = await syncApi.pushProject(
                    buildPayload(remoteCloudId, baseVersion),
                    state.token,
                    state.cloudUrl
                );
            } catch (err) {
                // Self-healing: if server was restarted and lost the in-memory cloud project,
                // re-push as a new cloud project (project_id = 0).
                const errMsg = err instanceof Error ? err.message : String(err);
                if (remoteCloudId > 0 && errMsg.includes("cloud project not found")) {
                    console.warn(`Cloud project ${remoteCloudId} not found on server. Re-creating as new cloud project...`);
                    result = await syncApi.pushProject(
                        buildPayload(0, 0),
                        state.token,
                        state.cloudUrl
                    );
                } else {
                    throw err;
                }
            }

            const updatedMappings = {
                ...state.cloudProjectMappings,
                [song.id]: {
                    cloudProjectId: result.project_id,
                    version: result.new_version,
                    lastSyncedAt: result.synced_at,
                },
            };

            saveStoredMappings(updatedMappings);

            internalStore.update((s) => ({
                ...s,
                isSyncing: false,
                lastSyncedAt: result.synced_at,
                cloudProjectMappings: updatedMappings,
            }));

            return result;
        } catch (err) {
            internalStore.update((s) => ({ ...s, isSyncing: false }));
            throw err;
        }
    },

    setSyncing: (isSyncing: boolean) =>
        internalStore.update((s) => ({ ...s, isSyncing })),

    setLock: (isLocked: boolean, lockedBy: string | null) =>
        internalStore.update((s) => ({ ...s, isLockedByOther: isLocked, lockedBy })),

    clearError: () =>
        internalStore.update((s) => ({ ...s, authError: null })),
};
