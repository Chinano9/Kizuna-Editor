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

            const updatedSong = await SaveSong(currentSong);

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
        background-color: var(--lcd-header-bg);
        color: var(--lcd-text);
        padding: 10px 15px;

        border-bottom: 1px solid var(--lcd-header-border);
        font-family: var(--lcd-font-ui);
        flex-shrink: 0;
        min-height: 40px;
        display: flex;
        flex-direction: row;
        justify-content: space-between;
        align-items: center;
        box-shadow: var(--lcd-header-shadow);
    }

    .control-bar {
        display: flex;
        justify-content: space-between;

        align-items: center;

        gap: 10px;

        width: 100%;
    }

    .title-input {
        background: transparent;

        border: none;

        color: var(--lcd-text);
        text-shadow: var(--lcd-glow-soft);
        font-size: 1rem;
        font-weight: bold;

        width: 100%;

        outline: none;

        border-bottom: 1px solid var(--lcd-border);
    }

    .title-input:focus {
        border-bottom-color: var(--lcd-border-strong);
    }

    .actions {
        display: flex;
        align-items: center;

        gap: 10px;
    }

    .save-btn {
        position: relative;

        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.45) 0,
            var(--lcd-button-bg) 60%
        );
        color: var(--lcd-amber);

        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        border: 1px solid var(--lcd-button-border);

        padding: 4px 12px;
        border-radius: 4px;
        cursor: pointer;

        white-space: nowrap;

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        font-family: var(--lcd-font-ui);

        font-size: 0.9rem;
        letter-spacing: 0.08em;

        text-transform: uppercase;

        transition:
            background-color 0.15s ease,
            border-color 0.15s ease,
            transform 0.1s ease,
            box-shadow 0.15s ease,
            filter 0.15s ease;
    }

    .save-btn::before {
        content: "";

        position: absolute;

        inset: 2px 2px auto 2px;
        height: 40%;
        background: linear-gradient(
            to bottom,

            rgba(255, 220, 170, 0.2),
            transparent
        );

        border-radius: 3px 3px 1px 1px;
        pointer-events: none;

        opacity: 0.85;
    }

    .save-btn:hover {
        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.6) 0,
            var(--lcd-button-bg-soft) 65%
        );
        border-color: var(--lcd-button-hover-border);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow);
        transform: translateY(-1px);

        filter: brightness(1.05);
    }

    .save-btn:active {
        transform: translateY(1px);

        filter: brightness(0.9);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.9),
            0 0 6px rgba(255, 201, 102, 0.45);
    }

    .toggle-switch {
        display: flex;

        align-items: center;

        cursor: pointer;

        font-size: 0.8rem;

        color: var(--lcd-text-soft);

        text-shadow: var(--lcd-glow-soft);

        white-space: nowrap;

        gap: 4px;
    }

    .toggle-switch input {
        margin-right: 5px;

        accent-color: var(--lcd-amber);
    }

    .toggle-switch span {
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--lcd-amber-soft);
    }
</style>
