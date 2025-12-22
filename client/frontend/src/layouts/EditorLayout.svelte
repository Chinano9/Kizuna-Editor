<script lang="ts">
    import { onMount } from "svelte";
    import { GetSong } from "../../wailsjs/go/main/App";

    import Toolbar from "@/components/Toolbar.svelte";
    import Editor from "@/components/Editor.svelte";
    import ScoreViewer from "@/components/ScoreViewer.svelte";

    import { songId, trackSource } from "@/stores/projectStore";
    import { currentView } from "@/stores/viewStore";

    // Data loading logic
    onMount(async () => {
        const id = $songId;

        // If ID is 0 the song is new — nothing to load from the DB
        if (id === 0) return;

        try {
            console.log("Loading full content for ID:", id);
            const songData = await GetSong(id);

            // Extract the first track's content and load it into the editor.
            // (Future: iterate tracks to select which one to edit.)
            if (songData.tracks && songData.tracks.length > 0) {
                trackSource.set(songData.tracks[0].data_content);
            } else {
                // Fallback when song exists but has no tracks (recovery)
                trackSource.set('\\title "Load Error" \\n.');
            }
        } catch (err) {
            console.error("Error fetching song:", err);
            alert("Failed to load the song.");
        }
    });

    function goBack() {
        currentView.set("dashboard");
    }
</script>

<div class="layout-wrapper">
    <div class="top-nav">
        <button class="back-btn" on:click={goBack}>← Back to Library</button>
        <div class="spacer"></div>
    </div>

    <div class="editor-workspace">
        <Editor>
            <div slot="toolbar">
                <Toolbar />
            </div>
        </Editor>
        <ScoreViewer />
    </div>
</div>

<style>
    .layout-wrapper {
        display: flex;
        flex-direction: column;
        height: 100%;
    }

    /* Small top bar for navigation */
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

    /* Workspace area fills the remaining space */
    .editor-workspace {
        flex: 1;
        display: flex;
        overflow: hidden;
    }
</style>
