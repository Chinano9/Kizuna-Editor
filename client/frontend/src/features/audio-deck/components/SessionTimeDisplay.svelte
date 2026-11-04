<script lang="ts">
    export let formattedTime: string = "00:00:00";
    export let isRecording: boolean = false;
</script>

<div class="deck-section lcd-deck">
    <span class="deck-label">SESSION TIME</span>
    <div class="lcd-screen">
        <span class="lcd-bg-digits">88:88:88</span>
        <span class="lcd-text" class:pulsing={isRecording}>{formattedTime}</span>
    </div>
</div>

<style>
    .deck-section {
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        position: relative;
        height: 100%;
        width: 140px;
    }

    .deck-label {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.72rem;
        color: #8f969b;
        letter-spacing: 0.12em;
        text-shadow: 0 1px 1px rgba(0, 0, 0, 0.6);
        margin-bottom: 2px;
        text-transform: uppercase;
        font-weight: bold;
    }

    .lcd-screen {
        position: relative;
        background-color: #0b110e;
        border: 2px solid #202b25;
        border-radius: 4px;
        padding: 2px 10px;
        height: 28px;
        width: 110px;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow:
            inset 0 2px 5px rgba(0, 0, 0, 0.9),
            0 1px 0 rgba(255, 255, 255, 0.05);
        overflow: hidden;
    }

    .lcd-bg-digits {
        font-family: var(--lcd-font-main, "VT323", monospace);
        font-size: 1.6rem;
        color: #032110;
        letter-spacing: 2px;
        position: absolute;
        opacity: 0.22;
        pointer-events: none;
        z-index: 1;
    }

    .lcd-text {
        font-family: var(--lcd-font-main, "VT323", monospace);
        font-size: 1.6rem;
        color: #00ff66;
        text-shadow: 0 0 6px rgba(0, 255, 102, 0.75);
        letter-spacing: 2px;
        line-height: 1;
        position: relative;
        z-index: 2;
    }

    .lcd-text.pulsing {
        animation: lcd-flicker 2s infinite alternate;
    }

    @keyframes lcd-flicker {
        0% { opacity: 0.95; filter: drop-shadow(0 0 4px rgba(0, 255, 102, 0.6)); }
        100% { opacity: 1; filter: drop-shadow(0 0 8px rgba(0, 255, 102, 0.9)); }
    }
</style>
