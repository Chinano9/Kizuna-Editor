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

import { test, describe } from "node:test";
import assert from "node:assert";
import {
    computeProjectSyncStatus,
    type CloudProjectMeta,
} from "../src/features/cloud-sync/stores/syncStore.ts";
import type { CloudProject } from "../src/features/cloud-sync/types/sync.ts";

describe("Project Change Visualization: SQLite vs Cloud Timestamps & Content Hash (Issue #21)", () => {
    test("Returns 'local_only' ('Solo local') when song has never been backed up to Kizuna Cloud", () => {
        const localSong: any = {
            id: 1,
            title: "Unbacked Acoustic Riff",
            updated_at: "2026-11-19T20:00:00Z",
            content_hash: "a1b2c3d4e5f60718",
        };

        const res = computeProjectSyncStatus(localSong, null, null);
        assert.strictEqual(res.status, "local_only");
        assert.strictEqual(res.label, "Solo local");
        assert.strictEqual(res.icon, "💾");
        assert.strictEqual(res.localHash, "a1b2c3d4e5f60718");
    });

    test("Returns 'synced' ('Sincronizado') when SQLite and Cloud versions, hashes, and timestamps match", () => {
        const localSong: any = {
            id: 2,
            title: "Synced Studio Track",
            updated_at: "2026-11-19T20:00:00Z",
            content_hash: "1122334455667788",
        };
        const mapping: CloudProjectMeta = {
            cloudProjectId: 1001,
            version: 2,
            lastSyncedAt: "2026-11-19T20:00:05Z",
            lastLocalUpdatedAt: "2026-11-19T20:00:00Z",
            lastSyncedHash: "1122334455667788",
        };
        const cloudProject: CloudProject = {
            id: 1001,
            owner_id: "usr_1",
            title: "Synced Studio Track",
            bpm: 120,
            time_signature: "4/4",
            key_signature: "C",
            version: 2,
            content_hash: "1122334455667788",
            updated_at: "2026-11-19T20:00:05Z",
        };

        const res = computeProjectSyncStatus(localSong, mapping, cloudProject);
        assert.strictEqual(res.status, "synced");
        assert.strictEqual(res.label, "Sincronizado");
        assert.strictEqual(res.icon, "☁️");
        assert.strictEqual(res.cloudVersion, 2);
    });

    test("Git-style ContentHash prevents false-positive 'local_ahead' when user saves without changing content", () => {
        // User clicked Save 10 minutes later (updated_at advanced), but content_hash is identical to cloud
        const localSong: any = {
            id: 2,
            title: "Synced Studio Track",
            updated_at: "2026-11-19T20:10:00Z",
            content_hash: "1122334455667788",
        };
        const mapping: CloudProjectMeta = {
            cloudProjectId: 1001,
            version: 2,
            lastSyncedAt: "2026-11-19T20:00:05Z",
            lastLocalUpdatedAt: "2026-11-19T20:00:00Z",
            lastSyncedHash: "1122334455667788",
        };
        const cloudProject: CloudProject = {
            id: 1001,
            owner_id: "usr_1",
            title: "Synced Studio Track",
            bpm: 120,
            time_signature: "4/4",
            key_signature: "C",
            version: 2,
            content_hash: "1122334455667788",
            updated_at: "2026-11-19T20:00:05Z",
        };

        const res = computeProjectSyncStatus(localSong, mapping, cloudProject);
        assert.strictEqual(res.status, "synced");
        assert.strictEqual(res.label, "Sincronizado");
    });

    test("Returns 'local_ahead' ('Tiene cambios sin subir') when SQLite has a newer recording/edit than the cloud", () => {
        const localSong: any = {
            id: 3,
            title: "Late Night Vocal Take",
            updated_at: "2026-11-19T21:15:00Z",
            content_hash: "ffeeddccbbaa9988", // Hash changed after recording new audio take
        };
        const mapping: CloudProjectMeta = {
            cloudProjectId: 1002,
            version: 1,
            lastSyncedAt: "2026-11-19T20:00:00Z",
            lastLocalUpdatedAt: "2026-11-19T20:00:00Z",
            lastSyncedHash: "1122334455667788",
        };
        const cloudProject: CloudProject = {
            id: 1002,
            owner_id: "usr_1",
            title: "Late Night Vocal Take",
            bpm: 120,
            time_signature: "4/4",
            key_signature: "Am",
            version: 1,
            content_hash: "1122334455667788",
            updated_at: "2026-11-19T20:00:00Z",
        };

        const res = computeProjectSyncStatus(localSong, mapping, cloudProject);
        assert.strictEqual(res.status, "local_ahead");
        assert.strictEqual(res.label, "Tiene cambios sin subir");
        assert.strictEqual(res.icon, "⬆️");
    });

    test("Returns 'local_ahead' ('Tiene cambios sin subir') by timestamp comparison when hashes are omitted", () => {
        const localSong: any = {
            id: 3,
            title: "Timestamp Only Song",
            updated_at: "2026-11-19T21:30:00Z",
        };
        const mapping: CloudProjectMeta = {
            cloudProjectId: 1002,
            version: 1,
            lastSyncedAt: "2026-11-19T20:00:00Z",
        };

        const res = computeProjectSyncStatus(localSong, mapping, null);
        assert.strictEqual(res.status, "local_ahead");
        assert.strictEqual(res.label, "Tiene cambios sin subir");
    });

    test("Returns 'cloud_ahead' ('La nube está más actualizada') when collaborator pushed a newer version to cloud", () => {
        const localSong: any = {
            id: 4,
            title: "Shared Band Arrangement",
            updated_at: "2026-11-19T20:00:00Z",
            content_hash: "1122334455667788",
        };
        const mapping: CloudProjectMeta = {
            cloudProjectId: 1003,
            version: 1,
            lastSyncedAt: "2026-11-19T20:00:00Z",
            lastLocalUpdatedAt: "2026-11-19T20:00:00Z",
            lastSyncedHash: "1122334455667788",
        };
        const cloudProject: CloudProject = {
            id: 1003,
            owner_id: "usr_1",
            title: "Shared Band Arrangement",
            bpm: 120,
            time_signature: "4/4",
            key_signature: "G",
            version: 2, // Collaborator pushed v2
            content_hash: "9988776655443322",
            updated_at: "2026-11-19T21:00:00Z",
        };

        const res = computeProjectSyncStatus(localSong, mapping, cloudProject);
        assert.strictEqual(res.status, "cloud_ahead");
        assert.strictEqual(res.label, "La nube está más actualizada");
        assert.strictEqual(res.icon, "⬇️");
        assert.strictEqual(res.cloudVersion, 2);
    });

    test("Returns 'conflict' ('Cambios locales y en nube') when both local SQLite and Kizuna Cloud diverged", () => {
        const localSong: any = {
            id: 5,
            title: "Diverged Song",
            updated_at: "2026-11-19T21:20:00Z",
            content_hash: "aaaa1111bbbb2222",
        };
        const mapping: CloudProjectMeta = {
            cloudProjectId: 1004,
            version: 1,
            lastSyncedAt: "2026-11-19T20:00:00Z",
            lastLocalUpdatedAt: "2026-11-19T20:00:00Z",
            lastSyncedHash: "0000000000000000",
        };
        const cloudProject: CloudProject = {
            id: 1004,
            owner_id: "usr_1",
            title: "Diverged Song",
            bpm: 120,
            time_signature: "4/4",
            key_signature: "D",
            version: 2,
            content_hash: "cccc3333dddd4444",
            updated_at: "2026-11-19T21:10:00Z",
        };

        const res = computeProjectSyncStatus(localSong, mapping, cloudProject);
        assert.strictEqual(res.status, "conflict");
        assert.strictEqual(res.label, "Cambios locales y en nube");
        assert.strictEqual(res.icon, "⚠️");
    });
});
