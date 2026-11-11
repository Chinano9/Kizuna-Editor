<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let isRecording: boolean = false;
    export let isPlaying: boolean = false;
    export let hasSelectedTake: boolean = false;

    const dispatch = createEventDispatcher<{
        togglePlay: void;
        toggleRecord: void;
    }>();

    function onPlayClick() {
        dispatch("togglePlay");
    }

    function onRecordClick() {
        dispatch("toggleRecord");
    }
</script>

<div class="deck-section transport-deck">
    <div class="transport-controls">
        <!-- Play / Pause Transport Button -->
        <button
            class="play-btn"
            class:playing={isPlaying}
            on:click={onPlayClick}
            disabled={isRecording || !hasSelectedTake}
            title={isPlaying ? "Pause Playback" : "Play Selected Take"}
            aria-label={isPlaying ? "Pause Playback" : "Play Selected Take"}
        >
            <div class="play-icon" class:pause={isPlaying}></div>
        </button>

        <!-- Record Bezel Dial Button -->
        <div class="btn-bezel">
            <button
                class="record-btn"
                class:recording={isRecording}
                on:click={onRecordClick}
                title={isRecording ? "Stop Recording" : "Start Recording"}
                aria-label={isRecording ? "Stop Recording" : "Start Recording"}
            >
                <div class="btn-rim">
                    <div class="record-icon" class:square={isRecording}></div>
                </div>
            </button>
        </div>
    </div>
    <span class="deck-label record-label">
        {isRecording ? "STOP RECORDING" : isPlaying ? "PLAYING TAKE" : "RECORD SESSION"}
    </span>
</div>

<style>
    .deck-section {
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        position: relative;
        height: 100%;
        width: 160px;
    }

    .transport-controls {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .play-btn {
        width: 28px;
        height: 28px;
        border-radius: 50%;
        background: linear-gradient(135deg, #4b4e50 0%, #1e2021 65%, #08090a 100%);
        border: 2px solid #5a5e60;
        cursor: pointer;
        padding: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow:
            0 2px 3px rgba(0, 0, 0, 0.8),
            inset 0 1px 2px rgba(255, 255, 255, 0.3);
        outline: none;
        transition: transform 0.1s ease, box-shadow 0.1s ease;
    }

    .play-btn:hover:not(:disabled) {
        transform: scale(1.05);
        border-color: #ffc966;
        box-shadow:
            0 2px 5px rgba(255, 201, 102, 0.3),
            inset 0 1px 2px rgba(255, 255, 255, 0.4);
    }

    .play-btn:active:not(:disabled) {
        transform: scale(0.95);
        box-shadow:
            0 1px 1px rgba(0, 0, 0, 0.9),
            inset 0 1px 3px rgba(0, 0, 0, 0.8);
    }

    .play-btn:disabled {
        opacity: 0.3;
        cursor: not-allowed;
    }

    .play-icon {
        width: 0;
        height: 0;
        border-style: solid;
        border-width: 5px 0 5px 8px;
        border-color: transparent transparent transparent #ffc966;
        margin-left: 2px;
        transition: all 0.1s ease;
    }

    .play-icon.pause {
        border-style: double;
        border-width: 0 0 0 8px;
        border-color: transparent transparent transparent #ffc966;
        width: 8px;
        height: 9px;
        margin-left: 0;
    }

    .btn-bezel {
        width: 44px;
        height: 44px;
        background: linear-gradient(135deg, #1b1d1e 0%, #303336 40%, #151617 100%);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow:
            0 3px 5px rgba(0, 0, 0, 0.6),
            inset 0 1px 2px rgba(255, 255, 255, 0.1),
            0 0 0 1px #0e0f10;
        margin-top: -16px;
        z-index: 1010;
    }

    .record-btn {
        width: 34px;
        height: 34px;
        border-radius: 50%;
        background: radial-gradient(circle at 35% 35%, #ff4d4d 0%, #bf0000 65%, #660000 100%);
        border: 2px solid #6b7073;
        cursor: pointer;
        padding: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow:
            0 2px 4px rgba(0, 0, 0, 0.8),
            inset 0 1.5px 3px rgba(255, 255, 255, 0.45);
        outline: none;
        transition:
            transform 0.15s cubic-bezier(0.2, 0.8, 0.2, 1),
            box-shadow 0.15s ease;
    }

    .record-btn:hover {
        transform: scale(1.04);
        background: radial-gradient(circle at 35% 35%, #ff6666 0%, #d40000 65%, #800000 100%);
        box-shadow:
            0 3px 8px rgba(255, 0, 0, 0.35),
            inset 0 1.5px 3px rgba(255, 255, 255, 0.5);
    }

    .record-btn:active {
        transform: scale(0.96);
        box-shadow:
            0 1px 2px rgba(0, 0, 0, 0.9),
            inset 0 2px 5px rgba(0, 0, 0, 0.8);
    }

    .record-btn.recording {
        border-color: #ff9999;
        animation: button-glow 1.5s infinite alternate;
    }

    @keyframes button-glow {
        0% { box-shadow: 0 0 8px rgba(255, 0, 0, 0.4), inset 0 1px 3px rgba(255, 255, 255, 0.4); }
        100% { box-shadow: 0 0 16px rgba(255, 0, 0, 0.85), inset 0 1px 3px rgba(255, 255, 255, 0.6); }
    }

    .btn-rim {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 100%;
        height: 100%;
    }

    .record-icon {
        width: 10px;
        height: 10px;
        background-color: white;
        border-radius: 50%;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
        transition:
            border-radius 0.25s cubic-bezier(0.4, 0, 0.2, 1),
            transform 0.2s ease;
    }

    .record-icon.square {
        border-radius: 2px;
        width: 9px;
        height: 9px;
        transform: scale(1.05);
    }

    .deck-label {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        letter-spacing: 0.12em;
        text-transform: uppercase;
        font-weight: bold;
    }

    .record-label {
        font-size: 0.65rem;
        margin-top: 4px;
        color: #ffc966;
        text-shadow: var(--lcd-glow-soft, 0 0 12px rgba(255, 201, 102, 0.4));
    }
</style>
