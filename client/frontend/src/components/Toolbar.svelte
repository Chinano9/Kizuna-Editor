<script lang="ts">
    import { get } from "svelte/store";
    import { SaveQuickIdea } from "../../wailsjs/go/main/App";
    import {
        songId,
        songTitle,
        trackSource,
        autoBar,
    } from "../stores/projectStore";

    async function save() {
        // Snapshot current state from stores without subscribing
        const currentTitle = get(songTitle);
        const currentSource = get(trackSource);
        const currentId = get(songId);

        if (!currentSource) {
            console.warn("Attempted to save without a track source.");
            return;
        }

        try {
            console.debug(`Saving project: "${currentTitle}"`);

            // Call Go Backend
            const newId = await SaveQuickIdea(
                currentId,
                currentTitle,
                currentSource,
            );

            // Update the store with the persisted ID from backend
            songId.set(newId);

            // TODO: Replace native alert with a Toast notification for better UX
            alert(`Saved successfully! ID: ${newId}`);
        } catch (err) {
            console.error("Failed to save project:", err);
            alert("Error while saving.");
        }
    }
</script>

<div class="panel-header control-bar">
    <input
        type="text"
        class="title-input"
        bind:value={$songTitle}
        placeholder="Song Title..."
    />

    <div class="actions">
        <label class="toggle-switch">
            <input type="checkbox" bind:checked={$autoBar} />
            <span>Auto |</span>
        </label>
        <button class="save-btn" on:click={save}>💾 Save</button>
    </div>
</div>

<style>
    .panel-header {
        background-color: #252526;
        color: #ccc;
        padding: 10px 15px;
        border-bottom: 1px solid #333;

        /* Layout stability */
        flex-shrink: 0; /* Prevents the editor area from compressing the header */
        min-height: 40px;
        display: flex;
        flex-direction: column;
        justify-content: center;
    }

    .control-bar {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 10px;
    }

    .title-input {
        background: transparent;
        border: none;
        color: white;
        font-size: 1rem;
        font-weight: bold;
        width: 100%;
        outline: none;
        border-bottom: 1px solid transparent;
    }

    .title-input:focus {
        border-bottom-color: #4ec9b0;
    }

    .actions {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .save-btn {
        background-color: #0e639c;
        color: white;
        border: none;
        padding: 6px 12px;
        border-radius: 4px;
        cursor: pointer;
        white-space: nowrap;
    }

    .save-btn:hover {
        background-color: #1177bb;
    }

    .toggle-switch {
        display: flex;
        align-items: center;
        cursor: pointer;
        font-size: 0.8rem;
        color: #4ec9b0;
        white-space: nowrap;
    }

    .toggle-switch input {
        margin-right: 5px;
    }
</style>
