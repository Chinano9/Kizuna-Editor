<script lang="ts">
    import { onMount } from "svelte";
    // Ajusta la ruta de importación según tu estructura (si usas alias @ o relativa)
    import { GetRecentSongs } from "wailsjs/go/main/App";
    import type { models } from "wailsjs/go/models";

    import { songId, songTitle, trackSource } from "../stores/projectStore";
    import { currentView } from "../stores/viewStore";

    let recentSongs: models.Song[] = [];

    // 1. CORRECCIÓN: Declaramos la variable de estado
    let isLoading = true;

    onMount(async () => {
        try {
            recentSongs = await GetRecentSongs();
        } catch (err) {
            console.error("Error loading songs:", err);
        } finally {
            // 2. CORRECCIÓN: Apagamos el loading al terminar
            isLoading = false;
        }
    });

    function openSong(song: models.Song) {
        songId.set(song.id);
        songTitle.set(song.title);
        currentView.set("editor");
    }

    function createNew() {
        songId.set(0);
        songTitle.set("New Song");
        // 3. CORRECCIÓN: Usamos el template "blindado" con afinación explícita
        // para evitar los errores de percusión que vimos antes.
        trackSource.set('\\title "New" \n\\tuning E5 B4 G4 D4 A3 E3\n.\n:4 ');
        currentView.set("editor");
    }
</script>

<div class="view-content">
    <header class="view-header">
        <h2>Recent Projects</h2>
        <button class="btn-new" on:click={createNew}>+ Create New Song</button>
    </header>

    <div class="scroll-area">
        {#if isLoading}
            <div class="loading">Loading library...</div>
        {:else}
            <div class="grid">
                {#each recentSongs as song}
                    <div class="card" on:click={() => openSong(song)}>
                        <div class="card-info">
                            <h3>{song.title}</h3>
                            <span class="date"
                                >{new Date(
                                    song.updated_at,
                                ).toLocaleDateString()}</span
                            >
                        </div>
                    </div>
                {:else}
                    <div class="empty-state">
                        <p>No hay canciones guardadas.</p>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .view-content {
        display: flex;
        flex-direction: column;
        height: 100%;
    }

    .view-header {
        padding: 20px 40px;
        border-bottom: 1px solid #333;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .view-header h2 {
        margin: 0;
        font-weight: 400;
        font-size: 1.5rem;
        color: #eee;
    }

    .scroll-area {
        flex: 1;
        overflow-y: auto;
        padding: 40px;
    }

    .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
        gap: 20px;
    }

    .card {
        background-color: #252526;
        padding: 20px;
        border-radius: 8px;
        cursor: pointer;
        border: 1px solid transparent;
        transition:
            transform 0.2s,
            background 0.2s,
            border-color 0.2s;
        display: flex;
        align-items: center;
        gap: 15px;
    }

    .card:hover {
        transform: translateY(-3px);
        background-color: #2d2d2e;
        border-color: #4ec9b0;
    }

    .card-icon {
        font-size: 2rem;
        opacity: 0.5;
    }

    .card-info h3 {
        margin: 0 0 5px 0;
        font-size: 1.1rem;
        color: white;
    }
    .card-info .date {
        font-size: 0.8rem;
        color: #888;
    }

    .btn-new {
        background: #0e639c;
        border: none;
        padding: 10px 20px;
        font-size: 1rem;
        cursor: pointer;
        color: white;
        font-weight: bold;
        border-radius: 4px;
        transition: background 0.2s;
    }
    .btn-new:hover {
        background: #1177bb;
    }

    .loading,
    .empty-state {
        text-align: center;
        color: #888;
        margin-top: 40px;
        font-style: italic;
        font-size: 1.1rem;
    }
</style>
