<script lang="ts">
    import { onDestroy } from "svelte";
    import { GetSong } from "../../wailsjs/go/main/App";

    import Editor from "@/components/Editor.svelte";
    import ScoreViewer from "@/components/ScoreViewer.svelte";
    import TrackTabs from "@/components/TrackTabs.svelte";
    import Workbar from "@/components/Workbar.svelte";

    import { songId, song, activeSongStore } from "@/stores/projectStore";
    import { currentView } from "@/stores/viewStore";
    import { activeTrackIndex } from "@/stores/trackStore";
    import { audioPlayerStore } from "@/features/audio-deck/stores/audioPlayerStore";
    import { toastStore } from "@shared/stores/toastStore";
    import { Track } from "@shared/types/wails";

    let activeTrack: Track | null = null;
    let distractionFree = false;

    // --- Reactive Data Loading ---
    $: {
        loadSong($songId);
    }

    async function loadSong(id: number) {
        if (id === 0) {
            if (!$song || !$song.tracks || $song.tracks.length === 0) {
                activeSongStore.createNewSong();
            }
            activeTrackIndex.set(0);
            return;
        }

        if ($song && $song.id === id && $song.tracks && $song.tracks.length > 0) {
            return;
        }

        try {
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

    // Keep `activeTrack` in sync with activeSongStore and activeTrackIndex
    $: {
        if ($song && $song.tracks) {
            activeTrack = $song.tracks[$activeTrackIndex] ?? null;
        } else {
            activeTrack = null;
        }
    }

    function handleSeekFromScore(event: CustomEvent<number>) {
        audioPlayerStore.requestSeek(event.detail);
    }

    function handleTogglePlayFromScore() {
        audioPlayerStore.requestTogglePlay();
    }

    function goBack() {
        audioPlayerStore.reset();
        currentView.set("dashboard");
    }

    onDestroy(() => {
        audioPlayerStore.reset();
    });
</script>

<div class="layout-wrapper">
    {#if !distractionFree}
        <div class="top-nav">
            <button class="back-btn" on:click={goBack}>← Back to Library</button>
            <div class="spacer" />
        </div>

        <TrackTabs />
    {/if}

    <div class="editor-workspace">
        {#if activeTrack}
            <Editor
                bind:track={activeTrack}
                bind:distractionFree
            />
            <ScoreViewer
                track={activeTrack}
                isPlaying={$audioPlayerStore.isPlaying}
                playbackPosition={$audioPlayerStore.positionSeconds}
                playbackDuration={$audioPlayerStore.durationSeconds}
                on:seek={handleSeekFromScore}
                on:togglePlay={handleTogglePlayFromScore}
            />
        {:else}
            <div class="placeholder">Select a track to start editing.</div>
        {/if}
    </div>

    {#if activeTrack}
        <Workbar
            songID={$songId}
            activeTrack={activeTrack}
        />
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
        height: 30px;
        background-color: #181818;
        border-bottom: 1px solid #333;
        display: flex;
        align-items: center;
        padding: 0 10px;
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
