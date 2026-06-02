<script lang="ts">
    import { GetSong } from "../../wailsjs/go/main/App";

    import Toolbar from "@/components/Toolbar.svelte";
    import Editor from "@/components/Editor.svelte";
    import ScoreViewer from "@/components/ScoreViewer.svelte";
    import TrackTabs from "@/components/TrackTabs.svelte";
    import Workbar from "@/components/Workbar.svelte";

    import { songId, song } from "@/stores/projectStore";
    import { currentView } from "@/stores/viewStore";
    import { activeTrackIndex } from "@/stores/trackStore";
    import type { main } from "../../wailsjs/go/models";

    let activeTrack: main.Track | null = null;

    // --- Reactive Data Loading ---
    $: {
        loadSong($songId);
    }

    async function loadSong(id: number) {
        if (id === 0) {
            // Create a default structure for a new song
            const newTrack: main.Track = {
                id: 0,
                song_id: 0,
                instrument_id: 1, // Default to guitar
                name: "Track 1",
                data_content: '\\title "New Song"\\n.',
                display_mode: "BOTH",
                is_muted: false,
                created_at: new Date().toISOString(),
            };
            const newSong: main.Song = {
                id: 0,
                title: "My New Idea",
                tracks: [newTrack],
                // Set other defaults as needed
                album_id: null,
                bpm: 120,
                time_signature: "4/4",
                key_signature: "",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            };
            song.set(newSong);
            activeTrackIndex.set(0);
            return;
        }

        try {
            console.log("Loading full content for ID:", id);
            const songData = await GetSong(id);
            song.set(songData);
            activeTrackIndex.set(0);
        } catch (err) {
            console.error("Error fetching song:", err);
            alert("Failed to load the song.");
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
        <div class="spacer" />
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
