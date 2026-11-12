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

    test("pushProject rejects when unauthenticated", async () => {
        syncStore.clearAuth();
        const fakeSong: any = { id: 1, title: "Test Song", bpm: 120, time_signature: "4/4" };

        await assert.rejects(
            async () => {
                await syncStore.pushProject(fakeSong, []);
            },
            {
                message: "Debes iniciar sesión en Kizuna Cloud para respaldar tu proyecto.",
            }
        );
    });

    test("pushProject updates projectVersions and lastSyncedAt on successful push", async () => {
        // Authenticate
        syncStore.setAuth({ id: "usr_1", username: "thom", email: "thom@radiohead.com" }, "mock.token");

        // Mock global fetch for sync push
        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
            const urlStr = String(input);
            if (urlStr.includes("/api/v1/sync/push")) {
                const body = JSON.parse(String(init?.body));
                assert.strictEqual(body.title, "Karma Police");
                assert.strictEqual(body.base_version, 0);

                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            project_id: 101,
                            new_version: 1,
                            synced_at: "2026-11-12T03:00:00Z",
                            status: "created",
                        },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input, init);
        };

        try {
            const fakeSong: any = {
                id: 101,
                title: "Karma Police",
                bpm: 75,
                time_signature: "4/4",
                key_signature: "A minor",
                tracks: [],
            };

            const result = await syncStore.pushProject(fakeSong, []);
            assert.strictEqual(result.project_id, 101);
            assert.strictEqual(result.new_version, 1);
            assert.strictEqual(result.status, "created");

            const state = get(syncStore);
            assert.strictEqual(state.isSyncing, false);
            assert.strictEqual(state.lastSyncedAt, "2026-11-12T03:00:00Z");
            assert.strictEqual(state.projectVersions[101], 1);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });
});
