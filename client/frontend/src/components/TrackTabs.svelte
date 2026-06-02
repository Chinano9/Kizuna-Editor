<script lang="ts">
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

    import { onMount, tick } from "svelte";

    import { activeTrackIndex } from "@/stores/trackStore";

    import { song } from "@/stores/projectStore";

    import type { main } from "../../wailsjs/go/models";

    import {
        AddTrack,
        DeleteTrack,
        UpdateTrack,
        GetInstruments,
    } from "../../wailsjs/go/main/App";

    let editingIndex: number | null = null;

    let editingName = "";

    let inputElement: HTMLInputElement;

    let instruments: main.Instrument[] = [];

    onMount(async () => {
        try {
            instruments = await GetInstruments();
        } catch (err) {
            console.error("Failed to load instruments:", err);
        }
    });

    function selectTrack(index: number) {
        // Evita cambiar de pista mientras se está editando el nombre de esa misma pista

        if (editingIndex !== index) {
            activeTrackIndex.set(index);
        }
    }

    async function addTrack() {
        if (!$song) return;

        try {
            const trackCount = $song.tracks ? $song.tracks.length : 0;
            const newTrackName = `Track ${trackCount + 1}`;

            const newTrack = await AddTrack($song.id, newTrackName);

            const currentTracks = $song.tracks || [];
            $song.tracks = [...currentTracks, newTrack];

            song.set($song);

            activeTrackIndex.set($song.tracks.length - 1);
        } catch (err) {
            console.error("Failed to add track:", err);

            alert("Error: Could not add a new track.");
        }
    }

    async function startEditing(index: number, currentName: string) {
        editingIndex = index;

        editingName = currentName;

        await tick();

        inputElement?.focus();

        inputElement?.select();
    }

    function handleKeydown(event: KeyboardEvent) {
        if (event.key === "Enter") {
            saveTrackName();
        } else if (event.key === "Escape") {
            cancelEditing();
        }
    }

    async function saveTrackName() {
        if (editingIndex === null) return;

        const trackToUpdate = $song.tracks[editingIndex];

        if (!trackToUpdate || trackToUpdate.name === editingName) {
            cancelEditing();

            return;
        }

        trackToUpdate.name = editingName;

        try {
            await UpdateTrack(trackToUpdate);

            song.set($song);
        } catch (err) {
            console.error("Failed to update track name:", err);

            alert("Error saving track name.");
        } finally {
            cancelEditing();
        }
    }

    function cancelEditing() {
        editingIndex = null;
    }

    async function deleteTrack(index: number) {
        if (!confirm("Are you sure you want to delete this track?")) return;

        const trackToDelete = $song.tracks[index];

        try {
            await DeleteTrack(trackToDelete.id);

            $song.tracks.splice(index, 1);

            // Ajustar el índice activo si es necesario

            if ($activeTrackIndex >= index) {
                activeTrackIndex.set(Math.max(0, $activeTrackIndex - 1));
            }

            song.set($song);
        } catch (err) {
            console.error("Failed to delete track:", err);

            alert("Error: Could not delete the track.");
        }
    }

    async function handleInstrumentChange(track: main.Track) {
        try {
            await UpdateTrack(track);

            song.set($song);
        } catch (err) {
            console.error("Failed to update instrument:", err);

            alert("Error saving instrument change.");
        }
    }
</script>

<div class="track-tabs-container">
    {#if $song && $song.tracks}
        {#each $song.tracks as track, i}
            <div
                class="tab"
                class:active={$activeTrackIndex === i && editingIndex !== i}
                class:editing={editingIndex === i}
                on:click={() => selectTrack(i)}
                on:keydown={(e) => e.key === "Enter" && selectTrack(i)}
                role="button"
                tabindex="0"
            >
                {#if editingIndex === i}
                    <input
                        bind:this={inputElement}
                        bind:value={editingName}
                        on:keydown={handleKeydown}
                        on:blur={saveTrackName}
                        class="name-input"
                    />
                {:else}
                    <span
                        class="track-name"
                        on:dblclick|stopPropagation={() =>
                            startEditing(i, track.name)}
                    >
                        {track.name || `Track ${i + 1}`}
                    </span>

                    <select
                        class="instrument-select"
                        bind:value={track.instrument_id}
                        on:change={() => handleInstrumentChange(track)}
                        on:click|stopPropagation
                    >
                        {#each instruments as instrument}
                            <option value={instrument.id}>
                                {instrument.name}
                            </option>
                        {/each}
                    </select>

                    <button
                        class="delete-btn"
                        on:click|stopPropagation={() => deleteTrack(i)}
                    >
                        &times;
                    </button>
                {/if}
            </div>
        {/each}
    {/if}

    <button class="add-track-btn" on:click={addTrack}>+</button>

    <div class="spacer" />
</div>

<style>
    .track-tabs-container {
        display: flex;

        align-items: stretch;

        background-color: var(--app-bg-alt);
        height: 30px; /* Match height of the top-nav */

        border-bottom: 1px solid var(--lcd-border);
        font-family: var(--lcd-font-ui);
    }

    .tab {
        background-color: #111819;
        border: none;

        border-right: 1px solid #060909;
        color: var(--lcd-text-muted);
        padding: 0 5px 0 10px;

        cursor: pointer;

        font-size: 0.85rem;

        display: flex;

        align-items: center;

        gap: 8px;

        transition: all 0.15s ease;
        user-select: none; /* Prevent text selection on double click */
    }

    .tab:hover {
        background-color: #162022;
        color: var(--lcd-text);
    }

    .tab.active {
        background-color: #050b0b; /* Match workspace LCD background */

        color: var(--lcd-text);

        border-top: 2px solid var(--lcd-amber);
        position: relative;

        bottom: -1px;

        border-bottom: 1px solid transparent;

        box-shadow:
            0 0 4px rgba(255, 201, 102, 0.4),
            0 0 12px rgba(255, 201, 102, 0.55);
    }

    .tab.editing {
        background-color: #1a2527;
    }

    .spacer {
        flex-grow: 1;

        border-bottom: 1px solid var(--lcd-border);
    }

    .add-track-btn {
        background-color: transparent;

        border: none;

        border-right: 1px solid var(--lcd-border);
        color: var(--lcd-text-soft);
        cursor: pointer;

        font-size: 1.2rem;

        font-weight: bold;

        padding: 0 12px;

        display: flex;

        align-items: center;

        justify-content: center;
    }

    .add-track-btn:hover {
        background-color: #182325;
        color: var(--lcd-text);
    }

    .name-input {
        background-color: #1a2527;
        border: 1px solid var(--lcd-border-strong);
        color: var(--lcd-text);
        font-family: inherit;

        font-size: inherit;

        padding: 2px 4px;

        width: 120px; /* Adjust width as needed */

        border-radius: 2px;

        outline: none;
    }

    .track-name {
        padding: 5px 0;

        border-radius: 2px;
    }

    .instrument-select {
        background-color: #141d1f;
        color: var(--lcd-text-soft);
        border: 1px solid var(--lcd-border);
        border-radius: 3px;

        font-size: 0.75rem;

        padding: 1px 3px;

        max-width: 110px;
    }

    .instrument-select:hover {
        background-color: #1d292b;
        color: var(--lcd-text);
    }

    .delete-btn {
        background: none;

        border: none;

        color: var(--lcd-text-muted);
        cursor: pointer;

        font-size: 1.2rem;

        line-height: 1;

        padding: 0 5px;

        border-radius: 50%;

        width: 20px;

        height: 20px;

        display: none; /* Hidden by default */

        align-items: center;

        justify-content: center;
    }

    .tab:hover .delete-btn,
    .tab.active .delete-btn {
        display: flex; /* Show on hover or when active */
    }

    .delete-btn:hover {
        background-color: #2b3739;
        color: var(--lcd-text);
    }
</style>
