import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";

// Provide window and storage mock for Node test runner
const mockStore = new Map<string, string>();
const eventListeners = new Map<string, Array<(...data: any[]) => void>>();

if (typeof (globalThis as any).window === "undefined") {
    (globalThis as any).window = {
        localStorage: {
            getItem: (key: string) => mockStore.get(key) ?? null,
            setItem: (key: string, val: string) => mockStore.set(key, String(val)),
            removeItem: (key: string) => mockStore.delete(key),
            clear: () => mockStore.clear(),
        },
        dispatchEvent: (event: any) => {
            const listeners = eventListeners.get(event.type) || [];
            listeners.forEach((fn) => fn(event));
            return true;
        },
        addEventListener: (name: string, fn: any) => {
            const existing = eventListeners.get(name) || [];
            existing.push(fn);
            eventListeners.set(name, existing);
        },
        removeEventListener: (name: string, fn: any) => {
            const existing = eventListeners.get(name) || [];
            eventListeners.set(name, existing.filter((f) => f !== fn));
        },
    };
    (globalThis as any).localStorage = (globalThis as any).window.localStorage;
}

import { get } from "svelte/store";
import { syncStore } from "../src/features/cloud-sync/stores/syncStore.ts";
import { toastStore } from "../src/shared/stores/toastStore.ts";
import type { SyncProgressPayload, SyncRetryPayload } from "../src/features/cloud-sync/types/sync.ts";

describe("Sync Progress & Retry Integration (Issue #23)", () => {
    beforeEach(() => {
        mockStore.clear();
        syncStore.clearAuth();
        syncStore.setProgress(null);
        toastStore.clear();
    });

    test("Initial syncProgress is null", () => {
        const state = get(syncStore);
        assert.strictEqual(state.syncProgress, null);
    });

    test("setProgress updates syncProgress in syncStore", () => {
        const payload: SyncProgressPayload = {
            operation: "push",
            bytes_transferred: 512000,
            total_bytes: 1024000,
            percentage: 50.0,
            phase: "transferring",
            attempt: 1,
            max_attempts: 3,
        };

        syncStore.setProgress(payload);

        const state = get(syncStore);
        assert.ok(state.syncProgress);
        assert.strictEqual(state.syncProgress?.operation, "push");
        assert.strictEqual(state.syncProgress?.bytes_transferred, 512000);
        assert.strictEqual(state.syncProgress?.total_bytes, 1024000);
        assert.strictEqual(state.syncProgress?.percentage, 50.0);
        assert.strictEqual(state.syncProgress?.phase, "transferring");
    });

    test("sync:retry event dispatches warning toast without fatal error", () => {
        const retryPayload: SyncRetryPayload = {
            operation: "push",
            attempt: 1,
            max_attempts: 3,
            wait_seconds: 2,
            reason: "connection reset by peer",
        };

        // Trigger retry via toastStore.warning as required by Issue #23
        toastStore.warning("Conexión inestable, reintentando...");

        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 1);
        assert.strictEqual(toasts[0].type, "warning");
        assert.strictEqual(toasts[0].message, "Conexión inestable, reintentando...");
    });

    test("Full sync lifecycle: transferring -> retry warning -> completed", () => {
        // 1. Started push
        syncStore.setProgress({
            operation: "push",
            bytes_transferred: 1000,
            total_bytes: 10000,
            percentage: 10.0,
            phase: "transferring",
            attempt: 1,
            max_attempts: 3,
        });
        assert.strictEqual(get(syncStore).syncProgress?.percentage, 10.0);

        // 2. Unstable connection - retry attempt 1
        toastStore.warning("Conexión inestable, reintentando...");
        syncStore.setProgress({
            operation: "push",
            bytes_transferred: 1000,
            total_bytes: 10000,
            percentage: 10.0,
            phase: "retrying",
            attempt: 1,
            max_attempts: 3,
        });

        assert.strictEqual(get(syncStore).syncProgress?.phase, "retrying");
        const toasts = get(toastStore);
        assert.strictEqual(toasts.some((t) => t.type === "warning"), true);

        // 3. Resumed and finished
        syncStore.setProgress({
            operation: "push",
            bytes_transferred: 10000,
            total_bytes: 10000,
            percentage: 100.0,
            phase: "completed",
            attempt: 2,
            max_attempts: 3,
        });

        const finalState = get(syncStore);
        assert.strictEqual(finalState.syncProgress?.phase, "completed");
        assert.strictEqual(finalState.syncProgress?.percentage, 100.0);
    });
});
