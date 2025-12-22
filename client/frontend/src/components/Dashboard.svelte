<script lang="ts">
    import { onMount } from "svelte";
    import { GetRecentSongs } from "../../wailsjs/go/main/App";
    // Import the generated models used by the frontend
    import type { models } from "../../wailsjs/go/models";

    // Stores for navigation
    import { songId, songTitle, trackSource } from "../stores/projectStore";
    import { currentView } from "../stores/viewStore";

    let recentSongs: models.Song[] = [];

    onMount(async () => {
        try {
            // Backend call: fetch a lightweight list of recent songs
            recentSongs = await GetRecentSongs();
        } catch (err) {
            console.error("Error loading songs:", err);
        }
    });

    function openSong(song: models.Song) {
        // 1. Populate stores with the selected song metadata
        songId.set(song.id);
        songTitle.set(song.title);

        // Note: GetRecentSongs does not include full (heavy) content.
        // The editor will request the full song data (GetSong) when mounted.
        // This keeps the recent list fast and lightweight.

        // 2. Switch to the editor view
        currentView.set("editor");
    }

    function createNew() {
        songId.set(0);
        songTitle.set("New Song");
        trackSource.set('\\title \"New\" \\n.');
        currentView.set("editor");
    }
</script>

<div class="dashboard">
    <h1>Kizuna Library</h1>

    <button class="new-btn" on:click={createNew}>+ New Song</button>

    <div class="song-list">
        {#each recentSongs as song}
            <div class="song-card" on:click={() => openSong(song)}>
                <h3>{song.title}</h3>
                <p>Edited: {new Date(song.updated_at).toLocaleDateString()}</p>
            </div>
        {:else}
            <p>You have no saved songs yet.</p>
        {/each}
    </div>
</div>

<style>
    .dashboard {
        padding: 40px;
        color: white;
        height: 100%;
        overflow-y: auto;
    }
    .song-list {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: 20px;
        margin-top: 20px;
    }
    .song-card {
        background: #333;
        padding: 20px;
        border-radius: 8px;
        cursor: pointer;
        transition:
            transform 0.2s,
            background 0.2s;
    }
    .song-card:hover {
        transform: translateY(-5px);
        background: #444;
    }
    .new-btn {
        background: #4ec9b0;
        border: none;
        padding: 10px 20px;
        font-size: 1.2rem;
        cursor: pointer;
        color: black;
        font-weight: bold;
        border-radius: 5px;
    }
</style>
