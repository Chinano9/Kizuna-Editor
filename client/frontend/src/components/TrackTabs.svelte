<script lang="ts">
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
    let editingName: string = "";
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
        if (editingIndex !== index) {
            activeTrackIndex.set(index);
        }
    }

    async function addTrack() {
        if (!$song) return;
        try {
            const newTrackName = `Track ${$song.tracks.length + 1}`;
            const newTrack = await AddTrack($song.id, newTrackName);
            $song.tracks = [...$song.tracks, newTrack];
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
            song.set($song); // Refresh UI
        } catch (err) {
            console.error("Failed to update track name:", err);
            alert("Error saving track name.");
            // Optionally revert the name change in the UI
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

            // Adjust active index if needed
            if ($activeTrackIndex >= index) {
                activeTrackIndex.set(Math.max(0, $activeTrackIndex - 1));
            }

            song.set($song); // Trigger UI update
        } catch (err) {
            console.error("Failed to delete track:", err);
            alert("Error: Could not delete the track.");
        }
    }

    async function handleInstrumentChange(track: main.Track) {
        try {
            await UpdateTrack(track);
            song.set($song); // Refresh UI to reflect change
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
        background-color: #181818;
        height: 30px; /* Match height of the top-nav */
    }

    .tab {
        background-color: #2d2d2d;
        border: none;
        border-right: 1px solid #181818;
        color: #aaa;
        padding: 0 5px 0 10px;
        cursor: pointer;
        font-size: 0.85rem;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: all 0.2s ease;
        user-select: none; /* Prevent text selection on double click */
    }

    .tab:hover {
        background-color: #3f3f3f;
        color: white;
    }

    .tab.active {
        background-color: #1e1e1e; /* Match workspace background */
        color: white;
        border-top: 2px solid #007acc;
    }

    .tab.editing {
        background-color: #3c3c3c;
    }

    .spacer {
        flex-grow: 1;
        border-bottom: 1px solid #333;
    }

    /* Adjust active tab to hide the container's bottom border */
    .tab.active {
        position: relative;
        bottom: -1px;
        border-bottom: 1px solid transparent;
    }

    /* The container itself provides the main bottom line */
    .track-tabs-container {
        border-bottom: 1px solid #333;
    }

    .add-track-btn {
        background-color: transparent;
        border: none;
        border-right: 1px solid #333;
        color: #888;
        cursor: pointer;
        font-size: 1.2rem;
        font-weight: bold;
        padding: 0 12px;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .add-track-btn:hover {
        background-color: #3f3f3f;
        color: white;
    }

    .name-input {
        background-color: #3c3c3c;
        border: 1px solid #007acc;
        color: white;
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
        background-color: #3c3c3c;
        color: #ccc;
        border: 1px solid #555;
        border-radius: 3px;
        font-size: 0.75rem;
        padding: 1px 3px;
        max-width: 100px;
    }
    .instrument-select:hover {
        background-color: #4f4f4f;
    }

    .delete-btn {
        background: none;
        border: none;
        color: #888;
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
        background-color: #555;
        color: white;
    }
</style>
