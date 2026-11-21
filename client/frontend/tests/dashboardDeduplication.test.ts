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

// Provide window and storage mock for Node test runner
const mockStore = new Map<string, string>();

if (typeof (globalThis as any).window === "undefined") {
    (globalThis as any).window = {
        localStorage: {
            getItem: (key: string) => mockStore.get(key) ?? null,
            setItem: (key: string, val: string) => mockStore.set(key, String(val)),
            removeItem: (key: string) => mockStore.delete(key),
            clear: () => mockStore.clear(),
        },
        dispatchEvent: () => true,
        addEventListener: () => {},
        removeEventListener: () => {},
    };
    (globalThis as any).localStorage = (globalThis as any).window.localStorage;
}

import {
    deduplicateRecentSongs,
    computeProjectSyncStatus,
    type CloudProjectMeta,
} from "../src/features/cloud-sync/stores/syncStore.ts";
import type { CloudProject } from "../src/features/cloud-sync/types/sync.ts";

describe("Dashboard Deduplication & Pull Overwrite Safeguards", () => {
    beforeEach(() => {
        mockStore.clear();
    });

    test("Deduplicates recent songs pointing to the same cloud project ID", () => {
        const recentSongs = [
            { id: 20, title: "Highway Star (Latest Pull)", updated_at: "2026-11-20T17:00:00Z" },
            { id: 19, title: "Highway Star (Duplicate Pull)", updated_at: "2026-11-20T16:00:00Z" },
            { id: 18, title: "Highway Star (Original Pull)", updated_at: "2026-11-20T15:00:00Z" },
            { id: 15, title: "Local Acoustic Jam", updated_at: "2026-11-20T14:00:00Z" },
            { id: 12, title: "Burn (Another Cloud Song)", updated_at: "2026-11-20T13:00:00Z" },
        ];

        const mappings: Record<number, CloudProjectMeta> = {
            20: { cloudProjectId: 1001, version: 3, lastSyncedAt: "2026-11-20T17:00:00Z" },
            19: { cloudProjectId: 1001, version: 2, lastSyncedAt: "2026-11-20T16:00:00Z" },
            18: { cloudProjectId: 1001, version: 1, lastSyncedAt: "2026-11-20T15:00:00Z" },
            12: { cloudProjectId: 2002, version: 1, lastSyncedAt: "2026-11-20T13:00:00Z" },
        };

        const result = deduplicateRecentSongs(recentSongs, mappings);

        // Should preserve only the first (latest) instance of cloud project 1001,
        // plus local song 15 and cloud song 12.
        assert.strictEqual(result.length, 3);
        assert.strictEqual(result[0].id, 20);
        assert.strictEqual(result[1].id, 15);
        assert.strictEqual(result[2].id, 12);
    });

    test("Preserves all local songs when none share a cloud project ID", () => {
        const recentSongs = [
            { id: 1, title: "Idea 1" },
            { id: 2, title: "Idea 2" },
            { id: 3, title: "Idea 3" },
        ];
        const mappings: Record<number, CloudProjectMeta> = {};

        const result = deduplicateRecentSongs(recentSongs, mappings);
        assert.strictEqual(result.length, 3);
        assert.deepStrictEqual(result.map((s) => s.id), [1, 2, 3]);
    });

    test("Handles empty or null song lists gracefully", () => {
        assert.deepStrictEqual(deduplicateRecentSongs([], {}), []);
        assert.deepStrictEqual(deduplicateRecentSongs(null as any, {}), []);
    });

    test("Overwrite Safeguard: Detects local_ahead when local modifications exist", () => {
        const localSong = {
            id: 20,
            title: "Highway Star (Edited)",
            updated_at: "2026-11-20T18:00:00Z",
            content_hash: "local_hash_999",
        };

        const mapping: CloudProjectMeta = {
            cloudProjectId: 1001,
            version: 2,
            lastSyncedAt: "2026-11-20T17:00:00Z",
            lastSyncedHash: "cloud_hash_111",
        };

        const cloudProject: CloudProject = {
            id: 1001,
            title: "Highway Star",
            bpm: 130,
            time_signature: "4/4",
            key_signature: "A",
            version: 2,
            content_hash: "cloud_hash_111",
            updated_at: "2026-11-20T17:00:00Z",
        } as any;

        const syncInfo = computeProjectSyncStatus(localSong as any, mapping, cloudProject);
        assert.strictEqual(syncInfo.status, "local_ahead");

        // The UI uses (status === 'local_ahead' || status === 'conflict') to prompt the user
        const shouldPromptUser = syncInfo.status === "local_ahead" || syncInfo.status === "conflict";
        assert.strictEqual(shouldPromptUser, true);
    });

    test("Overwrite Safeguard: Detects conflict when both local and cloud diverged", () => {
        const localSong = {
            id: 20,
            title: "Highway Star (Local Changes)",
            updated_at: "2026-11-20T18:00:00Z",
            content_hash: "local_hash_222",
        };

        const mapping: CloudProjectMeta = {
            cloudProjectId: 1001,
            version: 2,
            lastSyncedAt: "2026-11-20T17:00:00Z",
            lastSyncedHash: "baseline_hash_111",
        };

        const cloudProject: CloudProject = {
            id: 1001,
            title: "Highway Star (Cloud Changes)",
            version: 3,
            content_hash: "remote_hash_333",
            updated_at: "2026-11-20T18:30:00Z",
        } as any;

        const syncInfo = computeProjectSyncStatus(localSong as any, mapping, cloudProject);
        assert.strictEqual(syncInfo.status, "conflict");

        const shouldPromptUser = syncInfo.status === "local_ahead" || syncInfo.status === "conflict";
        assert.strictEqual(shouldPromptUser, true);
    });

    test("Overwrite Safeguard: Does NOT prompt when project is already in sync", () => {
        const localSong = {
            id: 20,
            title: "Highway Star",
            updated_at: "2026-11-20T17:00:00Z",
            content_hash: "hash_identical",
        };

        const mapping: CloudProjectMeta = {
            cloudProjectId: 1001,
            version: 2,
            lastSyncedAt: "2026-11-20T17:00:00Z",
            lastSyncedHash: "hash_identical",
        };

        const cloudProject: CloudProject = {
            id: 1001,
            title: "Highway Star",
            version: 2,
            content_hash: "hash_identical",
            updated_at: "2026-11-20T17:00:00Z",
        } as any;

        const syncInfo = computeProjectSyncStatus(localSong as any, mapping, cloudProject);
        assert.strictEqual(syncInfo.status, "synced");

        const shouldPromptUser = syncInfo.status === "local_ahead" || syncInfo.status === "conflict";
        assert.strictEqual(shouldPromptUser, false);
    });
});
