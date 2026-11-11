<script lang="ts">
    /*
        Kizuna Editor - Audio Deck Workbar (Modular Orchestrator)
    */
    import { onMount, onDestroy } from "svelte";
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
        SetAudioDriver,
    } from "@wails/go/main/App";
    import type { Track, AudioVersion, AudioDevice } from "@shared/types/wails";
    import { formatTime } from "@shared/lib/formatters";

    import HardwareDeck from "./HardwareDeck.svelte";
    import SessionTimeDisplay from "./SessionTimeDisplay.svelte";
    import TransportControls from "./TransportControls.svelte";
    import VuEqualizer from "./VuEqualizer.svelte";
    import TakesManager from "./TakesManager.svelte";
    import VolumeSlider from "./VolumeSlider.svelte";
    import TimelineRail from "./TimelineRail.svelte";

    // Props from layout
    export let songID: number;
    export let activeTrack: Track | null;

    // --- State Machine ---
    let isRecording = false;
    let secondsElapsed = 0;
    let timerId: any = null;
    let vuIntervalId: any = null;
    let currentLevel = 0.0;

    // --- Hardware Input Devices & Drivers ---
    let devices: AudioDevice[] = [];
    let selectedDeviceID = "default";
    let drivers: string[] = [];
    let selectedDriver = "WASAPI";

    // --- Target Recording Destination ---
    let selectedTargetTrackID = 0; // 0 = Global Project Audio

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
    let volume = 0.8;
    let playbackPosition = 0.0;
    let playbackDuration = 0.0;
    let playbackIntervalId: any = null;

    let takes: AudioVersion[] = [];
    let selectedTakePath = "";

    // 12 Equalizer bars with phase offsets
    let equalizerBars = Array.from({ length: 12 }, (_, i) => ({
        height: 0,
        phase: i * 0.45,
    }));

    // Time & Progress calculations
    $: formattedTimeString = formatTime(isRecording ? secondsElapsed : Math.floor(playbackPosition));
    $: timelineMax = isRecording ? 300 : playbackDuration > 0 ? playbackDuration : 10;
    $: timelineProgress =
        timelineMax > 0
            ? Math.min(100, ((isRecording ? secondsElapsed : playbackPosition) / timelineMax) * 100)
            : 0;

    // --- High-Fidelity Physics-Based VU Equalizer Polling ---
    let animationTime = 0.0;

    async function updateVUMeter() {
        if (!isRecording) return;
        try {
            const level = await GetAudioLevels();
            currentLevel = level;
            animationTime += 0.2;

            equalizerBars = equalizerBars.map((bar) => {
                let targetHeight = 0;
                if (isRecording) {
                    const wavePhase = Math.sin(animationTime + bar.phase);
                    targetHeight =
                        level * 100 * (0.35 + 0.65 * Math.abs(wavePhase)) * (0.8 + 0.2 * Math.random());
                }

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

    // --- Life Cycle ---
    onMount(async () => {
        await loadDrivers();
        await loadDevices();
        await loadTakes();

        try {
            await SetAudioVolume(volume);
        } catch (err) {
            console.error(err);
        }
    });

    async function loadDrivers() {
        try {
            drivers = (await GetAvailableDrivers()) || [];
            selectedDriver = (await GetCurrentDriver()) || "WASAPI";
        } catch (err) {
            console.error("Failed to query audio drivers:", err);
        }
    }

    async function handleDriverChange() {
        try {
            await SetAudioDriver(selectedDriver);
            await loadDevices();
            selectedDeviceID = "default";
        } catch (err) {
            console.error("Failed to switch driver:", err);
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

            if (takes.length > 0) {
                const exists = takes.some((t) => t.file_path === selectedTakePath);
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
                if (isPlaying) {
                    await togglePlay();
                }

                await StartRecording(
                    songID,
                    activeTrack ? activeTrack.name : "Global Sound",
                    selectedDeviceID
                );
                isRecording = true;
                secondsElapsed = 0;

                timerId = setInterval(() => {
                    secondsElapsed++;
                }, 1000);

                vuIntervalId = setInterval(updateVUMeter, 30);
            } catch (err) {
                console.error("Failed to start recording:", err);
            }
        } else {
            try {
                if (timerId) clearInterval(timerId);
                if (vuIntervalId) clearInterval(vuIntervalId);
                timerId = null;
                vuIntervalId = null;

                await StopRecording(selectedTargetTrackID);
                isRecording = false;

                equalizerBars = equalizerBars.map((bar) => ({ ...bar, height: 0 }));
                currentLevel = 0.0;

                await loadTakes();
            } catch (err) {
                console.error("Failed to stop recording:", err);
            }
        }
    }

    // --- Playback Controls ---
    async function togglePlay() {
        if (!selectedTakePath) return;

        try {
            if (!isPlaying) {
                await PlayAudio(selectedTakePath);
                const dur = await GetPlaybackDuration();
                playbackDuration = dur;
                isPlaying = true;

                if (playbackIntervalId) clearInterval(playbackIntervalId);
                playbackIntervalId = setInterval(async () => {
                    if (isPlaying && !isSeeking) {
                        try {
                            const pos = await GetPlaybackPosition();
                            if (!isSeeking) {
                                playbackPosition = pos;
                                if (pos >= playbackDuration - 0.05 && playbackDuration > 0) {
                                    stopPlayback();
                                }
                            }
                        } catch (e) {
                            console.error(e);
                        }
                    }
                }, 30);
            } else {
                await PauseAudio();
                isPlaying = false;
                if (playbackIntervalId) {
                    clearInterval(playbackIntervalId);
                    playbackIntervalId = null;
                }
            }
        } catch (err) {
            console.error("Playback transport call failed:", err);
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

    function handleSeekInput(e: CustomEvent<number>) {
        isSeeking = true;
        playbackPosition = e.detail;
    }

    async function handleSeekChange(e: CustomEvent<number>) {
        const targetSecs = e.detail;
        playbackPosition = targetSecs;
        try {
            await SeekAudio(targetSecs);
        } catch (err) {
            console.error("Seek action failed:", err);
        }
        isSeeking = false;
    }

    async function handleVolumeChange(e: CustomEvent<number>) {
        const vol = e.detail;
        volume = vol;
        try {
            await SetAudioVolume(vol);
        } catch (err) {
            console.error("Volume adjustment failed:", err);
        }
    }

    onDestroy(() => {
        if (timerId) clearInterval(timerId);
        if (vuIntervalId) clearInterval(vuIntervalId);
        if (playbackIntervalId) clearInterval(playbackIntervalId);
    });
</script>

<div class="workbar-container">
    <div class="control-deck">
        <!-- Deck 1: Hardware & Driver Selectors -->
        <HardwareDeck
            {drivers}
            bind:selectedDriver
            {devices}
            bind:selectedDeviceID
            bind:selectedTargetTrackID
            {isRecording}
            {isPlaying}
            on:driverChange={handleDriverChange}
            on:deviceChange={(e) => (selectedDeviceID = e.detail)}
            on:targetTrackChange={(e) => (selectedTargetTrackID = e.detail)}
        />

        <!-- Deck 2: LCD Digital Clock -->
        <SessionTimeDisplay formattedTime={formattedTimeString} {isRecording} />

        <!-- Deck 3: Transport Controls -->
        <TransportControls
            {isRecording}
            {isPlaying}
            hasSelectedTake={!!selectedTakePath}
            on:togglePlay={togglePlay}
            on:toggleRecord={toggleRecording}
        />

        <!-- Deck 4: LED VU Meter -->
        <VuEqualizer {isRecording} {equalizerBars} />

        <!-- Deck 5: Takes & Volume Monitor -->
        <div class="deck-section playback-deck">
            <TakesManager
                {takes}
                bind:selectedTakePath
                {isRecording}
                {isPlaying}
                on:changeTake={(e) => (selectedTakePath = e.detail)}
            />
            <VolumeSlider
                {volume}
                on:volumeChange={handleVolumeChange}
            />
            <div class="panel-screw top-right"></div>
            <div class="panel-screw bottom-right"></div>
        </div>
    </div>

    <!-- Timeline Sliding Fader -->
    <TimelineRail
        {timelineProgress}
        {timelineMax}
        currentPosition={isRecording ? secondsElapsed : playbackPosition}
        disabled={isRecording || !selectedTakePath}
        on:seekInput={handleSeekInput}
        on:seekChange={handleSeekChange}
    />
</div>

<style>
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

    .playback-deck {
        align-items: flex-end;
        padding-right: 18px;
        width: 175px;
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

    .top-right { top: 0px; right: 5px; }
    .bottom-right { bottom: 2px; right: 5px; }
</style>
