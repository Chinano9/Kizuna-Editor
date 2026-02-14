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
    import { SaveSong } from "../../wailsjs/go/main/App";
    import { song, autoBar } from "../stores/projectStore";

    async function save() {
        const currentSong = get(song);
        if (!currentSong) {
            alert("No song data to save.");
            return;
        }

        try {
            console.log("Saving song object:", currentSong);

            // Call the new backend function
            const updatedSong = await SaveSong(currentSong);

            // Update the store with the returned object, which contains new IDs
            song.set(updatedSong);

            // TODO: Replace browser alert with app toast/notification for better UX
            alert(`Saved successfully (ID: ${updatedSong.id})`);
        } catch (err) {
            console.error("Failed to save project:", err);
            alert("Failed to save project.");
        }
    }
</script>

<div class="panel-header control-bar">
    {#if $song}
        <input
            type="text"
            class="title-input"
            bind:value={$song.title}
            placeholder="Song Title..."
        />
    {/if}

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
        background-color: var(--editor-bg);
        color: var(--editor-text);
        padding: 10px 15px;
        border-bottom: 1px solid var(--editor-border);
        font-family: var(--editor-font);

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
        color: var(--editor-text);
        text-shadow: 0 0 5px var(--editor-text);
        font-size: 1rem;
        font-weight: bold;
        width: 100%;
        outline: none;
        border-bottom: 1px solid transparent;
        border-bottom-color: var(--editor-text);
    }

    .title-input:focus {
        border-bottom-color: var(--editor-text);
    }

    .actions {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .save-btn {
        background-color: var(--editor-border);
        color: var(--editor-text);
        text-shadow: 0 0 5px var(--editor-text);
        border: 1px solid var(--editor-text);
        padding: 6px 12px;
        border-radius: 4px;
        cursor: pointer;
        white-space: nowrap;
    }

    .save-btn:hover {
        background-color: var(--editor-text);
        color: var(--editor-bg);
    }

    .toggle-switch {
        display: flex;
        align-items: center;
        cursor: pointer;
        font-size: 0.8rem;
        color: var(--editor-text);
        text-shadow: 0 0 5px var(--editor-text);
        white-space: nowrap;
    }

    .toggle-switch input {
        margin-right: 5px;
    }
</style>
