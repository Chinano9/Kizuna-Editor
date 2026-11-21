/*
    Kizuna Editor - A local-first songwriting environment.
    Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";

const mockStore = new Map<string, string>();
const mockLocalStorage = {
    getItem: (key: string) => mockStore.get(key) ?? null,
    setItem: (key: string, val: string) => mockStore.set(key, String(val)),
    removeItem: (key: string) => mockStore.delete(key),
    clear: () => mockStore.clear(),
};

if (typeof (globalThis as any).window === "undefined") {
    (globalThis as any).window = {
        localStorage: mockLocalStorage,
        dispatchEvent: () => true,
        addEventListener: () => {},
        removeEventListener: () => {},
    };
    (globalThis as any).localStorage = mockLocalStorage;
} else {
    (globalThis as any).window.localStorage = mockLocalStorage;
    (globalThis as any).window.dispatchEvent = () => true;
}

import { get } from "svelte/store";
import { syncStore, isSongReadOnly } from "../src/features/cloud-sync/stores/syncStore.ts";
import { activeSongStore } from "../src/features/editor/stores/activeSongStore.ts";

describe("Collaborative Project Check-out (#16), Lock Release (#22) & Offline Local-First", () => {
    beforeEach(() => {
        mockStore.clear();
        syncStore.clearAuth();
        syncStore.resetLockState();
        activeSongStore.reset();
        if ((globalThis as any).window?.go) {
            delete (globalThis as any).window.go;
        }
    });

    test("Local-only project (cloudId == 0) is always editable without requiring a cloud lock", () => {
        syncStore.setAuth(
            { id: "usr_alice", username: "alice", email: "alice@kizuna.io" },
            "alice.jwt.token"
        );
        const state = get(syncStore);
        assert.strictEqual(state.hasActiveCheckout, false);
        assert.strictEqual(isSongReadOnly(state, 0), false);
    });

    test("Issue #16: Cloud project is Read-Only when online and user has NOT acquired checkout lock", () => {
        syncStore.setAuth(
            { id: "usr_alice", username: "alice", email: "alice@kizuna.io" },
            "alice.jwt.token"
        );
        const state = get(syncStore);
        assert.strictEqual(state.hasActiveCheckout, false);
        assert.strictEqual(state.isOffline, false);
        // Must block editing until user explicitly checks out the project!
        assert.strictEqual(isSongReadOnly(state, 1001), true);
    });

    test("Issue #16: checkoutProject acquires lock when project is free and unlocks editing (isSongReadOnly = false)", async () => {
        syncStore.setAuth(
            { id: "usr_alice", username: "alice", email: "alice@kizuna.io" },
            "alice.jwt.token"
        );

        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
            const urlStr = String(input);
            if (urlStr.includes("/api/v1/projects/1001/checkout") && init?.method === "POST") {
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            project_id: 1001,
                            user_id: "usr_alice",
                            username: "alice",
                            acquired_at: "2026-11-18T20:00:00Z",
                            expires_at: "2026-11-18T20:30:00Z",
                        },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input, init);
        };

        try {
            // Before checkout: read-only
            assert.strictEqual(isSongReadOnly(get(syncStore), 1001), true);

            const lock = await syncStore.checkoutProject(1001);
            assert.ok(lock);
            assert.strictEqual(lock?.project_id, 1001);
            assert.strictEqual(lock?.username, "alice");

            const state = get(syncStore);
            assert.strictEqual(state.hasActiveCheckout, true);
            assert.strictEqual(state.isLockedByOther, false);
            assert.strictEqual(state.lockedBy, null);
            assert.strictEqual(state.activeLock?.user_id, "usr_alice");
            // After checkout: editing is unlocked
            assert.strictEqual(isSongReadOnly(state, 1001), false);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });

    test("Issue #16: checkoutProject sets isLockedByOther = true and keeps Read-Only when locked by another user (409 Conflict)", async () => {
        syncStore.setAuth(
            { id: "usr_bob", username: "bob", email: "bob@kizuna.io" },
            "bob.jwt.token"
        );

        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL) => {
            const urlStr = String(input);
            if (urlStr.includes("/api/v1/projects/1001/checkout")) {
                return {
                    ok: false,
                    status: 409,
                    statusText: "Conflict",
                    json: async () => ({
                        success: false,
                        error: {
                            code: 409,
                            message:
                                "project is currently checked out by another user: user 'alice' holds the lock until 2026-11-18T20:30:00Z",
                            details: "alice",
                        },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input);
        };

        try {
            await assert.rejects(
                async () => {
                    await syncStore.checkoutProject(1001);
                },
                (err: Error) => {
                    assert.ok(err.message.includes("alice"));
                    return true;
                }
            );

            const state = get(syncStore);
            assert.strictEqual(state.isLockedByOther, true);
            assert.strictEqual(state.lockedBy, "alice");
            assert.strictEqual(state.hasActiveCheckout, false);
            assert.strictEqual(state.activeLock, null);
            assert.strictEqual(isSongReadOnly(state, 1001), true);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });

    test("Issue #16: verifyProjectLock detects active lock held by another collaborator", async () => {
        syncStore.setAuth(
            { id: "usr_bob", username: "bob", email: "bob@kizuna.io" },
            "bob.jwt.token"
        );

        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL) => {
            const urlStr = String(input);
            if (urlStr.endsWith("/api/v1/projects/1001")) {
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            id: 1001,
                            owner_id: "usr_alice",
                            title: "Shared Song",
                            bpm: 120,
                            time_signature: "4/4",
                            version: 2,
                            active_lock: {
                                project_id: 1001,
                                user_id: "usr_charlie",
                                username: "charlie",
                                acquired_at: "2026-11-18T20:00:00Z",
                                expires_at: "2026-11-18T20:30:00Z",
                            },
                        },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input);
        };

        try {
            const status = await syncStore.verifyProjectLock(1001);
            assert.strictEqual(status.isLockedByOther, true);
            assert.strictEqual(status.lockedBy, "charlie");

            const state = get(syncStore);
            assert.strictEqual(state.isLockedByOther, true);
            assert.strictEqual(state.lockedBy, "charlie");
            assert.strictEqual(state.hasActiveCheckout, false);
            assert.strictEqual(isSongReadOnly(state, 1001), true);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });

    test("Issue #16: pushProject blocks sync when project is locked by another user", async () => {
        syncStore.setAuth(
            { id: "usr_bob", username: "bob", email: "bob@kizuna.io" },
            "bob.jwt.token"
        );
        syncStore.setMapping(10, {
            cloudProjectId: 1001,
            version: 1,
            lastSyncedAt: "2026-11-18T19:00:00Z",
        });
        syncStore.setLock(true, "alice");

        const fakeSong: any = {
            id: 10,
            title: "Shared Song",
            bpm: 120,
            time_signature: "4/4",
            key_signature: "C",
            tracks: [],
        };

        await assert.rejects(
            async () => {
                await syncStore.pushProject(fakeSong, []);
            },
            {
                message: "No puedes sincronizar: el proyecto está bloqueado por alice.",
            }
        );
    });

    test("Issue #22: pushProject automatically releases lock AND blocks further editing (isSongReadOnly = true)", async () => {
        syncStore.setAuth(
            { id: "usr_alice", username: "alice", email: "alice@kizuna.io" },
            "alice.jwt.token"
        );
        syncStore.setMapping(10, {
            cloudProjectId: 1001,
            version: 1,
            lastSyncedAt: "2026-11-18T19:00:00Z",
        });

        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
            const urlStr = String(input);
            if (urlStr.includes("/api/v1/projects/1001/checkout")) {
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            project_id: 1001,
                            user_id: "usr_alice",
                            username: "alice",
                            acquired_at: "2026-11-18T20:00:00Z",
                            expires_at: "2026-11-18T20:30:00Z",
                        },
                    }),
                } as unknown as Response;
            }
            if (urlStr.includes("/api/v1/sync/push") && init?.method === "POST") {
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            project_id: 1001,
                            new_version: 2,
                            synced_at: "2026-11-18T20:10:00Z",
                            status: "synced",
                            lock_released: true,
                        },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input, init);
        };

        try {
            // 1. Alice checks out project -> editing enabled
            await syncStore.checkoutProject(1001);
            assert.strictEqual(get(syncStore).hasActiveCheckout, true);
            assert.strictEqual(isSongReadOnly(get(syncStore), 1001), false);

            // 2. Alice pushes changes
            const fakeSong: any = {
                id: 10,
                title: "Shared Song v2",
                bpm: 120,
                time_signature: "4/4",
                key_signature: "C",
                tracks: [],
            };
            const res = await syncStore.pushProject(fakeSong, []);
            assert.strictEqual(res.new_version, 2);
            assert.strictEqual(res.lock_released, true);

            // 3. Checkout lock is automatically released AND project immediately transitions to Read-Only!
            const state = get(syncStore);
            assert.strictEqual(state.hasActiveCheckout, false);
            assert.strictEqual(state.activeLock, null);
            assert.strictEqual(state.isLockedByOther, false);
            assert.strictEqual(state.lockedBy, null);
            assert.strictEqual(isSongReadOnly(state, 1001), true);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });

    test("Issue #22: releaseProjectLock explicitly releases lock via 'Terminar edición' and locks editor in Read-Only", async () => {
        syncStore.setAuth(
            { id: "usr_alice", username: "alice", email: "alice@kizuna.io" },
            "alice.jwt.token"
        );

        let releaseEndpointCalled = false;
        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
            const urlStr = String(input);
            if (urlStr.includes("/api/v1/projects/1001/checkout")) {
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            project_id: 1001,
                            user_id: "usr_alice",
                            username: "alice",
                            acquired_at: "2026-11-18T20:00:00Z",
                            expires_at: "2026-11-18T20:30:00Z",
                        },
                    }),
                } as unknown as Response;
            }
            if (urlStr.includes("/api/v1/projects/1001/release") && init?.method === "POST") {
                releaseEndpointCalled = true;
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: { message: "lock released successfully" },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input, init);
        };

        try {
            await syncStore.checkoutProject(1001);
            assert.strictEqual(get(syncStore).hasActiveCheckout, true);
            assert.strictEqual(isSongReadOnly(get(syncStore), 1001), false);

            await syncStore.releaseProjectLock(1001);
            assert.strictEqual(releaseEndpointCalled, true);

            const state = get(syncStore);
            assert.strictEqual(state.hasActiveCheckout, false);
            assert.strictEqual(state.activeLock, null);
            assert.strictEqual(state.isLockedByOther, false);
            assert.strictEqual(isSongReadOnly(state, 1001), true);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });

    test("Offline Local-First: Falls back to offline editable mode when network is unreachable, and returns to Read-Only after reconnecting and pushing", async () => {
        syncStore.setAuth(
            { id: "usr_alice", username: "alice", email: "alice@kizuna.io" },
            "alice.jwt.token"
        );
        syncStore.setMapping(10, {
            cloudProjectId: 1001,
            version: 1,
            lastSyncedAt: "2026-11-18T19:00:00Z",
        });

        let isServerOnline = false;
        const originalFetch = globalThis.fetch;
        globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
            if (!isServerOnline) {
                throw new TypeError("Failed to fetch: network unreachable");
            }
            const urlStr = String(input);
            if (urlStr.includes("/api/v1/sync/push") && init?.method === "POST") {
                return {
                    ok: true,
                    status: 200,
                    statusText: "OK",
                    json: async () => ({
                        success: true,
                        data: {
                            project_id: 1001,
                            new_version: 2,
                            synced_at: "2026-11-18T21:00:00Z",
                            status: "synced",
                            lock_released: true,
                        },
                    }),
                } as unknown as Response;
            }
            return originalFetch(input, init);
        };

        try {
            // 1. User opens cloud project while offline -> verifyProjectLock detects network failure
            const status = await syncStore.verifyProjectLock(1001);
            assert.strictEqual(status.isOffline, true);
            assert.strictEqual(status.isLockedByOther, false);

            const offlineState = get(syncStore);
            assert.strictEqual(offlineState.isOffline, true);
            // Local-First: must NOT lock out the user from local editing when offline!
            assert.strictEqual(isSongReadOnly(offlineState, 1001), false);

            // 2. Network comes back online and user pushes their local offline edits
            isServerOnline = true;
            const fakeSong: any = {
                id: 10,
                title: "Offline Composed Riff",
                bpm: 128,
                time_signature: "4/4",
                key_signature: "Em",
                tracks: [],
            };
            const pushRes = await syncStore.pushProject(fakeSong, []);
            assert.strictEqual(pushRes.new_version, 2);

            // 3. After reconnecting and pushing, isOffline is false, lock is released, and editor is Read-Only
            const onlineState = get(syncStore);
            assert.strictEqual(onlineState.isOffline, false);
            assert.strictEqual(onlineState.hasActiveCheckout, false);
            assert.strictEqual(isSongReadOnly(onlineState, 1001), true);
        } finally {
            globalThis.fetch = originalFetch;
        }
    });
});
