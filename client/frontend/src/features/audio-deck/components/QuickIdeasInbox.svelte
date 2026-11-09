<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import {
        ConvertQuickIdeaToSong,
        DeleteQuickIdea,
        GetPlaybackDuration,
        GetPlaybackPosition,
        GetQuickIdeas,
        PauseAudio,
        PlayAudio,
        SeekAudio,
        StopAudio,
    } from "@wails/go/main/App";
    import type { QuickIdea } from "@shared/types/wails";
    import { toastStore } from "@shared/stores/toastStore";
    import { songId } from "@/stores/projectStore";
    import { currentView } from "@/stores/viewStore";

    let ideas: QuickIdea[] = [];
    let isLoading = true;
    let loadError = "";
    let playingIdeaID: number | null = null;
    let isPlaying = false;
    let positionSeconds = 0;
    let seekPreviewSeconds = 0;
    let isSeeking = false;
    let durationSeconds = 0;
    let pollTimer: ReturnType<typeof setInterval> | undefined;
    let playbackSession = 0;
    let isPolling = false;
    let selectedIdea: QuickIdea | null = null;
    let songTitleDraft = "";
    let convertingIdeaID: number | null = null;
    let deletingIdeaID: number | null = null;

    onMount(() => {
        void loadIdeas();
        window.addEventListener("quick-idea-saved", handleQuickIdeaSaved);
        return () => window.removeEventListener("quick-idea-saved", handleQuickIdeaSaved);
    });

    function handleQuickIdeaSaved() {
        void loadIdeas();
    }

    async function loadIdeas() {
        isLoading = true;
        loadError = "";
        try {
            ideas = await GetQuickIdeas();
        } catch (error) {
            console.error("Failed to load quick ideas:", error);
            loadError = errorMessage(error);
        } finally {
            isLoading = false;
        }
    }

    function errorMessage(error: unknown): string {
        return error instanceof Error ? error.message : String(error);
    }

    function openConversionDialog(idea: QuickIdea) {
        selectedIdea = idea;
        songTitleDraft = `Quick Idea #${idea.id}`;
    }

    async function convertSelectedIdea() {
        const idea = selectedIdea;
        const title = songTitleDraft.trim();
        if (!idea || !title || convertingIdeaID !== null) return;

        convertingIdeaID = idea.id;
        try {
            if (playingIdeaID === idea.id) {
                await StopAudio();
                clearPollTimer();
                playbackSession++;
                playingIdeaID = null;
                isPlaying = false;
                positionSeconds = 0;
                seekPreviewSeconds = 0;
                durationSeconds = 0;
                isSeeking = false;
            }

            const newSongID = await ConvertQuickIdeaToSong(idea.id, title);
            selectedIdea = null;
            await loadIdeas();
            toastStore.success("Quick Idea converted to a song.");
            songId.set(newSongID);
            currentView.set("editor");
        } catch (error) {
            console.error("Failed to convert quick idea:", error);
            toastStore.error(`Could not convert Quick Idea: ${errorMessage(error)}`);
        } finally {
            convertingIdeaID = null;
        }
    }

    async function deleteIdea(idea: QuickIdea) {
        if (deletingIdeaID !== null) return;
        const confirmed = window.confirm(
            `Delete Quick Idea #${idea.id}? Its audio file will be permanently deleted.`
        );
        if (!confirmed) return;

        deletingIdeaID = idea.id;
        try {
            if (playingIdeaID === idea.id) {
                await StopAudio();
                clearPollTimer();
                playbackSession++;
                playingIdeaID = null;
                isPlaying = false;
                positionSeconds = 0;
                seekPreviewSeconds = 0;
                durationSeconds = 0;
                isSeeking = false;
            }

            await DeleteQuickIdea(idea.id);
            ideas = ideas.filter((savedIdea) => savedIdea.id !== idea.id);
            toastStore.success("Quick Idea and audio file deleted.");
        } catch (error) {
            console.error("Failed to delete quick idea:", error);
            toastStore.error(`Could not delete Quick Idea: ${errorMessage(error)}`);
        } finally {
            deletingIdeaID = null;
        }
    }

    function clearPollTimer() {
        if (pollTimer !== undefined) {
            clearInterval(pollTimer);
            pollTimer = undefined;
        }
    }

    function handleSeekInput(event: Event) {
        const target = event.currentTarget;
        if (!(target instanceof HTMLInputElement) || durationSeconds <= 0) return;
        seekPreviewSeconds = Math.min(durationSeconds, Math.max(0, Number(target.value)));
        isSeeking = true;
    }

    async function handleSeekCommit(event: Event) {
        const target = event.currentTarget;
        if (!(target instanceof HTMLInputElement) || durationSeconds <= 0) return;

        const seekPosition = Math.min(durationSeconds, Math.max(0, Number(target.value)));
        try {
            await SeekAudio(seekPosition);
            positionSeconds = seekPosition;
            seekPreviewSeconds = seekPosition;
        } catch (error) {
            console.error("Failed to seek quick idea playback:", error);
            toastStore.error(`Could not seek audio: ${errorMessage(error)}`);
        } finally {
            isSeeking = false;
        }
    }

    async function togglePlayback(idea: QuickIdea) {
        try {
            if (playingIdeaID === idea.id && isPlaying) {
                await PauseAudio();
                isPlaying = false;
                clearPollTimer();
                return;
            }

            if (playingIdeaID === idea.id && positionSeconds < durationSeconds) {
                await PauseAudio();
                isPlaying = true;
                beginPlaybackPolling();
                return;
            }

            clearPollTimer();
            playbackSession++;
            await PlayAudio(idea.file_path);
            playingIdeaID = idea.id;
            positionSeconds = 0;
            seekPreviewSeconds = 0;
            isSeeking = false;
            durationSeconds = idea.duration_seconds ?? (await GetPlaybackDuration());
            isPlaying = true;
            beginPlaybackPolling();
        } catch (error) {
            console.error("Failed to play quick idea:", error);
            clearPollTimer();
            playingIdeaID = null;
            positionSeconds = 0;
            durationSeconds = 0;
            isPlaying = false;
            toastStore.error(`Could not play audio: ${errorMessage(error)}`);
        }
    }

    function beginPlaybackPolling() {
        clearPollTimer();
        const session = ++playbackSession;
        pollTimer = setInterval(async () => {
            if (session !== playbackSession || isPolling) return;
            isPolling = true;
            try {
                const position = await GetPlaybackPosition();
                if (session !== playbackSession || isSeeking) return;
                positionSeconds = position;
                seekPreviewSeconds = position;
                if (durationSeconds <= 0) {
                    durationSeconds = await GetPlaybackDuration();
                }
                if (durationSeconds > 0 && position >= durationSeconds) {
                    positionSeconds = durationSeconds;
                    isPlaying = false;
                    clearPollTimer();
                }
            } catch (error) {
                console.error("Failed to update quick idea playback position:", error);
                clearPollTimer();
                isPlaying = false;
                toastStore.error(`Could not read playback status: ${errorMessage(error)}`);
            } finally {
                isPolling = false;
            }
        }, 200);
    }

    function formatTime(seconds: number | undefined): string {
        if (seconds === undefined || !Number.isFinite(seconds)) return "--:--";
        const safeSeconds = Math.max(0, Math.floor(seconds));
        const minutes = Math.floor(safeSeconds / 60).toString().padStart(2, "0");
        const remainder = (safeSeconds % 60).toString().padStart(2, "0");
        return `${minutes}:${remainder}`;
    }

    function formatDate(value: string | Date | undefined): string {
        if (!value) return "Date unavailable";
        const date = new Date(value);
        return Number.isNaN(date.getTime()) ? "Date unavailable" : date.toLocaleString();
    }

    onDestroy(() => {
        clearPollTimer();
        playbackSession++;
        if (playingIdeaID !== null) {
            void StopAudio().catch((error) => {
                console.error("Failed to stop playback when leaving ideas inbox:", error);
            });
        }
    });
</script>

<section class="inbox">
    <header class="inbox-header">
        <div>
            <h1>Quick Ideas</h1>
        </div>
    </header>

    {#if isLoading}
        <div class="state-message" role="status">Loading ideas...</div>
    {:else if loadError}
        <div class="state-message error" role="alert">
            <p>Could not load the inbox: {loadError}</p>
            <button type="button" class="retry-button" on:click={loadIdeas}>Retry</button>
        </div>
    {:else if ideas.length === 0}
        <div class="state-message empty">
            <span class="empty-icon" aria-hidden="true">◉</span>
            <h2>No ideas recorded yet</h2>
            <p>Record a quick idea from the Dashboard button and it will appear here.</p>
        </div>
    {:else}
        <div class="idea-list">
            {#each ideas as idea (idea.id)}
                <article class="idea-card" class:active={playingIdeaID === idea.id}>
                    <button
                        type="button"
                        class="play-button"
                        aria-label={playingIdeaID === idea.id && isPlaying ? "Pause idea" : "Play idea"}
                        on:click={() => togglePlayback(idea)}
                    >
                        {playingIdeaID === idea.id && isPlaying ? "Ⅱ" : "▶"}
                    </button>

                    <div class="idea-main">
                        <h2>Quick Idea #{idea.id}</h2>
                        <p class="idea-date">{formatDate(idea.created_at)}</p>
                        {#if playingIdeaID === idea.id}
                            <label class="sr-only" for={`idea-seek-${idea.id}`}>
                                Seek Quick Idea #{idea.id}
                            </label>
                            <input
                                id={`idea-seek-${idea.id}`}
                                class="progress-slider"
                                type="range"
                                min="0"
                                max={durationSeconds}
                                step="0.01"
                                value={isSeeking ? seekPreviewSeconds : positionSeconds}
                                disabled={durationSeconds <= 0}
                                aria-valuetext={`${formatTime(isSeeking ? seekPreviewSeconds : positionSeconds)} of ${formatTime(durationSeconds)}`}
                                on:input={handleSeekInput}
                                on:change={handleSeekCommit}
                            />
                        {/if}
                    </div>

                    <span class="duration">
                        {#if playingIdeaID === idea.id}
                            {formatTime(isSeeking ? seekPreviewSeconds : positionSeconds)} / {formatTime(durationSeconds)}
                        {:else if idea.duration_seconds !== undefined}
                            {formatTime(idea.duration_seconds)}
                        {:else}
                            Duration unavailable
                        {/if}
                    </span>

                    <div class="idea-actions">
                        <button
                            type="button"
                            disabled={deletingIdeaID !== null}
                            on:click={() => deleteIdea(idea)}
                        >
                            {deletingIdeaID === idea.id ? "Deleting..." : "Delete"}
                        </button>
                        <button type="button" on:click={() => openConversionDialog(idea)}>Convert to Song</button>
                    </div>
                </article>
            {/each}
        </div>
    {/if}
</section>

{#if selectedIdea}
    <div class="modal-backdrop">
        <form
            class="conversion-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="conversion-title"
            on:submit|preventDefault={convertSelectedIdea}
        >
            <h2 id="conversion-title">Convert Quick Idea #{selectedIdea.id}</h2>
            <label for="song-title">Song title</label>
            <input id="song-title" bind:value={songTitleDraft} required maxlength="200" />
            <div class="dialog-actions">
                <button type="button" disabled={convertingIdeaID !== null} on:click={() => (selectedIdea = null)}>
                    Cancel
                </button>
                <button type="submit" disabled={!songTitleDraft.trim() || convertingIdeaID !== null}>
                    {convertingIdeaID === selectedIdea.id ? "Converting..." : "Convert to Song"}
                </button>
            </div>
        </form>
    </div>
{/if}

<style>
    .inbox {
        height: 100%;
        overflow: auto;
        box-sizing: border-box;
        padding: clamp(24px, 5vw, 56px);
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--lcd-font-ui, "VT323", monospace);
        background:
            radial-gradient(ellipse at 75% 0, rgba(42, 75, 58, 0.26), transparent 45%),
            var(--app-bg, #050607);
    }

    .inbox-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        gap: 20px;
        margin: 0 auto 30px;
        max-width: 1100px;
        border-bottom: 1px solid var(--lcd-border, #243524);
        padding-bottom: 22px;
    }

    h1 {
        margin: 0;
        font-size: clamp(1.8rem, 4vw, 2.8rem);
        letter-spacing: 0.04em;
    }

    .idea-count {
        color: var(--lcd-amber, #ffc966);
        border: 1px solid var(--lcd-button-border, #c3933b);
        padding: 6px 12px;
        white-space: nowrap;
    }

    .idea-list {
        display: grid;
        gap: 12px;
        max-width: 1100px;
        margin: 0 auto;
    }

    .idea-card {
        display: grid;
        grid-template-columns: 48px minmax(140px, 1fr) auto auto;
        align-items: center;
        gap: 18px;
        padding: 18px;
        border: 1px solid var(--lcd-border, #243524);
        border-radius: 6px;
        background: rgba(10, 16, 14, 0.88);
        box-shadow: inset 0 0 18px rgba(0, 0, 0, 0.25);
    }

    .idea-card.active {
        border-color: var(--lcd-amber, #ffc966);
        box-shadow: 0 0 12px rgba(255, 201, 102, 0.12);
    }

    .play-button {
        width: 44px;
        height: 44px;
        border: 1px solid var(--lcd-button-border, #c3933b);
        border-radius: 50%;
        color: var(--lcd-amber, #ffc966);
        background: #15110a;
        font-size: 1.15rem;
        cursor: pointer;
    }

    .idea-main h2 {
        font-size: 1.15rem;
        margin: 0 0 5px;
    }

    .idea-date {
        color: var(--lcd-text-muted, #8ca38b);
        font-size: 0.88rem;
        margin: 0;
    }

    .duration {
        color: var(--lcd-amber, #ffc966);
        font-variant-numeric: tabular-nums;
        white-space: nowrap;
    }

    .progress-slider {
        display: block;
        width: 100%;
        margin-top: 12px;
        accent-color: var(--lcd-text, #c8ff9f);
        cursor: pointer;
    }

    .progress-slider:disabled {
        cursor: not-allowed;
        opacity: 0.5;
    }

    .sr-only {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
    }

    .idea-actions {
        display: flex;
        gap: 8px;
    }

    .idea-actions button,
    .retry-button {
        color: #8ca38b;
        border: 1px solid #35443a;
        background: #101512;
        border-radius: 4px;
        padding: 7px 10px;
        font: inherit;
    }

    .idea-actions button:disabled {
        opacity: 0.55;
        cursor: not-allowed;
    }

    .modal-backdrop {
        position: fixed;
        inset: 0;
        z-index: 20;
        display: grid;
        place-items: center;
        padding: 20px;
        background: rgba(0, 0, 0, 0.72);
    }

    .conversion-dialog {
        display: grid;
        gap: 14px;
        width: min(100%, 440px);
        padding: 24px;
        border: 1px solid var(--lcd-amber, #ffc966);
        border-radius: 6px;
        color: var(--lcd-text, #c8ff9f);
        background: #0a100e;
        box-shadow: 0 12px 40px rgba(0, 0, 0, 0.55);
    }

    .conversion-dialog h2 {
        margin: 0;
    }

    .conversion-dialog input {
        min-width: 0;
        padding: 10px;
        border: 1px solid var(--lcd-border, #243524);
        border-radius: 4px;
        color: var(--lcd-text, #c8ff9f);
        background: #050607;
        font: inherit;
    }

    .dialog-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 8px;
    }

    .dialog-actions button {
        padding: 8px 12px;
        border: 1px solid var(--lcd-button-border, #c3933b);
        border-radius: 4px;
        color: var(--lcd-amber, #ffc966);
        background: #15110a;
        font: inherit;
        cursor: pointer;
    }

    .dialog-actions button:disabled {
        opacity: 0.55;
        cursor: not-allowed;
    }

    .state-message {
        max-width: 1100px;
        margin: 40px auto;
        text-align: center;
        color: var(--lcd-text-muted, #8ca38b);
    }

    .state-message.empty {
        padding: 60px 20px;
        border: 1px dashed var(--lcd-border, #243524);
        border-radius: 6px;
    }

    .empty-icon {
        color: var(--lcd-amber, #ffc966);
        font-size: 2rem;
    }

    .state-message h2 {
        color: var(--lcd-text, #c8ff9f);
    }

    .state-message.error {
        color: #ff8585;
    }

    .retry-button {
        margin-top: 8px;
        color: var(--lcd-amber, #ffc966);
        cursor: pointer;
    }

    @media (max-width: 800px) {
        .idea-card {
            grid-template-columns: 44px minmax(0, 1fr) auto;
            gap: 12px;
        }

        .idea-actions {
            grid-column: 2 / -1;
            flex-wrap: wrap;
        }
    }

    @media (max-width: 520px) {
        .inbox-header {
            align-items: flex-start;
            flex-direction: column;
        }

        .idea-card {
            grid-template-columns: 40px minmax(0, 1fr);
        }

        .duration,
        .idea-actions {
            grid-column: 2;
        }
    }
</style>
