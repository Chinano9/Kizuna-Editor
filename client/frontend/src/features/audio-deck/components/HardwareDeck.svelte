<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { song } from "@/stores/projectStore";

    export let drivers: string[] = [];
    export let selectedDriver: string = "WASAPI";
    export let devices: any[] = [];
    export let selectedDeviceID: string = "default";
    export let selectedTargetTrackID: number = 0;
    export let isRecording: boolean = false;
    export let isPlaying: boolean = false;

    const dispatch = createEventDispatcher<{
        driverChange: void;
        deviceChange: string;
        targetTrackChange: number;
    }>();

    function onDriverChange() {
        dispatch("driverChange");
    }

    function onDeviceChange(e: Event) {
        const val = (e.target as HTMLSelectElement).value;
        dispatch("deviceChange", val);
    }

    function onTargetTrackChange(e: Event) {
        const val = parseInt((e.target as HTMLSelectElement).value, 10);
        dispatch("targetTrackChange", val);
    }
</script>

<div class="deck-section selectors-deck">
    <div class="deck-input-row">
        <div class="deck-input-group width-driver">
            <label for="deck-driver-select" class="deck-input-label">DRIVER</label>
            <select
                id="deck-driver-select"
                class="deck-select width-full"
                bind:value={selectedDriver}
                on:change={onDriverChange}
                disabled={isRecording || isPlaying}
            >
                {#each drivers as drv}
                    <option value={drv}>{drv}</option>
                {/each}
            </select>
        </div>
        <div class="deck-input-group width-device">
            <label for="deck-input-device" class="deck-input-label">INPUT DEVICE</label>
            <select
                id="deck-input-device"
                class="deck-select width-full"
                bind:value={selectedDeviceID}
                on:change={onDeviceChange}
                disabled={isRecording}
            >
                <option value="default">Default Mic</option>
                {#each devices as dev}
                    <option value={dev.id}>{dev.name}</option>
                {/each}
            </select>
        </div>
    </div>

    <div class="deck-input-group margin-top-3">
        <label for="deck-record-to" class="deck-input-label">RECORD TO</label>
        <select
            id="deck-record-to"
            class="deck-select width-130"
            bind:value={selectedTargetTrackID}
            on:change={onTargetTrackChange}
            disabled={isRecording}
        >
            <option value={0}>No Track (Global Audio)</option>
            {#if $song && $song.tracks}
                {#each $song.tracks as t}
                    <option value={t.id}>{t.name}</option>
                {/each}
            {/if}
        </select>
    </div>

    <div class="panel-screw top-left"></div>
    <div class="panel-screw bottom-left"></div>
</div>

<style>
    .deck-section {
        display: flex;
        flex-direction: column;
        justify-content: center;
        position: relative;
        height: 100%;
    }

    .selectors-deck {
        align-items: flex-start;
        padding-left: 18px;
        width: 210px;
    }

    .deck-input-row {
        display: flex;
        gap: 6px;
        width: 100%;
    }

    .width-driver {
        width: 45%;
    }

    .width-device {
        width: 55%;
    }

    .width-full {
        width: 100%;
    }

    .width-130 {
        width: 130px;
    }

    .margin-top-3 {
        margin-top: 3px;
    }

    .deck-input-group {
        display: flex;
        flex-direction: column;
        width: 100%;
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

    .panel-screw {
        position: absolute;
        width: 5px;
        height: 5px;
        background: radial-gradient(circle at 35% 35%, #5a5f62, #212324);
        border-radius: 50%;
        border: 0.5px solid #121314;
        box-shadow: 0 1px 1px rgba(255, 255, 255, 0.05);
        opacity: 0.85;
    }

    .top-left { top: 0px; left: 5px; }
    .bottom-left { bottom: 2px; left: 5px; }
</style>
