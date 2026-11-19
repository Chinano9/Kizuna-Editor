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
let mockShareProjectHandler: ((cloudId: string, collaborator: string) => Promise<void>) | null = null;

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
        go: {
            main: {
                App: {
                    ShareProject: (cloudId: string, collaborator: string) => {
                        if (mockShareProjectHandler) {
                            return mockShareProjectHandler(cloudId, collaborator);
                        }
                        return Promise.resolve();
                    },
                    SetCloudSession: () => {},
                },
            },
        },
    };
    (globalThis as any).localStorage = (globalThis as any).window.localStorage;
} else {
    (globalThis as any).window.go = {
        main: {
            App: {
                ShareProject: (cloudId: string, collaborator: string) => {
                    if (mockShareProjectHandler) {
                        return mockShareProjectHandler(cloudId, collaborator);
                    }
                    return Promise.resolve();
                },
                SetCloudSession: () => {},
            },
        },
    };
}

import { get } from "svelte/store";
import { activeSongStore } from "../src/features/editor/stores/activeSongStore.ts";
import { syncStore } from "../src/features/cloud-sync/stores/syncStore.ts";
import { toastStore } from "../src/shared/stores/toastStore.ts";
import { ShareProject } from "../wailsjs/go/main/App.js";

describe("Share Project Feature & Validation (Issue #15)", () => {
    beforeEach(() => {
        mockStore.clear();
        toastStore.clear();
        syncStore.clearAuth();
        activeSongStore.reset();
        mockShareProjectHandler = null;
    });

    test("Pre-validation: Blocks opening share modal if song has no cloud_id", () => {
        // Initialize an active local song without cloud_id
        activeSongStore.createNewSong();
        const currentSong = get(activeSongStore);

        assert.ok(currentSong);
        assert.strictEqual(currentSong?.cloud_id, undefined);

        // Validation logic as performed in Toolbar before opening modal
        let isModalOpened = false;
        let cloudId = currentSong?.cloud_id;
        if (!cloudId && currentSong && currentSong.id > 0) {
            cloudId = syncStore.getCloudIdForSong(currentSong.id);
        }

        if (!cloudId || String(cloudId).trim() === "" || String(cloudId) === "0") {
            toastStore.warning("Debes respaldar el proyecto en la nube antes de invitar colaboradores");
        } else {
            isModalOpened = true;
        }

        assert.strictEqual(isModalOpened, false);
        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 1);
        assert.strictEqual(toasts[0].type, "warning");
        assert.strictEqual(toasts[0].message, "Debes respaldar el proyecto en la nube antes de invitar colaboradores");
    });

    test("Pre-validation: Allows opening share modal when song has a valid cloud_id", () => {
        activeSongStore.createNewSong();
        activeSongStore.setCloudId(1001);

        const currentSong = get(activeSongStore);
        assert.strictEqual(currentSong?.cloud_id, 1001);

        let isModalOpened = false;
        let resolvedCloudId: string | number | null = null;

        let cloudId = currentSong?.cloud_id;
        if (!cloudId && currentSong && currentSong.id > 0) {
            cloudId = syncStore.getCloudIdForSong(currentSong.id);
        }

        if (!cloudId || String(cloudId).trim() === "" || String(cloudId) === "0") {
            toastStore.warning("Debes respaldar el proyecto en la nube antes de invitar colaboradores");
        } else {
            isModalOpened = true;
            resolvedCloudId = cloudId;
        }

        assert.strictEqual(isModalOpened, true);
        assert.strictEqual(resolvedCloudId, 1001);
        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 0);
    });

    test("Sync fallback: Resolves cloud_id from syncStore mappings if not present directly on song object", () => {
        activeSongStore.createNewSong();
        // Manually simulate a local song with id 42
        activeSongStore.setSong({
            id: 42,
            title: "Revolver Session",
            tracks: [],
            bpm: 120,
            time_signature: "4/4",
            key_signature: "Am",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        } as any);

        // Populate mapping in syncStore
        syncStore.setMapping(42, {
            cloudProjectId: 5055,
            version: 2,
            lastSyncedAt: new Date().toISOString(),
        });

        const resolvedCloudId = syncStore.getCloudIdForSong(42);
        assert.strictEqual(resolvedCloudId, 5055);
    });

    test("Modal validation: Emits warning if collaborator input is empty", async () => {
        let submitted = false;
        const collaboratorInput = "   ";

        if (!collaboratorInput.trim()) {
            toastStore.warning("Ingresa el correo o nombre de usuario del colaborador.");
        } else {
            submitted = true;
        }

        assert.strictEqual(submitted, false);
        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 1);
        assert.strictEqual(toasts[0].type, "warning");
        assert.strictEqual(toasts[0].message, "Ingresa el correo o nombre de usuario del colaborador.");
    });

    test("Wails binding: Successfully shares project and emits success toast", async () => {
        let calledWithCloudId = "";
        let calledWithCollaborator = "";

        mockShareProjectHandler = async (cId: string, collab: string) => {
            calledWithCloudId = cId;
            calledWithCollaborator = collab;
        };

        const cloudId = "1001";
        const collaborator = "george@thebeatles.com";

        await ShareProject(cloudId, collaborator);
        toastStore.success(`¡Invitación enviada a ${collaborator} con éxito!`);

        assert.strictEqual(calledWithCloudId, "1001");
        assert.strictEqual(calledWithCollaborator, "george@thebeatles.com");

        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 1);
        assert.strictEqual(toasts[0].type, "success");
        assert.strictEqual(toasts[0].message, "¡Invitación enviada a george@thebeatles.com con éxito!");
    });

    test("Wails binding: Catches 404 User Not Found error and notifies via toastStore.error", async () => {
        mockShareProjectHandler = async () => {
            throw new Error("Usuario no encontrado en Kizuna Cloud");
        };

        try {
            await ShareProject("1001", "nonexistent@unknown.com");
            assert.fail("should have thrown");
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);
            toastStore.error(message);
        }

        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 1);
        assert.strictEqual(toasts[0].type, "error");
        assert.strictEqual(toasts[0].message, "Usuario no encontrado en Kizuna Cloud");
    });

    test("Wails binding: Catches 403 Forbidden error and notifies via toastStore.error", async () => {
        mockShareProjectHandler = async () => {
            throw new Error("solo el propietario del proyecto puede invitar colaboradores");
        };

        try {
            await ShareProject("1001", "someone@kizuna.io");
            assert.fail("should have thrown");
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);
            toastStore.error(message);
        }

        const toasts = get(toastStore);
        assert.strictEqual(toasts.length, 1);
        assert.strictEqual(toasts[0].type, "error");
        assert.strictEqual(toasts[0].message, "solo el propietario del proyecto puede invitar colaboradores");
    });
});
