<script lang="ts">
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
        // Set the ID of the song to open
        songId.set(songToOpen.id);
        // Switch to the editor view
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
        color: #e0e0e0;
    }

    header {
        text-align: center;
        margin-bottom: 40px;
    }

    header h1 {
        font-size: 2.5rem;
        margin: 0;
        color: white;
    }

    header p {
        font-size: 1.1rem;
        color: #a0a0a0;
    }

    .main-actions {
        text-align: center;
        margin-bottom: 50px;
    }

    .action-btn {
        background-color: #007acc;
        color: white;
        border: none;
        padding: 12px 24px;
        font-size: 1rem;
        font-weight: bold;
        border-radius: 5px;
        cursor: pointer;
        transition: background-color 0.2s ease;
    }
    .action-btn:hover {
        background-color: #005a9e;
    }

    .recent-work h2 {
        border-bottom: 1px solid #444;
        padding-bottom: 10px;
        margin-bottom: 20px;
    }

    .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
        gap: 20px;
    }

    .card {
        background-color: #2a2a2e;
        border: 1px solid #3a3a3e;
        border-radius: 5px;
        padding: 20px;
        cursor: pointer;
        transition:
            transform 0.2s ease,
            background-color 0.2s ease,
            border-color 0.2s ease;
    }

    .card:hover {
        transform: translateY(-3px);
        background-color: #2d2d2e;
        border-color: #007acc;
    }

    .card-info h3 {
        margin: 0 0 10px 0;
        color: white;
    }

    .card-info .date {
        font-size: 0.8rem;
        color: #888;
    }
</style>
