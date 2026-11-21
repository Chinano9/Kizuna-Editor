import { test, describe, beforeEach } from "node:test";
import assert from "node:assert";
import { get } from "svelte/store";
import { activeSongStore, song, songId } from "../src/features/editor/stores/activeSongStore.ts";
import { autoSaveStore, AUTOSAVE_DEBOUNCE_MS } from "../src/features/editor/stores/autoSaveStore.ts";

describe("Reactive Debounced AutoSave Store", () => {
    beforeEach(() => {
        autoSaveStore.reset();
        autoSaveStore.setPersistHandler(null);
        activeSongStore.reset();
    });

    test("Default debounce delay is 1000ms (1 second)", () => {
        assert.strictEqual(AUTOSAVE_DEBOUNCE_MS, 1000);
        assert.strictEqual(get(autoSaveStore).debounceMs, 1000);
    });

    test("Rapid keystrokes reset the debounce timer and persist only once after user pauses", async () => {
        const currentSong = activeSongStore.createNewSong();
        const savedSnapshots: string[] = [];

        autoSaveStore.setPersistHandler(async (songToSave) => {
            savedSnapshots.push(songToSave.tracks[0].data_content);
            songToSave.id = 42;
            songToSave.tracks[0].id = 101;
            return songToSave;
        });

        // Simulate 3 rapid keystrokes within 40ms using a 60ms test debounce window
        currentSong.tracks[0].data_content = "[Am] Bajo";
        activeSongStore.updateTrack(currentSong.tracks[0]);
        autoSaveStore.scheduleSave(60);
        assert.strictEqual(get(autoSaveStore).status, "pending");

        await new Promise((r) => setTimeout(r, 20));
        currentSong.tracks[0].data_content = "[Am] Bajo el cielo";
        activeSongStore.updateTrack(currentSong.tracks[0]);
        autoSaveStore.scheduleSave(60);
        assert.strictEqual(savedSnapshots.length, 0);

        await new Promise((r) => setTimeout(r, 20));
        currentSong.tracks[0].data_content = "[Am] Bajo el cielo gris [G]";
        activeSongStore.updateTrack(currentSong.tracks[0]);
        autoSaveStore.scheduleSave(60);
        assert.strictEqual(savedSnapshots.length, 0);

        // Wait for the pause window to elapse
        await new Promise((r) => setTimeout(r, 90));

        assert.strictEqual(savedSnapshots.length, 1);
        assert.strictEqual(savedSnapshots[0], "[Am] Bajo el cielo gris [G]");
        assert.strictEqual(get(autoSaveStore).status, "saved");
        assert.ok(get(autoSaveStore).lastSavedAt);
        assert.strictEqual(get(songId), 42);
        assert.strictEqual(get(song)?.tracks[0].id, 101);
    });

    test("flushIfPending immediately persists pending changes when leaving the editor", async () => {
        const currentSong = activeSongStore.createNewSong();
        let persistedContent = "";

        autoSaveStore.setPersistHandler(async (songToSave) => {
            persistedContent = songToSave.tracks[0].data_content;
            return songToSave;
        });

        currentSong.tracks[0].data_content = ":4 0.1 2.2 3.3 5.4 |";
        activeSongStore.updateTrack(currentSong.tracks[0]);
        autoSaveStore.scheduleSave(5000); // Long debounce timer

        assert.strictEqual(get(autoSaveStore).status, "pending");
        await autoSaveStore.flushIfPending();

        assert.strictEqual(persistedContent, ":4 0.1 2.2 3.3 5.4 |");
        assert.strictEqual(get(autoSaveStore).status, "saved");
    });
});
