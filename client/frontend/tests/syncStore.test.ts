import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";

// Provide in-memory localStorage mock for Node test runner
const mockStore = new Map<string, string>();
const mockLocalStorage = {
    getItem: (key: string) => mockStore.get(key) ?? null,
    setItem: (key: string, val: string) => mockStore.set(key, String(val)),
    removeItem: (key: string) => mockStore.delete(key),
    clear: () => mockStore.clear(),
};

if (typeof (globalThis as any).window === "undefined") {
    (globalThis as any).window = { localStorage: mockLocalStorage };
    (globalThis as any).localStorage = mockLocalStorage;
} else if (!(globalThis as any).window.localStorage) {
    (globalThis as any).window.localStorage = mockLocalStorage;
}

import { get } from "svelte/store";
import { syncStore } from "../src/features/cloud-sync/stores/syncStore.ts";
import { authStorage } from "../src/features/cloud-sync/services/authApi.ts";

describe("Cloud Sync & Auth Store (Issue #12, #20)", () => {
    beforeEach(() => {
        mockStore.clear();
        syncStore.clearAuth();
    });

    test("Initial state is unauthenticated by default", () => {
        const state = get(syncStore);
        assert.strictEqual(state.isAuthenticated, false);
        assert.strictEqual(state.user, null);
        assert.strictEqual(state.token, null);
        assert.strictEqual(state.userEmail, null);
    });

    test("setAuth updates state with user object and token", () => {
        const user = {
            id: "usr_123",
            username: "jimi_hendrix",
            email: "jimi@guitar.com",
        };
        const token = "jwt.sample.token";

        syncStore.setAuth(user, token);

        const state = get(syncStore);
        assert.strictEqual(state.isAuthenticated, true);
        assert.strictEqual(state.user?.id, "usr_123");
        assert.strictEqual(state.user?.username, "jimi_hendrix");
        assert.strictEqual(state.userEmail, "jimi@guitar.com");
        assert.strictEqual(state.token, token);
    });

    test("setAuth accepts email string for backwards compatibility", () => {
        syncStore.setAuth("jimi@guitar.com", "jwt.token.legacy");

        const state = get(syncStore);
        assert.strictEqual(state.isAuthenticated, true);
        assert.strictEqual(state.userEmail, "jimi@guitar.com");
        assert.strictEqual(state.user?.username, "jimi");
        assert.strictEqual(state.token, "jwt.token.legacy");
    });

    test("clearAuth clears state and purges storage", () => {
        syncStore.setAuth(
            { id: "usr_1", username: "eric", email: "eric@clapton.com" },
            "token.to.purge"
        );
        assert.strictEqual(get(syncStore).isAuthenticated, true);

        syncStore.clearAuth();

        const state = get(syncStore);
        assert.strictEqual(state.isAuthenticated, false);
        assert.strictEqual(state.user, null);
        assert.strictEqual(state.token, null);
        assert.strictEqual(authStorage.getToken(), null);
    });

    test("setCloudUrl updates target server endpoint and persists it", () => {
        const customUrl = "https://kizuna-companion.example.com";
        syncStore.setCloudUrl(customUrl);

        const state = get(syncStore);
        assert.strictEqual(state.cloudUrl, customUrl);
        assert.strictEqual(authStorage.getCloudUrl(), customUrl);
    });

    test("Storage persistence round-trip saves and retrieves user data", () => {
        const user = {
            id: "usr_roundtrip",
            username: "david_gilmour",
            email: "david@pinkfloyd.com",
        };
        authStorage.saveSession("token_roundtrip", user);

        assert.strictEqual(authStorage.getToken(), "token_roundtrip");
        assert.deepStrictEqual(authStorage.getUser(), user);

        authStorage.clearSession();
        assert.strictEqual(authStorage.getToken(), null);
        assert.strictEqual(authStorage.getUser(), null);
    });
});
