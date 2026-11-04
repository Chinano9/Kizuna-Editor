<script lang="ts">
    export let isRecording: boolean = false;
    export let equalizerBars: Array<{ height: number; phase: number }> = [];

    function getLedColorClass(segIdx: number): string {
        if (segIdx >= 8) return "red";
        if (segIdx >= 5) return "yellow";
        return "green";
    }
</script>

<div class="deck-section vu-deck">
    <div class="meter-header">
        <span class="deck-label">INPUT LEVEL</span>
        <div class="level-indicator" class:active={isRecording}>
            <span class="led-indicator-dot" class:active={isRecording}></span>
            {isRecording ? "ACTIVE" : "STANDBY"}
        </div>
    </div>
    <div class="vu-equalizer">
        {#each equalizerBars as bar}
            <div class="eq-bar">
                <div class="led-grid">
                    {#each [9, 8, 7, 6, 5, 4, 3, 2, 1, 0] as segIdx}
                        <div
                            class="led-seg {getLedColorClass(segIdx)}"
                            class:active={segIdx * 10 < bar.height}
                        ></div>
                    {/each}
                </div>
            </div>
        {/each}
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
        width: 135px;
    }

    .meter-header {
        display: flex;
        justify-content: space-between;
        width: 110px;
        align-items: center;
        margin-bottom: 2px;
    }

    .deck-label {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.72rem;
        color: #8f969b;
        letter-spacing: 0.12em;
        text-shadow: 0 1px 1px rgba(0, 0, 0, 0.6);
        text-transform: uppercase;
        font-weight: bold;
    }

    .level-indicator {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.6rem;
        color: #727a80;
        display: flex;
        align-items: center;
        gap: 3px;
        letter-spacing: 0.05em;
    }

    .level-indicator.active {
        color: #ff5555;
        font-weight: bold;
    }

    .led-indicator-dot {
        width: 4px;
        height: 4px;
        border-radius: 50%;
        background-color: #310808;
        display: inline-block;
        transition: background-color 0.2s ease;
    }

    .led-indicator-dot.active {
        background-color: #ff3333;
        box-shadow: 0 0 4px #ff3333;
    }

    .vu-equalizer {
        display: flex;
        flex-direction: row;
        gap: 2.2px;
        background-color: #080a0b;
        border: 2px solid #1a1e20;
        border-radius: 4px;
        padding: 3px;
        box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.9);
        height: 28px;
        width: 110px;
        box-sizing: border-box;
    }

    .eq-bar {
        display: flex;
        flex-direction: column;
        justify-content: flex-end;
        height: 100%;
        width: 6px;
    }

    .led-grid {
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        height: 100%;
        width: 100%;
    }

    .led-seg {
        flex: 1;
        margin: 0.2px 0;
        border-radius: 0.5px;
        opacity: 0.15;
        transition: opacity 0.04s ease, filter 0.04s ease;
    }

    .led-seg.green { background-color: #00ff55; }
    .led-seg.yellow { background-color: #ffcc00; }
    .led-seg.red { background-color: #ff2200; }

    .led-seg.green.active {
        opacity: 1;
        filter: drop-shadow(0 0 1.5px #00ff55) brightness(1.1);
    }

    .led-seg.yellow.active {
        opacity: 1;
        filter: drop-shadow(0 0 1.5px #ffcc00) brightness(1.1);
    }

    .led-seg.red.active {
        opacity: 1;
        filter: drop-shadow(0 0 2px #ff2200) brightness(1.15);
    }
</style>
