<script lang="ts">
    import { onMount } from "svelte";
    // @ts-ignore: Suppress type errors if @coderline/alphatab types are missing or conflicting
    import { AlphaTabApi } from "@coderline/alphatab";
    import { injectBars } from "../lib/musicUtils";
    import { trackSource, autoBar } from "../stores/projectStore";

    let scoreContainer: HTMLElement;
    let api: any;

    onMount(() => {
        // Initialize AlphaTab engine targeting the container
        api = new AlphaTabApi(scoreContainer, {
            core: {
                tex: true,
                useWorkers: false, // Keep false for Electron/Wails compatibility usually
                engine: "svg",
            },
            display: {
                staveProfile: "Default",
                layoutMode: "page",
                padding: [20, 20, 20, 20],
            },
            player: { enablePlayer: false },
        });

        // Initial render
        renderMusic($trackSource, $autoBar);
    });

    // Reactive: Re-render when source or auto-formatting changes
    $: if (api) {
        renderMusic($trackSource, $autoBar);
    }

    function renderMusic(source: string, auto: boolean) {
        if (!api) return;

        // Inject automatic bar lines if enabled, otherwise render raw source
        const textToRender = auto ? injectBars(source) : source;
        api.tex(textToRender);
    }
</script>

<div class="visual-panel">
    <div class="panel-header">Kizuna Preview</div>
    <div class="score-wrapper" bind:this={scoreContainer}></div>
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
    }

    .panel-header {
        background-color: #e0e0e0;
        color: #333;
        padding: 10px 15px;
        border-bottom: 1px solid #ccc;
        font-weight: bold;
        flex-shrink: 0;
    }

    .score-wrapper {
        flex: 1;
        overflow-y: auto;
        padding: 20px;
        font-family: "alphaTab";
        width: 100%;
        box-sizing: border-box;
        display: block;
    }
</style>
