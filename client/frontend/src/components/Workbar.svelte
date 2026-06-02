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

    import { onMount, onDestroy } from "svelte";
    import { get } from "svelte/store";
    import { song, songId } from "../stores/projectStore";
    import { activeTrackIndex } from "../stores/trackStore";
    import {
        StartRecording,
        StopRecording,
        GetAudioLevels,
        GetInputDevices,
        PlayAudio,
        PauseAudio,
        SeekAudio,
        SetAudioVolume,
        GetPlaybackPosition,
        GetPlaybackDuration,
        GetAudioVersionsForSong,
        GetAvailableDrivers,
        GetCurrentDriver,
        SetAudioDriver
    } from "../../wailsjs/go/main/App";
    import type { main } from "../../wailsjs/go/models";

    // Props from layout
    export let songID: number;
    export let activeTrack: main.Track | null;

    // --- State Machine ---
    let isRecording = false;
    let secondsElapsed = 0;
    let timerId: any = null;
    let vuIntervalId: any = null;
    let currentLevel = 0.0;

    // --- Hardware Input Devices & Drivers ---
    let devices: any[] = [];
    let selectedDeviceID = "default";
    let drivers: string[] = [];
    let selectedDriver = "WASAPI";

    // --- Target Recording Destination ---
    let selectedTargetTrackID = 0; // 0 = Global Project Audio (No track)

    // Sync target destination when active tab track changes (allows manual changes)
    let lastActiveTrackID = -1;
    $: {
        if (activeTrack && activeTrack.id !== lastActiveTrackID) {
            lastActiveTrackID = activeTrack.id;
            selectedTargetTrackID = activeTrack.id;
        }
    }

    let isSeeking = false;

    // --- Playback State ---
    let isPlaying = false;
    let volume = 0.8; // Default volume: 80%
    let playbackPosition = 0.0;
    let playbackDuration = 0.0;
    let playbackIntervalId: any = null;
    
    let takes: main.AudioVersion[] = [];
    let selectedTakePath = "";

    // Equalizer: 12 bars with phase offsets
    let equalizerBars = Array.from({ length: 12 }, (_, i) => ({
        height: 0,
        phase: i * 0.45,
    }));

    // --- Formatter ---
    // Displays elapsed recording time if recording, otherwise elapsed playback time
    $: formattedTime = formatTime(isRecording ? secondsElapsed : Math.floor(playbackPosition));
    // Calculate the exact synchronized maximum and progress ratio to eliminate timeline-fader visual drift
    $: timelineMax = isRecording ? 300 : (playbackDuration > 0 ? playbackDuration : 10);
    $: timelineProgress = timelineMax > 0
        ? Math.min(100, ((isRecording ? secondsElapsed : playbackPosition) / timelineMax) * 100)
        : 0;

    function formatTime(totalSecs: number): string {
        const hrs = Math.floor(totalSecs / 3600);
        const mins = Math.floor((totalSecs % 3600) / 60);
        const secs = totalSecs % 60;
        return [hrs, mins, secs]
            .map(v => v.toString().padStart(2, "0"))
            .join(":");
    }

    function getLedColorClass(segIdx: number): string {
        if (segIdx >= 8) return "red";
        if (segIdx >= 5) return "yellow";
        return "green";
    }

    // --- High-Fidelity Physics-Based VU Equalizer Polling ---
    let animationTime = 0.0;

    async function updateVUMeter() {
        if (!isRecording) return;
        try {
            const level = await GetAudioLevels();
            currentLevel = level;
            animationTime += 0.2;

            equalizerBars = equalizerBars.map((bar, i) => {
                let targetHeight = 0;
                if (isRecording) {
                    const wavePhase = Math.sin(animationTime + bar.phase);
                    targetHeight = level * 100 * (0.35 + 0.65 * Math.abs(wavePhase)) * (0.8 + 0.2 * Math.random());
                }

                // Smooth linear interpolation (lerp)
                const currentHeight = bar.height;
                const newHeight = currentHeight + (targetHeight - currentHeight) * 0.25;

                return {
                    ...bar,
                    height: Math.max(0, Math.min(100, newHeight)),
                };
            });
        } catch (err) {
            console.error("Error polling VU audio level:", err);
        }
    }

    // --- Life Cycle Queries ---
    onMount(async () => {
        await loadDrivers();
        await loadDevices();
        await loadTakes();
        
        // Push initial volume level to Go player
        try {
            await SetAudioVolume(volume);
        } catch (err) {
            console.error(err);
        }
    });

    async function loadDrivers() {
        try {
            drivers = await GetAvailableDrivers() || [];
            selectedDriver = await GetCurrentDriver() || "WASAPI";
        } catch (err) {
            console.error("Failed to query audio drivers:", err);
        }
    }

    async function handleDriverChange() {
        try {
            console.log("Changing driver to:", selectedDriver);
            await SetAudioDriver(selectedDriver);
            // Refresh devices list
            await loadDevices();
            selectedDeviceID = "default";
        } catch (err) {
            console.error("Failed to switch driver:", err);
            alert("Driver change failed: " + err);
            // Reload current active driver
            selectedDriver = await GetCurrentDriver();
            await loadDevices();
        }
    }

    async function loadDevices() {
        try {
            const list = await GetInputDevices();
            devices = list || [];
        } catch (err) {
            console.error("Failed to list microphones:", err);
        }
    }

    async function loadTakes() {
        if (!songID) return;
        try {
            const list = await GetAudioVersionsForSong(songID);
            takes = list || [];
            
            // Auto-select the latest recorded take if any
            if (takes.length > 0) {
                // Keep selected path if it still exists in the takes list, otherwise select the first
                const exists = takes.some(t => t.file_path === selectedTakePath);
                if (!exists) {
                    selectedTakePath = takes[0].file_path;
                }
            } else {
                selectedTakePath = "";
            }
        } catch (err) {
            console.error("Failed to query song takes:", err);
        }
    }

    // --- Recording Control ---
    async function toggleRecording() {
        if (!isRecording) {
            try {
                // Pause playback if active before recording starts
                if (isPlaying) {
                    await togglePlay();
                }

                // Start hardware capture in Go with selected device hex ID
                await StartRecording(songID, activeTrack ? activeTrack.name : "Global Sound", selectedDeviceID);
                isRecording = true;
                secondsElapsed = 0;

                timerId = setInterval(() => {
                    secondsElapsed++;
                }, 1000);

                vuIntervalId = setInterval(updateVUMeter, 30);
                console.log(`Native Go audio recording started on device ${selectedDeviceID}`);
            } catch (err) {
                console.error("Failed to start recording:", err);
                alert("Capture initialization failed: " + err);
            }
        } else {
            // STOP recording
            try {
                if (timerId) clearInterval(timerId);
                if (vuIntervalId) clearInterval(vuIntervalId);
                timerId = null;
                vuIntervalId = null;

                // Stop capture, write WAV, and register as AudioVersion in SQLite
                const newTake = await StopRecording(selectedTargetTrackID);
                isRecording = false;

                // Reset visualizer bars
                equalizerBars = equalizerBars.map(bar => ({ ...bar, height: 0 }));
                currentLevel = 0.0;

                // Reload takes list to select the newly recorded take
                await loadTakes();
                console.log("Audio take registered inside SQLite successfully:", newTake);
            } catch (err) {
                console.error("Failed to stop recording:", err);
                alert("Capture stop failed: " + err);
            }
        }
    }

    // --- Playback Controls ---
    async function togglePlay() {
        if (!selectedTakePath) {
            alert("No recorded audio take selected. Please record a take or sound first!");
            return;
        }

        try {
            if (!isPlaying) {
                // 1. Play WAV file natively in Go
                await PlayAudio(selectedTakePath);
                
                // 2. Fetch total WAV file duration from Go Player to scale seek fader
                const dur = await GetPlaybackDuration();
                playbackDuration = dur;
                isPlaying = true;
                
                // 3. Start high-frequency playback seek-head tracking
                if (playbackIntervalId) clearInterval(playbackIntervalId);
                playbackIntervalId = setInterval(async () => {
                    if (isPlaying && !isSeeking) {
                        try {
                            const pos = await GetPlaybackPosition();
                            if (!isSeeking) {
                                playbackPosition = pos;

                                // Handle audio completing on Go side
                                if (pos >= playbackDuration - 0.05 && playbackDuration > 0) {
                                    stopPlayback();
                                }
                            }
                        } catch (e) {
                            console.error(e);
                        }
                    }
                }, 30);
                console.log("Native Go playback initiated:", selectedTakePath);
            } else {
                // Pause playback in Go
                await PauseAudio();
                isPlaying = false;
                if (playbackIntervalId) {
                    clearInterval(playbackIntervalId);
                    playbackIntervalId = null;
                }
                console.log("Native Go playback paused.");
            }
        } catch (err) {
            console.error("Playback transport call failed:", err);
            alert("Playback failed: " + err);
        }
    }

    function stopPlayback() {
        isPlaying = false;
        playbackPosition = 0.0;
        if (playbackIntervalId) {
            clearInterval(playbackIntervalId);
            playbackIntervalId = null;
        }
    }

    // Drag-Seek visual updating (updates Svelte visual position in real time)
    function handleSeekInput(e: any) {
        isSeeking = true;
        playbackPosition = parseFloat(e.target.value);
    }

    // Drag-Seek execution (triggers native Go CGO seek only when dragging ends)
    async function handleSeekChange(e: any) {
        const targetSecs = parseFloat(e.target.value);
        playbackPosition = targetSecs;
        try {
            await SeekAudio(targetSecs);
        } catch (err) {
            console.error("Seek action failed:", err);
        }
        isSeeking = false;
    }

    // Volume Knob fader change
    async function handleVolumeChange(e: any) {
        const vol = parseFloat(e.target.value);
        volume = vol;
        try {
            await SetAudioVolume(vol);
        } catch (err) {
            console.error("Volume adjustment failed:", err);
        }
    }

    // Clean up threads on component destruction
    onDestroy(() => {
        if (timerId) clearInterval(timerId);
        if (vuIntervalId) clearInterval(vuIntervalId);
        if (playbackIntervalId) clearInterval(playbackIntervalId);
    });
</script>

<div class="workbar-container">
    <div class="control-deck">
        <!-- DECK 1: INPUT HARDWARE SELECTORS -->
        <div class="deck-section selectors-deck" style="width: 210px;">
            <div class="deck-input-row" style="display: flex; gap: 6px; width: 100%;">
                <div class="deck-input-group" style="width: 45%;">
                    <label class="deck-input-label">DRIVER</label>
                    <select class="deck-select" style="width: 100%;" bind:value={selectedDriver} on:change={handleDriverChange} disabled={isRecording || isPlaying}>
                        {#each drivers as drv}
                            <option value={drv}>{drv}</option>
                        {/each}
                    </select>
                </div>
                <div class="deck-input-group" style="width: 55%;">
                    <label class="deck-input-label">INPUT DEVICE</label>
                    <select class="deck-select" style="width: 100%;" bind:value={selectedDeviceID} disabled={isRecording}>
                        <option value="default">Default Mic</option>
                        {#each devices as dev}
                            <option value={dev.id}>{dev.name}</option>
                        {/each}
                    </select>
                </div>
            </div>
            <div class="deck-input-group margin-top-4">
                <label class="deck-input-label">RECORD TO</label>
                <select class="deck-select width-130" bind:value={selectedTargetTrackID} disabled={isRecording}>
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

        <!-- DECK 2: LCD DIGITAL CLOCK -->
        <div class="deck-section lcd-deck">
            <span class="deck-label">SESSION TIME</span>
            <div class="lcd-screen">
                <span class="lcd-bg-digits">88:88:88</span>
                <span class="lcd-text" class:pulsing={isRecording}>{formattedTime}</span>
            </div>
        </div>

        <!-- DECK 3: RECORD & PLAY TRANSPORT -->
        <div class="deck-section transport-deck">
            <div class="transport-controls">
                <!-- Retro play/pause transport button -->
                <button
                    class="play-btn"
                    class:playing={isPlaying}
                    on:click={togglePlay}
                    disabled={isRecording || !selectedTakePath}
                    title={isPlaying ? "Pause Playback" : "Play Selected Take"}
                    aria-label={isPlaying ? "Pause Playback" : "Play Selected Take"}
                >
                    <div class="play-icon" class:pause={isPlaying}></div>
                </button>

                <!-- Record bezel dial -->
                <div class="btn-bezel">
                    <button
                        class="record-btn"
                        class:recording={isRecording}
                        on:click={toggleRecording}
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

        <!-- DECK 4: LED VU METER EQUALIZER -->
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

        <!-- DECK 5: TAKES SELECTOR & VOLUME FADER -->
        <div class="deck-section playback-deck">
            <div class="deck-input-group align-right">
                <label class="deck-input-label">PLAY TAKE</label>
                <select class="deck-select width-130 text-right" bind:value={selectedTakePath} disabled={isRecording || isPlaying}>
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
            <div class="volume-control-group margin-top-4">
                <label class="deck-input-label align-right">MONITOR VOL</label>
                <div class="volume-slider-container">
                    <span class="volume-icon">🔊</span>
                    <input
                        type="range"
                        min="0"
                        max="1"
                        step="0.05"
                        class="volume-slider"
                        bind:value={volume}
                        on:input={handleVolumeChange}
                    />
                    <span class="volume-text">{Math.round(volume * 100)}%</span>
                </div>
            </div>
            <div class="panel-screw top-right"></div>
            <div class="panel-screw bottom-right"></div>
        </div>
    </div>

    <!-- TIMELINE SLIDING FADER -->
    <div class="timeline-deck">
        <div class="timeline-rail">
            <div class="timeline-progress-bar" style="width: {timelineProgress}%"></div>
            <!-- Slider range input for dragging and seeking directly -->
            <input
                type="range"
                min="0"
                max={timelineMax}
                step="0.1"
                value={isRecording ? secondsElapsed : playbackPosition}
                on:input={handleSeekInput}
                on:change={handleSeekChange}
                disabled={isRecording || !selectedTakePath}
                class="timeline-range-input"
            />
        </div>
    </div>
</div>

<style>
    /* --- Main Container: Brushed Dark Metal Panel --- */
    .workbar-container {
        position: fixed;
        bottom: 0;
        left: 0;
        right: 0;
        height: 85px;
        background: linear-gradient(
            to bottom,
            #2c2e30 0%,
            #17191a 8%,
            #0f1011 48%,
            #060607 92%,
            #151618 100%
        );
        border-top: 2px solid #3d4143;
        box-shadow:
            0 -4px 20px rgba(0, 0, 0, 0.8),
            inset 0 1px 0 rgba(255, 255, 255, 0.08);
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        padding-top: 6px;
        z-index: 1000;
        user-select: none;
        box-sizing: border-box;
    }

    /* --- Upper 5-Deck Layer --- */
    .control-deck {
        display: flex;
        flex-direction: row;
        justify-content: space-between;
        align-items: center;
        width: 98%;
        margin: 0 auto;
        height: 52px;
    }

    .deck-section {
        display: flex;
        flex-direction: column;
        justify-content: center;
        position: relative;
        height: 100%;
    }

    .deck-label {
        font-family: var(--lcd-font-ui), monospace;
        font-size: 0.72rem;
        color: #8f969b;
        letter-spacing: 0.12em;
        text-shadow: 0 1px 1px rgba(0, 0, 0, 0.6);
        margin-bottom: 2px;
        text-transform: uppercase;
        font-weight: bold;
    }

    /* --- Deck 1: Selector Deck --- */
    .selectors-deck {
        align-items: flex-start;
        padding-left: 18px;
        width: 175px;
    }

    .deck-input-group {
        display: flex;
        flex-direction: column;
        width: 100%;
    }

    .deck-input-label {
        font-family: var(--lcd-font-ui), monospace;
        font-size: 0.62rem;
        color: #8f969b;
        letter-spacing: 0.05em;
        font-weight: bold;
        text-shadow: 0 1px 1px rgba(0, 0, 0, 0.6);
        margin-bottom: 1.5px;
    }

    .deck-input-label.align-right {
        text-align: right;
    }

    /* Textured Dark Plastic console dropdown selectors */
    .deck-select {
        background-color: #0b0d0e;
        color: #c8ff9f; /* retro emerald green text */
        border: 1.5px solid #202b25;
        border-radius: 3px;
        font-family: var(--lcd-font-ui), monospace;
        font-size: 0.7rem;
        padding: 1px 4px;
        outline: none;
        cursor: pointer;
        box-shadow: inset 0 1px 3px rgba(0,0,0,0.8);
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

    .margin-top-4 {
        margin-top: 3px;
    }

    /* --- Deck 2: LCD Stopwatch Deck --- */
    .lcd-deck {
        align-items: center;
        width: 140px;
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
        font-family: var(--lcd-font-main), monospace;
        font-size: 1.6rem;
        color: #032110;
        letter-spacing: 2px;
        position: absolute;
        opacity: 0.22;
        pointer-events: none;
        z-index: 1;
    }

    .lcd-text {
        font-family: var(--lcd-font-main), monospace;
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

    /* --- Deck 3: Transport Controls --- */
    .transport-deck {
        align-items: center;
        width: 160px;
    }

    .transport-controls {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    /* Retro circular transport play button */
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
        border-color: transparent transparent transparent #ffc966; /* glowing amber */
        margin-left: 2px;
        transition: all 0.1s ease;
    }

    /* Pause double bar state */
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

    .record-label {
        font-size: 0.65rem;
        margin-top: 4px;
        color: #ffc966;
        text-shadow: var(--lcd-glow-soft);
    }

    /* --- Deck 4: LED Equalizer Deck --- */
    .vu-deck {
        align-items: center;
        width: 135px;
    }

    .meter-header {
        display: flex;
        justify-content: space-between;
        width: 110px;
        align-items: center;
        margin-bottom: 2px;
    }

    .level-indicator {
        font-family: var(--lcd-font-ui), monospace;
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

    /* --- Deck 5: Takes & Volume Monitor --- */
    .playback-deck {
        align-items: flex-end;
        padding-right: 18px;
        width: 175px;
    }

    .text-right {
        text-align: right;
    }

    .volume-control-group {
        display: flex;
        flex-direction: column;
        width: 100%;
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

    /* Metallic styled sliders */
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
        box-shadow: 0 1px 3px rgba(0,0,0,0.8);
        cursor: pointer;
    }

    .volume-text {
        font-family: var(--lcd-font-ui), monospace;
        font-size: 0.65rem;
        color: #c8ff9f;
        min-width: 25px;
        text-align: right;
        text-shadow: 0 0 4px rgba(200, 255, 159, 0.45);
    }

    /* --- Bottom Layer: Timeline Seek slider --- */
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

    /* Semi-transparent timeline range input laying over the rail, styled directly */
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

    /* Style the slider thumb natively to eliminate drag drift! */
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

    /* --- Aesthetic Panel Screws --- */
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
    .top-right { top: 0px; right: 5px; }
    .bottom-right { bottom: 2px; right: 5px; }
</style>
