/*
    Kizuna Editor - Cloud Sync & Auth Store (Feature: Cloud Sync - Dev 1 & 2)
    Provides reactive session state, JWT token management, and auto-persistence (Issue #12, #20).
*/
import { writable, get } from "svelte/store";
import type { AuthUser, LoginPayload, RegisterPayload } from "../types/auth.ts";
import { authApi, authStorage, DEFAULT_CLOUD_URL } from "../services/authApi.ts";

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
}

// Rehydrate state from storage if previously logged in
const persistedToken = authStorage.getToken();
const persistedUser = authStorage.getUser();
const persistedUrl = authStorage.getCloudUrl();

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

    setSyncing: (isSyncing: boolean) =>
        internalStore.update((s) => ({ ...s, isSyncing })),

    setLock: (isLocked: boolean, lockedBy: string | null) =>
        internalStore.update((s) => ({ ...s, isLockedByOther: isLocked, lockedBy })),

    clearError: () =>
        internalStore.update((s) => ({ ...s, authError: null })),
};
