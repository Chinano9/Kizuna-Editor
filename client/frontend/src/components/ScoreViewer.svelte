<script lang="ts">
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

    // Reactive block to determine the instrument name whenever the track changes
    $: {
        if (track) {
            instrumentName = instrumentStore.getNameById(track.instrument_id);
        } else {
            instrumentName = "";
        }
    }

    function initializeApi() {
        // Guard against multiple initializations or initializing without a container
        if (api || !scoreContainer) return;

        api = new AlphaTabApi(scoreContainer, {
            core: {
                tex: true,
                useWorkers: false,
                engine: "svg",
            },
            display: {
                staveProfile: "Default", // Start with a default
                layoutMode: "page",
                padding: [20, 20, 20, 20],
            },
            player: { enablePlayer: false },
        });
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

    // Main reactive block to handle rendering logic
    $: {
        if (track) {
            if (instrumentName === "Vocals") {
                destroyApi(); // Clean up if it was previously active
            } else {
                // This IIFE (Immediately Invoked Function Expression) allows us to use async/await
                // to solve the race condition.
                (async () => {
                    // Wait for Svelte to update the DOM and render the #if block
                    await tick();

                    // Now that the DOM is updated, scoreContainer is guaranteed to exist.
                    initializeApi();

                    let newStaveProfile = "Default"; // For Guitar, Bass, etc.
                    if (instrumentName === "Piano") {
                        newStaveProfile = "Score";
                    }

                    // Only update settings if they have changed
                    if (api && newStaveProfile !== currentStaveProfile) {
                        api.settings.display.staveProfile = newStaveProfile;
                        currentStaveProfile = newStaveProfile;
                    }

                    // Finally, render the music content
                    renderMusic(track.data_content, $autoBar);
                })();
            }
        } else {
            // If there's no track, ensure we clean up and show nothing.
            destroyApi();
        }
    }

    function renderMusic(source: string, auto: boolean) {
        if (!api) return;

        if (typeof source !== "string" || source.trim() === "") {
            api.tex("");
            return;
        }

        const textToRender = auto ? injectBars(source) : source;
        api.tex(textToRender);
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
        <div class="preview-container" bind:this={scoreContainer} />
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
        font-family: "alphaTab";
        width: 100%;
        box-sizing: border-box;
        display: block;
        background-color: #fff; /* White background for score */
    }
</style>
