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

            // Call backend to persist a quick idea and receive (possibly new) ID
            const newId = await SaveQuickIdea(
                currentId,
                currentTitle,
                currentSource,
            );

            // Persist returned ID to the store
            songId.set(newId);

            // TODO: Replace browser alert with app toast/notification for better UX
            alert(`Saved successfully (ID: ${newId})`);
        } catch (err) {
            console.error("Failed to save project:", err);
            alert("Failed to save project.");
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
