<script lang="ts">
    import { onDestroy } from "svelte";
    import { get } from "svelte/store";
    import { GetSong } from "../../wailsjs/go/main/App";

    import Editor from "@/components/Editor.svelte";
    import ScoreViewer from "@/components/ScoreViewer.svelte";
    import TrackTabs from "@/components/TrackTabs.svelte";
    import Workbar from "@/components/Workbar.svelte";
    import AlbumSelector from "@/features/albums/components/AlbumSelector.svelte";

    import { songId, song, activeSongStore, autoSaveStore } from "@/stores/projectStore";
    import { currentView } from "@/stores/viewStore";
    import { activeTrackIndex } from "@/stores/trackStore";
    import { audioPlayerStore } from "@/features/audio-deck/stores/audioPlayerStore";
    import { syncStore } from "@/features/cloud-sync/stores/syncStore";
    import { toastStore } from "@shared/stores/toastStore";
    import { Track } from "@shared/types/wails";

    let activeTrack: Track | null = null;
    let distractionFree = false;

    // --- Reactive Data Loading ---
    $: {
        loadSong($songId);
    }

    async function inspectCloudLockIfSynced(localId: number) {
        const cloudId = syncStore.getCloudIdForSong(localId);
        if (!cloudId || cloudId <= 0) {
            syncStore.resetLockState();
            return;
        }

        activeSongStore.setCloudId(cloudId);

        const state = get(syncStore);
        if (!state.isAuthenticated) {
            syncStore.resetLockState();
            return;
        }

        try {
            const status = await syncStore.verifyProjectLock(cloudId);
            if (status.isLockedByOther) {
                toastStore.warning(
                    `🔒 Proyecto bloqueado por ${status.lockedBy || "otro colaborador"}. Modo solo lectura.`
                );
            } else if (status.isOffline) {
                toastStore.info(
                    "⚡ Sin conexión a Kizuna Cloud: Modo Local-First activado (puedes editar y guardar localmente)."
                );
            }
        } catch (err) {
            const msg = err instanceof Error ? err.message : String(err);
            console.warn("Could not verify cloud checkout lock:", msg);
        }
    }

    async function releaseActiveCheckoutOnExit() {
        const state = get(syncStore);
        if (!state.hasActiveCheckout || !state.isAuthenticated) {
            syncStore.resetLockState();
            return;
        }

        const currentSong = get(activeSongStore);
        const cloudId =
            Number(currentSong?.cloud_id) ||
            (currentSong?.id ? syncStore.getCloudIdForSong(currentSong.id) : 0) ||
            state.activeLock?.project_id ||
            0;

        if (cloudId > 0) {
            try {
                await syncStore.releaseProjectLock(cloudId);
            } catch (err) {
                console.warn("Failed to auto-release checkout lock on exit:", err);
                syncStore.resetLockState();
            }
        } else {
            syncStore.resetLockState();
        }
    }

    async function loadSong(id: number) {
        if (id === 0) {
            syncStore.resetLockState();
            // If the store is not already initialized, create default structure
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
                await inspectCloudLockIfSynced(id);
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

    async function goBack() {
        await releaseActiveCheckoutOnExit();
        await autoSaveStore.flushIfPending();
        audioPlayerStore.reset();
        currentView.set("dashboard");
    }

    onDestroy(() => {
        void releaseActiveCheckoutOnExit();
        void autoSaveStore.flushIfPending();
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
