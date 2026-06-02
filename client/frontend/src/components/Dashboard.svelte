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

    import { onMount } from "svelte";

    import { GetRecentSongs } from "wailsjs/go/main/App";

    import type { models } from "wailsjs/go/models";

    import { songId } from "../stores/projectStore";

    import { currentView } from "../stores/viewStore";

    let recentSongs: models.Song[] = [];

    onMount(async () => {
        try {
            recentSongs = await GetRecentSongs();
        } catch (err) {
            console.error("Failed to load recent songs:", err);
        }
    });

    function newSong() {
        // Setting songId to 0 signals the editor to create a new song

        songId.set(0);

        currentView.set("editor");
    }

    function openSong(songToOpen: models.Song) {
        songId.set(songToOpen.id);

        currentView.set("editor");
    }
</script>

<div class="dashboard-container">
    <header>
        <h1>Kizuna Editor</h1>

        <p>Your local-first songwriting environment.</p>
    </header>

    <div class="main-actions">
        <button class="action-btn" on:click={newSong}>+ New Idea</button>
    </div>

    <section class="recent-work">
        <h2>Recent Work</h2>

        {#if recentSongs.length === 0}
            <p>No recent songs found. Start a new idea!</p>
        {:else}
            <div class="grid">
                {#each recentSongs as song}
                    <div
                        class="card"
                        on:click={() => openSong(song)}
                        on:keydown={(e) => e.key === "Enter" && openSong(song)}
                        role="button"
                        tabindex="0"
                    >
                        <div class="card-info">
                            <h3>{song.title}</h3>

                            <span class="date"
                                >Last updated: {new Date(
                                    song.updated_at,
                                ).toLocaleDateString()}</span
                            >
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </section>
</div>

<style>
    .dashboard-container {
        width: 100%;

        max-width: 960px;

        margin: 0 auto;

        padding: 40px 20px;

        color: var(--lcd-text);
        background: radial-gradient(
            circle at top,
            #0f1f1a 0,
            var(--app-bg) 45%
        );
        font-family: var(--lcd-font-ui);
        text-shadow: var(--lcd-glow-soft);
    }

    header {
        text-align: center;

        margin-bottom: 40px;
    }

    header h1 {
        font-size: 2.5rem;

        margin: 0;

        color: var(--lcd-text);
        letter-spacing: 0.08em;
    }

    header p {
        font-size: 1.1rem;

        color: var(--lcd-text-muted);
    }

    .main-actions {
        text-align: center;

        margin-bottom: 50px;
    }

    .action-btn {
        position: relative;

        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.45) 0,
            var(--lcd-button-bg) 60%
        );
        color: var(--lcd-amber);

        border: 1px solid var(--lcd-button-border);

        padding: 10px 26px;
        font-size: 1.05rem;
        font-weight: bold;

        border-radius: 4px;
        cursor: pointer;

        text-transform: uppercase;

        letter-spacing: 0.12em;

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.7),
            var(--lcd-glow-soft);

        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.9),
            var(--lcd-glow-soft);

        transition:
            background-color 0.15s ease,
            border-color 0.15s ease,
            transform 0.1s ease,
            box-shadow 0.15s ease,
            filter 0.15s ease;
    }

    .action-btn::before {
        content: "";

        position: absolute;

        inset: 2px 2px auto 2px;
        height: 35%;
        background: linear-gradient(
            to bottom,

            rgba(255, 220, 170, 0.18),
            transparent
        );

        border-radius: 3px 3px 1px 1px;
        pointer-events: none;

        opacity: 0.8;
    }

    .action-btn:hover {
        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.55) 0,
            var(--lcd-button-bg-soft) 65%
        );
        border-color: var(--lcd-button-hover-border);

        transform: translateY(-1px);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.7),
            var(--lcd-glow);
        filter: brightness(1.05);
    }

    .action-btn:active {
        transform: translateY(1px);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.9),
            0 0 6px rgba(255, 201, 102, 0.5);

        filter: brightness(0.92);
    }

    .recent-work h2 {
        border-bottom: 1px solid var(--lcd-border);
        padding-bottom: 10px;

        margin-bottom: 20px;

        color: var(--lcd-text-soft);
        text-transform: uppercase;
        letter-spacing: 0.1em;
        font-size: 0.9rem;
    }

    .grid {
        display: grid;

        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));

        gap: 20px;
    }

    .card {
        background: linear-gradient(135deg, #050b0b, #060f10);
        border: 1px solid var(--lcd-border);
        border-radius: 5px;

        padding: 16px 18px;
        cursor: pointer;

        transition:
            transform 0.15s ease,
            background-color 0.15s ease,
            border-color 0.15s ease,
            box-shadow 0.15s ease;
        box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.7);
    }

    .card:hover {
        transform: translateY(-2px);
        background-color: #071616;
        border-color: var(--lcd-border-strong);
        box-shadow: var(--lcd-glow);
    }

    .card-info h3 {
        margin: 0 0 6px 0;
        color: var(--lcd-text);
        font-size: 1.05rem;
    }

    .card-info .date {
        font-size: 0.8rem;

        color: var(--lcd-text-muted);
    }
</style>
