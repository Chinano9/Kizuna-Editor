<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let timelineProgress: number = 0;
    export let timelineMax: number = 10;
    export let currentPosition: number = 0;
    export let disabled: boolean = false;

    const dispatch = createEventDispatcher<{
        seekInput: number;
        seekChange: number;
    }>();

    function onSeekInput(e: Event) {
        const val = parseFloat((e.target as HTMLInputElement).value);
        dispatch("seekInput", val);
    }

    function onSeekChange(e: Event) {
        const val = parseFloat((e.target as HTMLInputElement).value);
        dispatch("seekChange", val);
    }
</script>

<div class="timeline-deck">
    <div class="timeline-rail">
        <div class="timeline-progress-bar" style="width: {timelineProgress}%"></div>
        <input
            type="range"
            min="0"
            max={timelineMax}
            step="0.1"
            value={currentPosition}
            on:input={onSeekInput}
            on:change={onSeekChange}
            {disabled}
            class="timeline-range-input"
            aria-label="Timeline seek scrubber"
        />
    </div>
</div>

<style>
    .timeline-deck {
        width: 100%;
        height: 14px;
        display: flex;
        align-items: center;
        justify-content: center;
        background-color: #070809;
        border-top: 1px solid #1c1d1e;
        box-sizing: border-box;
    }

    .timeline-rail {
        position: relative;
        width: 96%;
        height: 3px;
        background-color: #1a1c1d;
        border-bottom: 1.5px solid #2d3032;
        border-radius: 1px;
        display: flex;
        align-items: center;
    }

    .timeline-progress-bar {
        position: absolute;
        left: 0;
        top: 0;
        height: 100%;
        background-color: rgba(255, 201, 102, 0.45);
        box-shadow: 0 0 5px rgba(255, 201, 102, 0.35);
        border-radius: 1px;
        pointer-events: none;
        z-index: 1;
    }

    .timeline-range-input {
        -webkit-appearance: none;
        appearance: none;
        width: 100%;
        background: transparent;
        margin: 0;
        outline: none;
        position: relative;
        z-index: 3;
        cursor: grab;
        height: 14px;
    }

    .timeline-range-input::-webkit-slider-runnable-track {
        background: transparent;
        border: none;
        height: 14px;
    }

    .timeline-range-input::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 10px;
        height: 14px;
        background: linear-gradient(
            to bottom,
            #7a8084 0%, #434749 42%,
            #ffc966 45%, #ffc966 55%,
            #434749 58%, #212223 100%
        );
        border: 1.5px solid #151617;
        border-radius: 1px;
        box-shadow:
            0 2px 4px rgba(0, 0, 0, 0.75),
            inset 0 1px 1px rgba(255, 255, 255, 0.3);
        cursor: grab;
    }

    .timeline-range-input::-moz-range-thumb {
        width: 10px;
        height: 14px;
        background: linear-gradient(
            to bottom,
            #7a8084 0%, #434749 42%,
            #ffc966 45%, #ffc966 55%,
            #434749 58%, #212223 100%
        );
        border: 1.5px solid #151617;
        border-radius: 1px;
        box-shadow:
            0 2px 4px rgba(0, 0, 0, 0.75),
            inset 0 1px 1px rgba(255, 255, 255, 0.3);
        cursor: grab;
    }

    .timeline-range-input:active {
        cursor: grabbing;
    }
</style>
