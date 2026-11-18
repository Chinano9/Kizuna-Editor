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
    import { onMount, onDestroy, tick, createEventDispatcher } from "svelte";
    // @ts-ignore
    import { AlphaTabApi } from "@coderline/alphatab";

    import {
        looksLikeAlphaTex,
        prepareAlphaTexForRender,
        extractScoreMetadata,
        calculatePlaybackSyncState,
        getTimestampForBar,
    } from "@shared/lib/musicUtils";
    import { formatTime } from "@shared/lib/formatters";
    import { autoBar, song } from "../stores/projectStore";
    import { instrumentStore } from "@/stores/instrumentStore";
    import { audioPlayerStore } from "@/features/audio-deck/stores/audioPlayerStore";
    import type { Track } from "@shared/types/wails";

    import LyricsViewer from "./LyricsViewer.svelte";
    import "../styles/preview-theme.css";

    export let track: Track | null = null;
    export let isPlaying: boolean | undefined = undefined;
    export let playbackPosition: number | undefined = undefined;
    export let playbackDuration: number | undefined = undefined;

    const dispatch = createEventDispatcher<{
        seek: number;
        togglePlay: void;
    }>();

    const DEBOUNCE_DELAY_MS = 220;

    let scoreContainer: HTMLElement;
    let previewScrollContainer: HTMLElement;
    let api: any = null;
    let instrumentName = "";
    let currentStaveProfile: "Default" | "Score" | "" = "";
    let currentScale = 0;

    // Debounce & non-blocking render state
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;
    let rafId: number | null = null;
    let lastTrackId: number | null = null;
    let lastScheduleKey = "";
    let lastRenderedTex = "";
    let hasRenderedScore = false;
    let isRendering = false;
    let renderError = "";

    // Synchronized score metadata
    let parsedScoreBars = 0;
    let parsedScoreTempo = 120;

    // Effective playback values (accept both direct props from EditorLayout and audioPlayerStore)
    $: effectiveIsPlaying = isPlaying !== undefined ? isPlaying : $audioPlayerStore.isPlaying;
    $: effectivePosition =
        playbackPosition !== undefined ? playbackPosition : $audioPlayerStore.positionSeconds;
    $: effectiveDuration =
        playbackDuration !== undefined ? playbackDuration : $audioPlayerStore.durationSeconds;

    $: fallbackBpm = $song?.bpm || 120;
    $: fallbackTs = $song?.time_signature || "4/4";
    $: scoreMeta = extractScoreMetadata(track?.data_content ?? "", fallbackBpm, fallbackTs);
    $: totalBars = Math.max(parsedScoreBars, scoreMeta.barCount, 1);
    $: activeBpm = parsedScoreTempo > 0 ? parsedScoreTempo : scoreMeta.bpm;

    $: syncState = calculatePlaybackSyncState(
        effectivePosition,
        effectiveDuration,
        totalBars,
        activeBpm,
        scoreMeta.timeSignature,
    );

    // Keep global store informed of active bar/beat
    $: {
        audioPlayerStore.setScoreSync(
            syncState.currentBarIndex,
            syncState.currentBeat,
            syncState.totalBars,
            syncState.bpm,
        );
    }

    // Synchronize AlphaTab timePosition and auto-scroll when playback position advances
    $: if (api && (effectiveIsPlaying || effectivePosition > 0)) {
        syncAlphaTabCursor(effectivePosition, syncState.scoreProgress, effectiveIsPlaying);
    }

    function syncAlphaTabCursor(posSeconds: number, progress: number, playing: boolean) {
        try {
            if (api && typeof api.timePosition === "number") {
                api.timePosition = Math.round(posSeconds * 1000);
            }
        } catch {
            // Player might be disabled in lightweight SVG mode; visual sync bar handles cursor
        }

        if (playing && previewScrollContainer) {
            const maxScroll =
                previewScrollContainer.scrollHeight - previewScrollContainer.clientHeight;
            if (maxScroll > 40) {
                const targetScroll = Math.max(0, Math.min(maxScroll, maxScroll * progress - 60));
                if (Math.abs(previewScrollContainer.scrollTop - targetScroll) > 35) {
                    previewScrollContainer.scrollTo({
                        top: targetScroll,
                        behavior: "smooth",
                    });
                }
            }
        }
    }

    function handleBarClick(barIndex: number) {
        const targetSeconds = getTimestampForBar(barIndex, activeBpm, scoreMeta.timeSignature);
        dispatch("seek", targetSeconds);
        audioPlayerStore.requestSeek(targetSeconds);
    }

    function handleTogglePlayClick() {
        dispatch("togglePlay");
        audioPlayerStore.requestTogglePlay();
    }

    function rescaleScoreToFitWidth() {
        if (!scoreContainer) return;

        // Keep scoreContainer responsive so AlphaTab's ResizeObserver and layout engine
        // always measure the full available panel width and never collapse height to a single partial.
        scoreContainer.style.width = "100%";
        scoreContainer.style.height = "auto";
        scoreContainer.style.transform = "none";

        const container =
            previewScrollContainer || (scoreContainer.parentElement as HTMLElement | null);
        if (!container) return;

        const availableWidth = container.clientWidth;
        if (!availableWidth) return;

        const surface = scoreContainer.querySelector(".at-surface") as HTMLElement | null;
        if (!surface) return;

        const surfaceWidth = parseFloat(surface.style.width || "0") || surface.offsetWidth;
        if (!surfaceWidth) return;

        const scale = Math.min(1, (availableWidth - 16) / surfaceWidth);
        if (Math.abs(scale - currentScale) < 0.005) return;
        currentScale = scale;

        if (scale < 0.995) {
            surface.style.transformOrigin = "top left";
            surface.style.transform = `scale(${scale})`;
        } else {
            surface.style.transform = "none";
        }
    }

    // Reactive block to determine the instrument name whenever track or instrument catalog changes
    $: {
        const _catalog = $instrumentStore;
        if (track) {
            instrumentName = instrumentStore.getNameById(track.instrument_id);
        } else {
            instrumentName = "";
        }
    }

    $: isPlainLyricsTrack =
        instrumentName === "Vocals" ||
        (Boolean(track?.data_content?.trim()) && !looksLikeAlphaTex(track?.data_content ?? ""));

    function clearScoreSurfaceOnly() {
        if (scoreContainer) {
            const surface = scoreContainer.querySelector(".at-surface") as HTMLElement | null;
            if (surface) {
                surface.innerHTML = "";
                surface.style.height = "0px";
            }
        }
    }

    function clearRenderedScore() {
        clearScoreSurfaceOnly();
        lastRenderedTex = "";
        hasRenderedScore = false;
        isRendering = false;
        renderError = "";
        parsedScoreBars = 0;
    }

    async function initializeApi(staveProfile: "Default" | "Score" = "Default"): Promise<boolean> {
        if (!scoreContainer) return false;

        if (api) {
            if (currentStaveProfile !== staveProfile) {
                try {
                    api.settings.display.staveProfile = staveProfile === "Score" ? 2 : 0;
                    api.updateSettings();
                    currentStaveProfile = staveProfile;
                } catch {
                    // Fallback if updateSettings is unavailable
                }
            }
            return false;
        }

        scoreContainer.style.width = "100%";
        scoreContainer.style.height = "auto";
        scoreContainer.style.transform = "none";

        api = new AlphaTabApi(scoreContainer, {
            core: {
                tex: false,
                enableLazyLoading: false,
                useWorkers: false,
                engine: "svg",
                fontDirectory: "/font/",
            },
            display: {
                staveProfile,
                layoutMode: "page",
                padding: [20, 20, 20, 20],
            },
            player: { enablePlayer: false },
        });

        currentStaveProfile = staveProfile;

        if (api.scoreLoaded && typeof api.scoreLoaded.on === "function") {
            api.scoreLoaded.on((loadedScore: any) => {
                if (loadedScore) {
                    if (Array.isArray(loadedScore.masterBars) && loadedScore.masterBars.length > 0) {
                        parsedScoreBars = loadedScore.masterBars.length;
                    }
                    if (typeof loadedScore.tempo === "number" && loadedScore.tempo > 0) {
                        parsedScoreTempo = loadedScore.tempo;
                    }
                }
            });
        }

        if (api.renderFinished && typeof api.renderFinished.on === "function") {
            api.renderFinished.on(() => {
                isRendering = false;
                renderError = "";
                hasRenderedScore = Boolean(lastRenderedTex);
                currentScale = 0;
                rescaleScoreToFitWidth();
            });
        }

        if (api.error && typeof api.error.on === "function") {
            api.error.on((err: any) => {
                isRendering = false;
                renderError = err?.message || "Sintaxis AlphaTex incompleta";
            });
        }

        // Wait one animation frame so AlphaTab's deferred uiFacade.initialRender()
        // registers its preRender listener and container width before api.tex() is called.
        if (typeof requestAnimationFrame === "function") {
            await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
        }

        return true;
    }

    function clearPendingRender() {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
            debounceTimer = null;
        }
        if (rafId !== null && typeof cancelAnimationFrame === "function") {
            cancelAnimationFrame(rafId);
            rafId = null;
        }
    }

    function destroyApi() {
        clearPendingRender();
        if (api) {
            api.destroy();
            api = null;
        }
        currentStaveProfile = "";
        lastRenderedTex = "";
        lastScheduleKey = "";
        hasRenderedScore = false;
        isRendering = false;
    }

    onMount(() => {
        if (track) {
            scheduleAlphaTabUpdate(track, instrumentName, $autoBar, true);
        }
    });

    onDestroy(() => {
        destroyApi();
    });

    // Main reactive block: schedules debounced non-blocking rendering as the user types
    $: scheduleAlphaTabUpdate(track, instrumentName, $autoBar, false);

    function scheduleAlphaTabUpdate(
        currentTrack: Track | null,
        currentInstrumentName: string,
        auto: boolean,
        immediate: boolean,
    ) {
        if (!currentTrack) {
            clearPendingRender();
            clearRenderedScore();
            instrumentName = "";
            lastTrackId = null;
            lastScheduleKey = "";
            return;
        }

        const content = currentTrack.data_content ?? "";
        const scheduleKey = `${currentTrack.id}:${currentInstrumentName}:${auto ? 1 : 0}:${content}`;
        if (!immediate && scheduleKey === lastScheduleKey) {
            return;
        }
        lastScheduleKey = scheduleKey;

        clearPendingRender();

        if (currentInstrumentName === "Vocals") {
            clearRenderedScore();
            lastTrackId = currentTrack.id;
            return;
        }

        const isTrackSwitch = lastTrackId !== currentTrack.id;
        lastTrackId = currentTrack.id;

        if (content.trim() && !looksLikeAlphaTex(content)) {
            clearRenderedScore();
            return;
        }

        const delay = immediate || isTrackSwitch || !lastRenderedTex ? 0 : DEBOUNCE_DELAY_MS;
        isRendering = true;

        debounceTimer = setTimeout(() => {
            debounceTimer = null;
            executeAlphaTabRender(currentTrack, currentInstrumentName, auto);
        }, delay);
    }

    async function executeAlphaTabRender(
        currentTrack: Track,
        currentInstrumentName: string,
        auto: boolean,
    ) {
        const content = currentTrack.data_content ?? "";
        const effectiveContent = content.trim();
        const preparedTex = prepareAlphaTexForRender(effectiveContent, auto);
        const fallbackRawTex = auto ? prepareAlphaTexForRender(effectiveContent, false) : preparedTex;

        if (!preparedTex) {
            clearRenderedScore();
            return;
        }

        await tick();
        if (!scoreContainer) {
            isRendering = false;
            return;
        }

        const newStaveProfile: "Default" | "Score" =
            currentInstrumentName === "Piano" ? "Score" : "Default";

        if (
            api &&
            lastRenderedTex === preparedTex &&
            currentStaveProfile === newStaveProfile &&
            hasRenderedScore
        ) {
            isRendering = false;
            return;
        }

        await initializeApi(newStaveProfile);
        if (!api || !scoreContainer) {
            isRendering = false;
            return;
        }

        parsedScoreBars = scoreMeta.barCount;
        parsedScoreTempo = scoreMeta.bpm;
        renderMusic(preparedTex, fallbackRawTex);
    }

    function renderMusic(textToRender: string, fallbackRawTex: string = "") {
        if (!api) {
            isRendering = false;
            return;
        }

        const trimmed = textToRender.trim();
        if (!trimmed) {
            clearRenderedScore();
            return;
        }

        try {
            renderError = "";
            lastRenderedTex = trimmed;
            api.tex(trimmed, [-1]);
            hasRenderedScore = !renderError;

            // If autoBar injection produced syntax that AlphaTab rejected, fall back to raw AlphaTex
            if (renderError && fallbackRawTex && fallbackRawTex.trim() !== trimmed) {
                renderError = "";
                lastRenderedTex = fallbackRawTex.trim();
                api.tex(lastRenderedTex, [-1]);
                hasRenderedScore = !renderError;
            }

            if (renderError) {
                clearScoreSurfaceOnly();
                hasRenderedScore = false;
            }

            tick().then(() => {
                isRendering = false;
                if (hasRenderedScore) {
                    rescaleScoreToFitWidth();
                }
            });
        } catch (err: any) {
            if (fallbackRawTex && fallbackRawTex.trim() !== trimmed) {
                try {
                    renderError = "";
                    lastRenderedTex = fallbackRawTex.trim();
                    api.tex(lastRenderedTex, [-1]);
                    hasRenderedScore = !renderError;
                    if (renderError) {
                        clearScoreSurfaceOnly();
                        hasRenderedScore = false;
                    }
                    tick().then(() => {
                        isRendering = false;
                        if (hasRenderedScore) {
                            rescaleScoreToFitWidth();
                        }
                    });
                    return;
                } catch {
                    // Fall through to report error
                }
            }
            clearScoreSurfaceOnly();
            hasRenderedScore = false;
            isRendering = false;
            renderError = err?.message || "Error al procesar AlphaTex";
        }
    }
</script>

<svelte:window on:resize={rescaleScoreToFitWidth} />

<div class="visual-panel">
    <div class="panel-header">
        <div class="header-title">
            <span>Kizuna Preview ({instrumentName || "No Track"})</span>
            {#if isRendering}
                <span class="render-badge syncing">Sincronizando partitura...</span>
            {:else if renderError}
                <span class="render-badge warning" title={renderError}>Sintaxis en edición</span>
            {:else if lastRenderedTex}
                <span class="render-badge ready">AlphaTab Listo</span>
            {/if}
        </div>

        {#if track && instrumentName !== "Vocals"}
            <div class="sync-transport-pill" class:playing={effectiveIsPlaying}>
                <button
                    type="button"
                    class="mini-sync-play"
                    on:click={handleTogglePlayClick}
                    title={effectiveIsPlaying ? "Pausar reproducción sincronizada" : "Reproducir en paralelo con la partitura"}
                >
                    {effectiveIsPlaying ? "⏸" : "▶"}
                </button>
                <span class="sync-metric">
                    Compás <strong>{syncState.currentBarIndex + 1}</strong>/{syncState.totalBars}
                </span>
                <span class="sync-divider">•</span>
                <span class="sync-metric">
                    Pulso <strong>{syncState.currentBeat}</strong>/{scoreMeta.timeSignature.num}
                </span>
                <span class="sync-divider">•</span>
                <span class="sync-metric">{activeBpm} BPM</span>
                <span class="sync-time">{formatTime(Math.floor(effectivePosition))}</span>
            </div>
        {/if}
    </div>

    {#if track && instrumentName !== "Vocals" && hasRenderedScore}
        <!-- Synchronized Measure Timeline Bar -->
        <div class="measure-sync-strip" aria-label="Navegador de compases sincronizado">
            <div class="measure-cells">
                {#each Array.from({ length: Math.min(totalBars, 32) }, (_, idx) => idx) as barIdx}
                    <button
                        type="button"
                        class="measure-cell"
                        class:active={barIdx === syncState.currentBarIndex && (effectiveIsPlaying || effectivePosition > 0)}
                        class:passed={barIdx < syncState.currentBarIndex && (effectiveIsPlaying || effectivePosition > 0)}
                        on:click={() => handleBarClick(barIdx)}
                        title={`Ir al compás ${barIdx + 1} (${formatTime(Math.floor(getTimestampForBar(barIdx, activeBpm, scoreMeta.timeSignature)))})`}
                    >
                        <span class="bar-num">{barIdx + 1}</span>
                        {#if barIdx === syncState.currentBarIndex && (effectiveIsPlaying || effectivePosition > 0)}
                            <div
                                class="bar-beat-fill"
                                style="width: {Math.round(syncState.barProgress * 100)}%"
                            />
                        {/if}
                    </button>
                {/each}
            </div>
            <div class="score-progress-track">
                <div
                    class="score-progress-fill"
                    style="width: {Math.round(syncState.scoreProgress * 100)}%"
                />
            </div>
        </div>
    {/if}

    {#if isPlainLyricsTrack}
        <LyricsViewer source={track?.data_content} />
    {/if}

    <!-- Keep AlphaTab container mounted across track switches to avoid costly re-initialization -->
    <div
        class="preview-container"
        class:hidden={isPlainLyricsTrack}
        bind:this={previewScrollContainer}
    >
        {#if !hasRenderedScore && !isRendering && !renderError}
            <div class="empty-score-placeholder">
                <span class="empty-score-title">Vista previa lista</span>
                <p class="empty-score-hint">
                    Escribe o pega notación AlphaTex o usa los botones <strong>+ Compás</strong>,
                    <strong>+ Acorde</strong> y <strong>| Barra</strong> para renderizar la partitura.
                </p>
            </div>
        {:else if !hasRenderedScore && !isRendering && renderError}
            <div class="empty-score-placeholder">
                <span class="empty-score-title">Sintaxis AlphaTex en edición</span>
                <p class="empty-score-hint">{renderError}</p>
            </div>
        {/if}
        <div class="score-scale-wrapper" bind:this={scoreContainer} />
    </div>
</div>

<style>
    .visual-panel {
        width: 60%;
        height: 100%;
        display: flex;
        flex-direction: column;
        background-color: #f5f5f5;
        color: black;
        min-width: 300px;
        overflow: hidden;
        box-sizing: border-box;
        flex: 1;
    }

    .panel-header {
        background-color: #e0e0e0;
        color: #333;
        padding: 8px 15px;
        border-bottom: 1px solid #ccc;
        font-weight: bold;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        flex-wrap: wrap;
    }

    .header-title {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .render-badge {
        font-size: 0.72rem;
        font-weight: 600;
        padding: 2px 8px;
        border-radius: 10px;
    }

    .render-badge.syncing {
        background-color: #fff3cd;
        color: #856404;
        border: 1px solid #ffeeba;
    }

    .render-badge.ready {
        background-color: #d4edda;
        color: #155724;
        border: 1px solid #c3e6cb;
    }

    .render-badge.warning {
        background-color: #f8d7da;
        color: #721c24;
        border: 1px solid #f5c6cb;
    }

    .sync-transport-pill {
        display: flex;
        align-items: center;
        gap: 6px;
        background-color: #1c1f21;
        color: #d7e2dc;
        padding: 3px 10px;
        border-radius: 14px;
        font-family: var(--lcd-font-ui, monospace);
        font-size: 0.76rem;
        border: 1px solid #3c4245;
    }

    .sync-transport-pill.playing {
        border-color: #ffc966;
        box-shadow: 0 0 8px rgba(255, 201, 102, 0.35);
    }

    .mini-sync-play {
        background: #2c3135;
        color: #ffc966;
        border: 1px solid #555b5e;
        border-radius: 50%;
        width: 20px;
        height: 20px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.65rem;
        cursor: pointer;
        padding: 0;
    }

    .mini-sync-play:hover {
        background: #3d4348;
        border-color: #ffc966;
    }

    .sync-metric strong {
        color: #ffc966;
    }

    .sync-divider {
        opacity: 0.45;
    }

    .sync-time {
        background: #0d1011;
        color: #50fa7b;
        padding: 1px 6px;
        border-radius: 4px;
        font-weight: bold;
    }

    .measure-sync-strip {
        background: #ececec;
        border-bottom: 1px solid #d0d0d0;
        padding: 5px 12px 4px;
        display: flex;
        flex-direction: column;
        gap: 4px;
        flex-shrink: 0;
    }

    .measure-cells {
        display: flex;
        align-items: center;
        gap: 4px;
        overflow-x: auto;
        padding-bottom: 2px;
    }

    .measure-cell {
        position: relative;
        min-width: 28px;
        height: 20px;
        background: #ffffff;
        border: 1px solid #bbb;
        border-radius: 3px;
        font-size: 0.7rem;
        font-weight: 600;
        color: #444;
        cursor: pointer;
        overflow: hidden;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 0 4px;
        transition: border-color 0.12s ease, background-color 0.12s ease;
    }

    .measure-cell:hover {
        border-color: #e69500;
        background: #fff9eb;
    }

    .measure-cell.passed {
        background: #f2efe9;
        color: #666;
    }

    .measure-cell.active {
        border-color: #d98200;
        background: #fff3d1;
        color: #111;
        box-shadow: 0 0 0 1px rgba(217, 130, 0, 0.35);
    }

    .bar-num {
        position: relative;
        z-index: 2;
    }

    .bar-beat-fill {
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        background: rgba(255, 184, 51, 0.45);
        z-index: 1;
        transition: width 0.06s linear;
    }

    .score-progress-track {
        width: 100%;
        height: 3px;
        background: #d5d5d5;
        border-radius: 2px;
        overflow: hidden;
    }

    .score-progress-fill {
        height: 100%;
        background: linear-gradient(90deg, #ff9900, #ffc966);
        transition: width 0.06s linear;
    }

    .preview-container {
        flex: 1;
        overflow-y: auto;
        overflow-x: hidden;
        font-family: "alphaTab";
        width: 100%;
        max-width: 100%;
        box-sizing: border-box;
        display: block;
        background-color: #fff;
        position: relative;
    }

    .preview-container.hidden {
        display: none;
    }

    .empty-score-placeholder {
        position: absolute;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        text-align: center;
        padding: 32px;
        color: #6b7280;
        font-family: var(--lcd-font-ui, sans-serif);
        pointer-events: none;
    }

    .empty-score-title {
        font-size: 1rem;
        font-weight: 700;
        color: #374151;
        margin-bottom: 6px;
    }

    .empty-score-hint {
        font-size: 0.85rem;
        max-width: 420px;
        line-height: 1.5;
        margin: 0;
    }

    .score-scale-wrapper {
        width: 100%;
        min-height: 100%;
        transform-origin: top left;
    }
</style>
