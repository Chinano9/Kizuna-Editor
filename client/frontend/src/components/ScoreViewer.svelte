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
    import { onMount, onDestroy, tick } from "svelte";
    // @ts-ignore
    import { AlphaTabApi } from "@coderline/alphatab";

    import { injectBars } from "../lib/musicUtils";
    import { autoBar } from "../stores/projectStore";
    import { instrumentStore } from "@/stores/instrumentStore";
    import type { main } from "../../../wailsjs/go/models";

    import LyricsViewer from "./LyricsViewer.svelte";

    export let track: main.Track | null = null;

    let scoreContainer: HTMLElement;
    let api: any;
    let instrumentName = "";
    let currentStaveProfile = "";
    let currentScale = 1;

    function rescaleScoreToFitWidth() {
        if (!scoreContainer) return;
        const svg = scoreContainer.querySelector("svg") as SVGSVGElement | null;
        if (!svg) return;

        const bbox = svg.getBBox();
        const svgWidth = bbox.width;
        const svgHeight = bbox.height;
        if (!svgWidth || !svgHeight) return;

        const container = scoreContainer.parentElement as HTMLElement | null;
        if (!container) return;

        const availableWidth = container.clientWidth;
        if (!availableWidth) return;

        const scale = Math.min(1, availableWidth / svgWidth);
        currentScale = scale;

        scoreContainer.style.transformOrigin = "top left";
        scoreContainer.style.transform = `scale(${scale})`;
        scoreContainer.style.width = `${svgWidth}px`;
        scoreContainer.style.height = `${svgHeight}px`;
    }

    // Reactive block to determine the instrument name whenever el track cambia
    $: {
        if (track) {
            instrumentName = instrumentStore.getNameById(track.instrument_id);
        } else {
            instrumentName = "";
        }
    }

    function initializeApi(staveProfile: "Default" | "Score" = "Default") {
        // Guard against múltiples inicializaciones o inicializar sin contenedor
        if (api || !scoreContainer) return;

        api = new AlphaTabApi(scoreContainer, {
            core: {
                tex: true,
                useWorkers: false,
                engine: "svg",
                fontDirectory: "/font/", // Path to the font directory
            },
            display: {
                staveProfile, // Usamos el perfil recibido
                layoutMode: "page",
                padding: [20, 20, 20, 20],
            },
            player: { enablePlayer: false },
        });
    }

    function looksLikeAlphaTex(source: string): boolean {
        const trimmed = source.trim();
        if (!trimmed) return false;

        // Heurística básica: directivas AlphaTex típicas al inicio
        if (
            trimmed.startsWith("score ") ||
            trimmed.startsWith("part ") ||
            trimmed.startsWith("tempo ")
        ) {
            return true;
        }

        // Otra heurística: líneas que empiezan con ':' seguidas de una duración válida
        const alphaLines = trimmed
            .split("\n")
            .map((l) => l.trim())
            .filter((l) => l.startsWith(":"));

        const durationRegex = /^: ?(-4|-2|1|2|4|8|16|32|64|128|256)\b/;
        if (alphaLines.some((l) => durationRegex.test(l))) {
            return true;
        }

        return false;
    }

    function destroyApi() {
        if (api) {
            api.destroy();
            api = null;
        }
    }

    // When the component is removed from the DOM, clean up the AlphaTab instance
    onDestroy(() => {
        destroyApi();
    });

    // Main reactive block to handle AlphaTab lifecycle and rendering
    $: updateAlphaTab(track, instrumentName, $autoBar);

    async function updateAlphaTab(
        currentTrack: main.Track | null,
        currentInstrumentName: string,
        auto: boolean,
    ) {
        if (!currentTrack) {
            destroyApi();
            instrumentName = "";
            return;
        }

        if (currentInstrumentName === "Vocals") {
            // Para voces usamos LyricsViewer, no AlphaTab
            destroyApi();
            return;
        }

        const content = currentTrack.data_content ?? "";
        const rawText = auto ? injectBars(content) : content;

        // Si aún no parece AlphaTex, no inicializamos AlphaTab ni intentamos renderizar
        if (!looksLikeAlphaTex(rawText)) {
            destroyApi();
            return;
        }

        // En este punto ya tenemos algo que parece AlphaTex
        await tick();

        let newStaveProfile: "Default" | "Score" = "Default"; // For Guitar, Bass, etc.
        if (currentInstrumentName === "Piano") {
            newStaveProfile = "Score";
        }

        initializeApi(newStaveProfile);
        currentStaveProfile = newStaveProfile;

        // Finalmente, renderizamos el contenido AlphaTex
        renderMusic(rawText);
    }

    function renderMusic(textToRender: string) {
        if (!api) return;

        const trimmed = textToRender.trim();
        if (!trimmed) {
            api.tex("");
            return;
        }

        // En este punto asumimos que looksLikeAlphaTex(trimmed) ya fue true
        api.tex(trimmed);

        // Reescalar después de que AlphaTab haya renderizado el SVG
        tick().then(() => {
            rescaleScoreToFitWidth();
        });
    }

    import "../styles/preview-theme.css";
</script>

<div class="visual-panel">
    <div class="panel-header">
        Kizuna Preview ({instrumentName || "No Track"})
    </div>

    {#if instrumentName === "Vocals"}
        <LyricsViewer source={track?.data_content} />
    {:else}
        <!-- This container is for AlphaTab -->
        <div class="preview-container">
            <div class="score-scale-wrapper" bind:this={scoreContainer} />
        </div>
    {/if}
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
    }
    .panel-header {
        background-color: #e0e0e0;
        color: #333;
        padding: 10px 15px;
        border-bottom: 1px solid #ccc;
        font-weight: bold;
        flex-shrink: 0;
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
        background-color: #fff; /* White background for score */
        position: relative;
    }

    .score-scale-wrapper {
        transform-origin: top left;
        will-change: transform;
    }
</style>
