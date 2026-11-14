<script lang="ts">
    import { GetSong } from "../../wailsjs/go/main/App";

    import Toolbar from "@/components/Toolbar.svelte";
    import Editor from "@/components/Editor.svelte";
    import ScoreViewer from "@/components/ScoreViewer.svelte";
    import TrackTabs from "@/components/TrackTabs.svelte";
    import Workbar from "@/components/Workbar.svelte";
    import AlbumSelector from "@/features/albums/components/AlbumSelector.svelte";

    import { songId, song, activeSongStore } from "@/stores/projectStore";
    import { currentView } from "@/stores/viewStore";
    import { activeTrackIndex } from "@/stores/trackStore";
    import { toastStore } from "@shared/stores/toastStore";
    import { Track, Song } from "@shared/types/wails";

    let activeTrack: Track | null = null;

    // --- Reactive Data Loading ---
    $: {
        loadSong($songId);
    }

    async function loadSong(id: number) {
        if (id === 0) {
            // If the store is not already initialized, create default structure
            if (!$song || !$song.tracks || $song.tracks.length === 0) {
                activeSongStore.createNewSong();
            }
            activeTrackIndex.set(0);
            return;
        }

        try {
            console.log("Loading full content for ID:", id);
            const songData = await GetSong(id);
            if (songData) {
                song.set(songData);
                activeTrackIndex.set(0);
            }
        } catch (err) {
            console.error("Error fetching song:", err);
            toastStore.error("Failed to load the song.");
        }
    }

    // This reactive block now simply keeps `activeTrack` in sync.
    // The actual data binding will happen inside the Editor component.
    $: {
        if ($song && $song.tracks) {
            activeTrack = $song.tracks[$activeTrackIndex] ?? null;
        } else {
            activeTrack = null;
        }
    }

    function goBack() {
        currentView.set("dashboard");
    }
</script>

<div class="layout-wrapper">
    <div class="top-nav">
        <button class="back-btn" on:click={goBack}>← Back to Library</button>
        <div class="spacer"></div>
        <AlbumSelector />
    </div>

    <TrackTabs />
    <div class="editor-workspace">
        {#if activeTrack}
            <Editor bind:track={activeTrack} />
            <ScoreViewer track={activeTrack} />
        {:else}
            <div class="placeholder">Select a track to start editing.</div>
        {/if}
    </div>

    {#if activeTrack}
        <Workbar songID={$songId} activeTrack={activeTrack} />
    {/if}
</div>

<style>
    .layout-wrapper {
        display: flex;
        flex-direction: column;
        height: 100%;
        width: 100%;
    }
    
    .top-nav {
        min-height: 46px;
        background-color: #181818;
        border-bottom: 1px solid #333;
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 0 14px;
    }

    .spacer {
        flex: 1;
    }

    .back-btn {
        background: none;
        border: none;
        color: #888;
        cursor: pointer;
        font-size: 0.85rem;
        padding: 5px;
    }
    .back-btn:hover {
        color: white;
        text-decoration: underline;
    }

    .editor-workspace {
        flex: 1;
        display: flex;
        overflow: hidden;
        width: 100%;
        height: 100%;
        padding-bottom: 85px; /* Prevent fixed bottom Workbar from covering scrollable sheet music and editing areas */
    }

    .placeholder {
        display: flex;
        justify-content: center;
        align-items: center;
        width: 100%;
        height: 100%;
        font-size: 1.2rem;
        color: #666;
    }
</style>
