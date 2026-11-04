<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let volume: number = 0.8;

    const dispatch = createEventDispatcher<{
        volumeChange: number;
    }>();

    function onVolumeInput(e: Event) {
        const val = parseFloat((e.target as HTMLInputElement).value);
        volume = val;
        dispatch("volumeChange", val);
    }
</script>

<div class="volume-control-group margin-top-3">
    <label for="deck-monitor-vol" class="deck-input-label align-right">MONITOR VOL</label>
    <div class="volume-slider-container">
        <span class="volume-icon">🔊</span>
        <input
            id="deck-monitor-vol"
            type="range"
            min="0"
            max="1"
            step="0.05"
            class="volume-slider"
            bind:value={volume}
            on:input={onVolumeInput}
        />
        <span class="volume-text">{Math.round(volume * 100)}%</span>
    </div>
</div>

<style>
    .volume-control-group {
        display: flex;
        flex-direction: column;
        width: 100%;
    }

    .margin-top-3 {
        margin-top: 3px;
    }

    .align-right {
        text-align: right;
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

    .volume-slider-container {
        display: flex;
        align-items: center;
        gap: 4px;
        height: 18px;
    }

    .volume-icon {
        font-size: 0.7rem;
        color: #7a8286;
    }

    .volume-slider {
        -webkit-appearance: none;
        appearance: none;
        flex: 1;
        height: 3px;
        background: #151617;
        border-bottom: 1px solid #2d3032;
        border-radius: 1px;
        outline: none;
    }

    .volume-slider::-webkit-scrollbar {
        display: none;
    }

    .volume-slider::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 8px;
        height: 12px;
        background: linear-gradient(135deg, #7a8084 0%, #434749 50%, #212223 100%);
        border: 1px solid #0f1011;
        border-radius: 1px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.8);
        cursor: pointer;
    }

    .volume-slider::-moz-range-thumb {
        width: 8px;
        height: 12px;
        background: linear-gradient(135deg, #7a8084 0%, #434749 50%, #212223 100%);
        border: 1px solid #0f1011;
        border-radius: 1px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.8);
        cursor: pointer;
    }

    .volume-text {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.65rem;
        color: #c8ff9f;
        min-width: 25px;
        text-align: right;
        text-shadow: 0 0 4px rgba(200, 255, 159, 0.45);
    }
</style>
