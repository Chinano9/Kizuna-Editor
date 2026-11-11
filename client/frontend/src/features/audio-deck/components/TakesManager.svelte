<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import type { AudioVersion } from "@shared/types/wails";

    export let takes: AudioVersion[] = [];
    export let selectedTakePath: string = "";
    export let isRecording: boolean = false;
    export let isPlaying: boolean = false;

    const dispatch = createEventDispatcher<{
        changeTake: string;
    }>();

    function onChangeTake(e: Event) {
        const val = (e.target as HTMLSelectElement).value;
        dispatch("changeTake", val);
    }
</script>

<div class="deck-input-group align-right">
    <label for="deck-play-take" class="deck-input-label align-right">PLAY TAKE</label>
    <select
        id="deck-play-take"
        class="deck-select width-130 text-right"
        bind:value={selectedTakePath}
        on:change={onChangeTake}
        disabled={isRecording || isPlaying}
    >
        {#if takes.length === 0}
            <option value="">No takes recorded</option>
        {:else}
            {#each takes as take}
                <option value={take.file_path}>
                    {take.version_name.replace("Take: ", "")}
                </option>
            {/each}
        {/if}
    </select>
</div>

<style>
    .deck-input-group {
        display: flex;
        flex-direction: column;
        width: 100%;
    }

    .align-right {
        text-align: right;
        align-items: flex-end;
    }

    .deck-input-label {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.62rem;
        color: #8f969b;
        letter-spacing: 0.05em;
        font-weight: bold;
        text-shadow: 0 1px 1px rgba(0, 0, 0, 0.6);
        margin-bottom: 1.5px;
        text-transform: uppercase;
    }

    .deck-select {
        background-color: #0b0d0e;
        color: #c8ff9f;
        border: 1.5px solid #202b25;
        border-radius: 3px;
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.7rem;
        padding: 1px 4px;
        outline: none;
        cursor: pointer;
        box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.8);
        height: 18px;
        box-sizing: border-box;
    }

    .deck-select:focus {
        border-color: #c8ff9f;
        box-shadow: 0 0 4px rgba(200, 255, 159, 0.35);
    }

    .deck-select:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .width-130 {
        width: 130px;
    }

    .text-right {
        text-align: right;
    }
</style>
