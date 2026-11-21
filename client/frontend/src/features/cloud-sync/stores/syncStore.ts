/*
    Kizuna Editor - Cloud Sync & Auth Store (Feature: Cloud Sync - Dev 1 & 2)
    Provides reactive session state, JWT token management, auto-persistence, and cloud backup (Issue #12, #13, #20).
*/
import { writable, get } from "svelte/store";
import type { AuthUser, LoginPayload, RegisterPayload } from "../types/auth.ts";
import type { CloudPushPayload, CloudPushResult, CloudProject, ProjectLock, SyncProgressPayload, SyncRetryPayload } from "../types/sync.ts";
import { authApi, authStorage, DEFAULT_CLOUD_URL, AuthApiError } from "../services/authApi.ts";
import { syncApi } from "../services/syncApi.ts";
import type { Song, AudioVersion } from "@shared/types/wails";
import { activeSongStore } from "../../editor/stores/activeSongStore.ts";

export interface CloudProjectMeta {
    cloudProjectId: number;
    version: number;
    lastSyncedAt: string;
    lastLocalUpdatedAt?: string;
    lastSyncedHash?: string;
}

export type ProjectSyncState =
    | "local_only"
    | "synced"
    | "local_ahead"
    | "cloud_ahead"
    | "conflict";

export interface ProjectSyncComparison {
    status: ProjectSyncState;
    label: string;
    icon: string;
    cloudVersion?: number;
    localVersion?: number;
    localHash?: string;
    cloudHash?: string;
    localUpdatedAt: string;
    cloudUpdatedAt?: string | null;
    lastSyncedAt?: string | null;
}

/**
 * Compares a local SQLite song against its synchronization metadata and remote Kizuna Cloud project
 * using both deterministic SHA-256 content hashes (Git-style) and modification timestamps (Issue #21).
 */
export function computeProjectSyncStatus(
    song: Song,
    mapping?: CloudProjectMeta | null,
    cloudProject?: CloudProject | null
): ProjectSyncComparison {
    const localUpdatedAt = song?.updated_at ? String(song.updated_at) : "";
    const localHash = (song?.content_hash || "").trim();

    if (!mapping || !mapping.cloudProjectId || mapping.cloudProjectId <= 0) {
        return {
            status: "local_only",
            label: "Solo local",
            icon: "💾",
            localUpdatedAt,
            localHash: localHash || undefined,
        };
    }

    const cloudHash = (cloudProject?.content_hash || "").trim();
    const syncedHash = (mapping.lastSyncedHash || "").trim();
    const cloudUpdatedAt = cloudProject?.updated_at ?? null;
    const cloudVersion = cloudProject?.version ?? mapping.version;
    const localVersion = mapping.version;

    // 1. Exact Git-style content hash match between local SQLite and Kizuna Cloud:
    // If both hashes are present and identical, the content is 100% in sync regardless of clock skew or no-op saves.
    if (localHash !== "" && cloudHash !== "" && localHash === cloudHash) {
        return {
            status: "synced",
            label: "Sincronizado",
            icon: "☁️",
            cloudVersion,
            localVersion,
            localHash,
            cloudHash,
            localUpdatedAt,
            cloudUpdatedAt,
            lastSyncedAt: mapping.lastSyncedAt,
        };
    }

    // 2. Determine if local SQLite has unpushed modifications
    let isLocalModified = false;
    if (localHash !== "" && syncedHash !== "") {
        isLocalModified = localHash !== syncedHash;
    } else {
        const localMs = localUpdatedAt ? new Date(localUpdatedAt).getTime() : 0;
        const baselineStr = mapping.lastLocalUpdatedAt || mapping.lastSyncedAt || "";
        const baselineMs = baselineStr ? new Date(baselineStr).getTime() : 0;
        if (localMs > 0 && baselineMs > 0 && !Number.isNaN(localMs) && !Number.isNaN(baselineMs)) {
            isLocalModified = localMs - baselineMs > 1000;
        }
    }

    // 3. Determine if Kizuna Cloud has a newer version than the local baseline
    let isCloudModified = false;
    if (cloudProject) {
        const isVersionAhead =
            typeof cloudProject.version === "number" &&
            typeof mapping.version === "number" &&
            cloudProject.version > mapping.version;

        const isHashAhead =
            cloudHash !== "" && syncedHash !== "" && cloudHash !== syncedHash;

        let isDateAhead = false;
        if (!isHashAhead && (cloudHash === "" || syncedHash === "")) {
            const cloudMs = cloudUpdatedAt ? new Date(cloudUpdatedAt).getTime() : 0;
            const syncedMs = mapping.lastSyncedAt ? new Date(mapping.lastSyncedAt).getTime() : 0;
            if (cloudMs > 0 && syncedMs > 0 && !Number.isNaN(cloudMs) && !Number.isNaN(syncedMs)) {
                isDateAhead = cloudMs - syncedMs > 1000;
            }
        }

        isCloudModified = isVersionAhead || isHashAhead || isDateAhead;
    }

    if (isLocalModified && isCloudModified) {
        return {
            status: "conflict",
            label: "Cambios locales y en nube",
            icon: "⚠️",
            cloudVersion,
            localVersion,
            localHash: localHash || undefined,
            cloudHash: cloudHash || undefined,
            localUpdatedAt,
            cloudUpdatedAt,
            lastSyncedAt: mapping.lastSyncedAt,
        };
    }

    if (isLocalModified) {
        return {
            status: "local_ahead",
            label: "Tiene cambios sin subir",
            icon: "⬆️",
            cloudVersion,
            localVersion,
            localHash: localHash || undefined,
            cloudHash: cloudHash || undefined,
            localUpdatedAt,
            cloudUpdatedAt,
            lastSyncedAt: mapping.lastSyncedAt,
        };
    }

    if (isCloudModified) {
        return {
            status: "cloud_ahead",
            label: "La nube está más actualizada",
            icon: "⬇️",
            cloudVersion,
            localVersion,
            localHash: localHash || undefined,
            cloudHash: cloudHash || undefined,
            localUpdatedAt,
            cloudUpdatedAt,
            lastSyncedAt: mapping.lastSyncedAt,
        };
    }

    return {
        status: "synced",
        label: "Sincronizado",
        icon: "☁️",
        cloudVersion,
        localVersion,
        localHash: localHash || undefined,
        cloudHash: cloudHash || undefined,
        localUpdatedAt,
        cloudUpdatedAt,
        lastSyncedAt: mapping.lastSyncedAt,
    };
}

export interface SyncState {
    isAuthenticated: boolean;
    user: AuthUser | null;
    userEmail: string | null;
    token: string | null;
    cloudUrl: string;
    isSyncing: boolean;
    syncProgress: SyncProgressPayload | null;
    lastSyncedAt: string | null;
    isLockedByOther: boolean;
    lockedBy: string | null;
    hasActiveCheckout: boolean;
    activeLock: ProjectLock | null;
    isCheckingLock: boolean;
    isOffline: boolean;
    authError: string | null;
    cloudProjectMappings: Record<number, CloudProjectMeta>;
}

/**
 * Determines whether a project should be locked in Read-Only mode in the editor (Issues #16, #22):
 * - Purely local projects (cloudId <= 0): Editable (false).
 * - Offline / Unauthenticated fallback (Local-First mode): Editable locally in SQLite (false).
 * - Online cloud-backed projects (cloudId > 0):
 *   - Read-only (true) if locked by another collaborator OR if current user has NOT acquired the checkout lock.
 *   - Editable (false) ONLY when the current user actively holds the checkout lock (hasActiveCheckout === true).
 */
export function isSongReadOnly(state: SyncState, cloudId: number): boolean {
    if (!cloudId || cloudId <= 0) {
        return false;
    }
    if (state.isOffline || !state.isAuthenticated) {
        return false;
    }
    if (state.isLockedByOther) {
        return true;
    }
    return !state.hasActiveCheckout;
}

export function isNetworkConnectivityError(err: unknown): boolean {
    if (err instanceof AuthApiError && err.status === 0) {
        return true;
    }
    const msg = (err instanceof Error ? err.message : String(err)).toLowerCase();
    return (
        msg.includes("unable to connect") ||
        msg.includes("no se pudo conectar") ||
        msg.includes("failed to connect") ||
        msg.includes("network request failed") ||
        msg.includes("failed to fetch") ||
        msg.includes("connection refused") ||
        msg.includes("econnrefused") ||
        msg.includes("networkerror")
    );
}

/**
 * Deduplicates recent songs so that each project appears only once in the dashboard.
 * If multiple local songs point to the same remote cloudProjectId (e.g. from repeated pulls),
 * only the most recently updated local song is kept.
 */
export function deduplicateRecentSongs<T extends { id: number }>(
    songs: T[],
    mappings: Record<number, CloudProjectMeta>
): T[] {
    if (!songs || songs.length === 0) return [];

    const seenCloudIds = new Set<number>();
    const seenLocalIds = new Set<number>();
    const uniqueList: T[] = [];

    for (const songItem of songs) {
        if (!songItem || !songItem.id) continue;

        const mapping = mappings ? mappings[songItem.id] : undefined;
        const cloudId = mapping?.cloudProjectId;

        if (cloudId && cloudId > 0) {
            if (seenCloudIds.has(cloudId)) {
                continue;
            }
            seenCloudIds.add(cloudId);
        }

        if (seenLocalIds.has(songItem.id)) {
            continue;
        }
        seenLocalIds.add(songItem.id);

        uniqueList.push(songItem);
    }

    return uniqueList;
}

export function extractLockedByUsername(err: unknown): string {
    if (err instanceof AuthApiError && err.details && !err.details.includes(" ")) {
        return err.details.trim();
    }
    const msg = err instanceof Error ? err.message : String(err);
    const match = msg.match(/user\s+'([^']+)'\s+holds\s+the\s+lock/i) || msg.match(/bloqueado\s+por\s+([^\s.,]+)/i);
    if (match && match[1]) {
        return match[1];
    }
    return "otro colaborador";
}

function isLockConflictError(err: unknown): boolean {
    if (err instanceof AuthApiError && err.status === 409) {
        const msg = err.message.toLowerCase();
        if (msg.includes("checked out") || msg.includes("lock") || msg.includes("bloqueado")) {
            return true;
        }
    }
    const msg = (err instanceof Error ? err.message : String(err)).toLowerCase();
    return msg.includes("checked out by another user") || msg.includes("holds the lock") || msg.includes("bloqueado por");
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
    syncProgress: null,
    lastSyncedAt: null,
    isLockedByOther: false,
    lockedBy: null,
    hasActiveCheckout: false,
    activeLock: null,
    isCheckingLock: false,
    isOffline: false,
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
            isLockedByOther: false,
            lockedBy: null,
            hasActiveCheckout: false,
            activeLock: null,
            isCheckingLock: false,
            isOffline: false,
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
     * Acquire or renew an exclusive checkout lease on a cloud project before editing (Issue #16).
     * Sets isLockedByOther = true if another collaborator currently holds the lock.
     * Falls back to Offline Local-First mode if the cloud server is unreachable.
     */
    checkoutProject: async (cloudProjectId: number): Promise<ProjectLock | null> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token || cloudProjectId <= 0) {
            return null;
        }

        internalStore.update((s) => ({ ...s, isCheckingLock: true }));

        try {
            let lock: ProjectLock;
            const wailsSync =
                typeof window !== "undefined"
                    ? (window as any).go?.bindings?.SyncHandler || (window as any).go?.main?.App
                    : null;

            if (wailsSync && typeof wailsSync.CheckoutProject === "function") {
                lock = await wailsSync.CheckoutProject(state.cloudUrl, state.token, cloudProjectId);
            } else {
                lock = await syncApi.checkoutProject(cloudProjectId, state.token, state.cloudUrl);
            }

            internalStore.update((s) => ({
                ...s,
                isCheckingLock: false,
                isOffline: false,
                isLockedByOther: false,
                lockedBy: null,
                hasActiveCheckout: true,
                activeLock: lock,
            }));

            return lock;
        } catch (err) {
            if (isLockConflictError(err)) {
                const holder = extractLockedByUsername(err);
                internalStore.update((s) => ({
                    ...s,
                    isCheckingLock: false,
                    isOffline: false,
                    isLockedByOther: true,
                    lockedBy: holder,
                    hasActiveCheckout: false,
                    activeLock: null,
                }));
                throw new Error(
                    `El proyecto está bloqueado por ${holder}. La edición se ha deshabilitado temporalmente.`
                );
            }

            if (isNetworkConnectivityError(err)) {
                internalStore.update((s) => ({
                    ...s,
                    isCheckingLock: false,
                    isOffline: true,
                    isLockedByOther: false,
                    lockedBy: null,
                    hasActiveCheckout: false,
                    activeLock: null,
                }));
                throw new Error(
                    "Sin conexión con Kizuna Cloud: Modo Offline Local-First activado (puedes editar y guardar localmente)."
                );
            }

            internalStore.update((s) => ({ ...s, isCheckingLock: false }));
            throw err;
        }
    },

    /**
     * Query Kizuna Cloud to check whether a project is currently locked by another user (Issue #16).
     * When the project is free in the cloud, leaves hasActiveCheckout = false (Read-Only mode)
     * until the user explicitly requests checkout via checkoutProject().
     * When offline, enables Local-First offline editing mode (isOffline = true).
     */
    verifyProjectLock: async (
        cloudProjectId: number
    ): Promise<{ isLockedByOther: boolean; lockedBy: string | null; isOffline?: boolean }> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token || cloudProjectId <= 0) {
            return { isLockedByOther: false, lockedBy: null, isOffline: false };
        }

        internalStore.update((s) => ({ ...s, isCheckingLock: true }));

        try {
            let activeLock: ProjectLock | null = null;
            const wailsSync =
                typeof window !== "undefined"
                    ? (window as any).go?.bindings?.SyncHandler || (window as any).go?.main?.App
                    : null;

            if (wailsSync && typeof wailsSync.GetProjectLockStatus === "function") {
                activeLock = await wailsSync.GetProjectLockStatus(
                    state.cloudUrl,
                    state.token,
                    cloudProjectId
                );
            } else {
                const project = await syncApi.getProjectDetails(
                    cloudProjectId,
                    state.token,
                    state.cloudUrl
                );
                activeLock = project.active_lock ?? null;
            }

            if (activeLock && activeLock.user_id) {
                const currentUserId = state.user?.id || "";
                const currentUsername = state.user?.username || "";
                const isMine =
                    (currentUserId !== "" && activeLock.user_id === currentUserId) ||
                    (currentUsername !== "" && activeLock.username === currentUsername);

                if (!isMine) {
                    const holder = activeLock.username || "otro colaborador";
                    internalStore.update((s) => ({
                        ...s,
                        isCheckingLock: false,
                        isOffline: false,
                        isLockedByOther: true,
                        lockedBy: holder,
                        hasActiveCheckout: false,
                        activeLock,
                    }));
                    return { isLockedByOther: true, lockedBy: holder, isOffline: false };
                }

                internalStore.update((s) => ({
                    ...s,
                    isCheckingLock: false,
                    isOffline: false,
                    isLockedByOther: false,
                    lockedBy: null,
                    hasActiveCheckout: true,
                    activeLock,
                }));
                return { isLockedByOther: false, lockedBy: null, isOffline: false };
            }

            internalStore.update((s) => ({
                ...s,
                isCheckingLock: false,
                isOffline: false,
                isLockedByOther: false,
                lockedBy: null,
                hasActiveCheckout: false,
                activeLock: null,
            }));
            return { isLockedByOther: false, lockedBy: null, isOffline: false };
        } catch (err) {
            if (isNetworkConnectivityError(err)) {
                internalStore.update((s) => ({
                    ...s,
                    isCheckingLock: false,
                    isOffline: true,
                    isLockedByOther: false,
                    lockedBy: null,
                    hasActiveCheckout: false,
                    activeLock: null,
                }));
                return { isLockedByOther: false, lockedBy: null, isOffline: true };
            }

            internalStore.update((s) => ({ ...s, isCheckingLock: false }));
            throw err;
        }
    },

    /**
     * Explicitly release the checkout lock on a cloud project (Check-in, Issue #22).
     * Immediately transitions the project back to Read-Only mode (hasActiveCheckout = false).
     */
    releaseProjectLock: async (cloudProjectId: number): Promise<void> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token || cloudProjectId <= 0) {
            internalStore.update((s) => ({
                ...s,
                hasActiveCheckout: false,
                activeLock: null,
            }));
            return;
        }

        internalStore.update((s) => ({ ...s, isCheckingLock: true }));

        try {
            const wailsSync =
                typeof window !== "undefined"
                    ? (window as any).go?.bindings?.SyncHandler || (window as any).go?.main?.App
                    : null;

            if (wailsSync && typeof wailsSync.ReleaseProjectLock === "function") {
                await wailsSync.ReleaseProjectLock(state.cloudUrl, state.token, cloudProjectId);
            } else {
                await syncApi.releaseProject(cloudProjectId, state.token, state.cloudUrl);
            }

            internalStore.update((s) => ({
                ...s,
                isCheckingLock: false,
                isOffline: false,
                hasActiveCheckout: false,
                activeLock: null,
                isLockedByOther: false,
                lockedBy: null,
            }));
        } catch (err) {
            internalStore.update((s) => ({ ...s, isCheckingLock: false }));
            throw err;
        }
    },

    /**
     * Reset local lock state when navigating away or switching songs.
     */
    resetLockState: () => {
        internalStore.update((s) => ({
            ...s,
            isLockedByOther: false,
            lockedBy: null,
            hasActiveCheckout: false,
            activeLock: null,
            isCheckingLock: false,
            isOffline: false,
        }));
    },

    setOffline: (isOffline: boolean) => {
        internalStore.update((s) => ({
            ...s,
            isOffline,
            ...(isOffline ? { isLockedByOther: false, hasActiveCheckout: false } : {}),
        }));
    },

    /**
     * Backup a complete local song (metadata, tracks, audio versions) to Kizuna Cloud (Issue #13, #16, #22).
     * Handles initial backup (project_id = 0), pre-push lock validation, offline reconciliation,
     * and automatic lock release after push (transitioning to Read-Only mode).
     */
    pushProject: async (
        song: Song,
        audioVersions: AudioVersion[] = []
    ): Promise<CloudPushResult> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token) {
            throw new Error("Debes iniciar sesión en Kizuna Cloud para respaldar tu proyecto.");
        }

        const mapping = state.cloudProjectMappings[song.id];
        const activeSong = get(activeSongStore);
        let remoteCloudId = mapping
            ? mapping.cloudProjectId
            : activeSong?.cloud_id
            ? Number(activeSong.cloud_id)
            : 0;
        let baseVersion = mapping ? mapping.version : 0;

        // Pre-push lock validation (Issue #16): if this project is already in the cloud:
        if (remoteCloudId > 0) {
            if (state.isLockedByOther) {
                const holder = state.lockedBy || "otro colaborador";
                throw new Error(
                    `No puedes sincronizar: el proyecto está bloqueado por ${holder}.`
                );
            }
        }

        internalStore.update((s) => ({ ...s, isSyncing: true }));

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
            // Check if running in Wails desktop app
            const wailsSync =
                typeof window !== "undefined"
                    ? (window as any).go?.bindings?.SyncHandler || (window as any).go?.main?.App
                    : null;
            if (wailsSync && typeof wailsSync.PushProjectToCloud === "function") {
                if (remoteCloudId > 0 && typeof wailsSync.SetCloudProjectID === "function") {
                    await wailsSync.SetCloudProjectID(song.id, remoteCloudId);
                }
                try {
                    result = await wailsSync.PushProjectToCloud(state.cloudUrl, state.token, song.id, baseVersion);
                } catch (err) {
                    const errMsg = err instanceof Error ? err.message : String(err);
                    if (baseVersion > 0 && errMsg.includes("cloud project not found")) {
                        console.warn("Cloud project not found on server. Re-creating as initial cloud project (version = 0)...");
                        result = await wailsSync.PushProjectToCloud(state.cloudUrl, state.token, song.id, 0);
                    } else {
                        throw err;
                    }
                }
            } else {
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
            }

            const currentState = get(internalStore);
            const resolvedHash =
                (result.content_hash || song.content_hash || "").trim() || undefined;
            const localUpdatedStr = song.updated_at ? String(song.updated_at) : result.synced_at;
            const updatedMappings = {
                ...currentState.cloudProjectMappings,
                [song.id]: {
                    cloudProjectId: result.project_id,
                    version: result.new_version,
                    lastSyncedAt: result.synced_at,
                    lastLocalUpdatedAt: localUpdatedStr,
                    lastSyncedHash: resolvedHash,
                },
            };

            saveStoredMappings(updatedMappings);

            // Update activeSongStore with cloud_id on push
            activeSongStore.setCloudId(result.project_id);

            // Issue #22: Release active checkout lock state upon successful push -> transitions to Read-Only mode
            internalStore.update((s) => ({
                ...s,
                isSyncing: false,
                isOffline: false,
                lastSyncedAt: result.synced_at,
                hasActiveCheckout: false,
                activeLock: null,
                isLockedByOther: false,
                lockedBy: null,
                cloudProjectMappings: updatedMappings,
            }));

            return result;
        } catch (err) {
            if (isLockConflictError(err)) {
                const holder = extractLockedByUsername(err);
                internalStore.update((s) => ({
                    ...s,
                    isSyncing: false,
                    isOffline: false,
                    isLockedByOther: true,
                    lockedBy: holder,
                    hasActiveCheckout: false,
                    activeLock: null,
                }));
                throw new Error(
                    `No puedes sincronizar: el proyecto está bloqueado por ${holder}.`
                );
            }

            if (isNetworkConnectivityError(err)) {
                internalStore.update((s) => ({
                    ...s,
                    isSyncing: false,
                    isOffline: true,
                }));
            } else {
                internalStore.update((s) => ({ ...s, isSyncing: false }));
            }
            throw err;
        }
    },

    /**
     * List user projects available in Kizuna Cloud (Issue #14)
     */
    listCloudProjects: async (): Promise<CloudProject[]> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token) {
            throw new Error("Debes iniciar sesión para consultar tus proyectos en la nube.");
        }
        const wailsSync =
            typeof window !== "undefined"
                ? (window as any).go?.bindings?.SyncHandler || (window as any).go?.main?.App
                : null;
        if (wailsSync && typeof wailsSync.ListCloudProjects === "function") {
            return (await wailsSync.ListCloudProjects(state.cloudUrl, state.token)) || [];
        }
        return await syncApi.listProjects(state.token, state.cloudUrl);
    },

    /**
     * Pull project from Kizuna Cloud and import into local SQLite (Issue #14).
     * Opens in Read-Only mode (hasActiveCheckout = false) until user explicitly checks out.
     */
    pullAndImportProject: async (cloudProjectId: number): Promise<number> => {
        const state = get(internalStore);
        if (!state.isAuthenticated || !state.token) {
            throw new Error("Debes iniciar sesión para descargar proyectos de la nube.");
        }

        internalStore.update((s) => ({ ...s, isSyncing: true }));

        try {
            let localSongId: number;
            let pulledVersion = 1;
            let pulledHash: string | undefined = undefined;
            let pulledUpdatedAt: string = new Date().toISOString();

            // Check if running in Wails desktop app
            const wailsSync = typeof window !== "undefined" ? (window as any).go?.bindings?.SyncHandler || (window as any).go?.main?.App : null;
            if (wailsSync && typeof wailsSync.PullAndImportProject === "function") {
                const existingEntry = Object.entries(state.cloudProjectMappings).find(
                    ([_, meta]) => meta.cloudProjectId === cloudProjectId
                );
                const existingLocalId = existingEntry ? Number(existingEntry[0]) : 0;
                if (existingLocalId > 0 && typeof wailsSync.SetCloudProjectID === "function") {
                    await wailsSync.SetCloudProjectID(existingLocalId, cloudProjectId);
                }
                localSongId = await wailsSync.PullAndImportProject(state.cloudUrl, state.token, cloudProjectId);
                try {
                    const cp = await syncApi.getProjectDetails(cloudProjectId, state.token, state.cloudUrl);
                    if (cp.version && cp.version > 0) pulledVersion = cp.version;
                    if (cp.content_hash) pulledHash = cp.content_hash;
                    if (cp.updated_at) pulledUpdatedAt = cp.updated_at;
                } catch {
                    // Ignore metadata enrichment failure
                }
            } else {
                // Browser fallback / mock environment
                const cp = await syncApi.pullProject(cloudProjectId, state.token, state.cloudUrl);
                localSongId = cp.id;
                if (cp.version && cp.version > 0) {
                    pulledVersion = cp.version;
                }
                if (cp.content_hash) {
                    pulledHash = cp.content_hash;
                }
                if (cp.updated_at) {
                    pulledUpdatedAt = cp.updated_at;
                }
            }

            const nowIso = new Date().toISOString();
            const syncedTimestamp =
                new Date(pulledUpdatedAt).getTime() > new Date(nowIso).getTime()
                    ? pulledUpdatedAt
                    : nowIso;

            // Update mapping
            const updatedMappings = {
                ...state.cloudProjectMappings,
                [localSongId]: {
                    cloudProjectId,
                    version: pulledVersion,
                    lastSyncedAt: syncedTimestamp,
                    lastLocalUpdatedAt: syncedTimestamp,
                    lastSyncedHash: pulledHash,
                },
            };
            saveStoredMappings(updatedMappings);

            // Update activeSongStore with cloud_id on pull import
            activeSongStore.setCloudId(cloudProjectId);

            internalStore.update((s) => ({
                ...s,
                isSyncing: false,
                isOffline: false,
                hasActiveCheckout: false,
                activeLock: null,
                isLockedByOther: false,
                lockedBy: null,
                lastSyncedAt: new Date().toISOString(),
                cloudProjectMappings: updatedMappings,
            }));

            return localSongId;
        } catch (err) {
            internalStore.update((s) => ({ ...s, isSyncing: false }));
            throw err;
        }
    },

    /**
     * Handle expired session (401 Unauthorized) (Issue #20)
     */
    handleSessionExpired: () => {
        syncStore.clearAuth();
        internalStore.update((s) => ({
            ...s,
            authError: "Tu sesión ha expirado. Por favor inicia sesión nuevamente.",
        }));
        if (typeof window !== "undefined" && typeof window.dispatchEvent === "function") {
            window.dispatchEvent(new CustomEvent("kizuna-session-expired"));
        }
    },

    setSyncing: (isSyncing: boolean) =>
        internalStore.update((s) => ({ ...s, isSyncing })),

    setProgress: (progress: SyncProgressPayload | null) =>
        internalStore.update((s) => ({ ...s, syncProgress: progress })),

    setLock: (isLocked: boolean, lockedBy: string | null) =>
        internalStore.update((s) => ({ ...s, isLockedByOther: isLocked, lockedBy })),

    clearError: () =>
        internalStore.update((s) => ({ ...s, authError: null })),

    getCloudIdForSong: (songId: number): number | null => {
        const state = get(internalStore);
        return state.cloudProjectMappings[songId]?.cloudProjectId ?? null;
    },

    setMapping: (songId: number, meta: CloudProjectMeta) => {
        internalStore.update((s) => {
            const updated = {
                ...s.cloudProjectMappings,
                [songId]: meta,
            };
            saveStoredMappings(updated);
            return {
                ...s,
                cloudProjectMappings: updated,
            };
        });
    },
};
