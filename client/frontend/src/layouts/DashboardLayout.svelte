<script lang="ts">
    import { onDestroy } from "svelte";
    import {
        StartQuickIdeaRecording,
        StopQuickIdeaRecording,
    } from "@wails/go/main/App";
    import { audioRecorderStore } from "@/features/audio-deck/stores/audioRecorderStore";
    import { toastStore } from "@shared/stores/toastStore";

    let isBusy = false;
    let timerId: ReturnType<typeof setInterval> | undefined;
    let isDestroyed = false;

    function clearTimer() {
        if (timerId !== undefined) {
            clearInterval(timerId);
            timerId = undefined;
        }
    }

    function errorMessage(error: unknown): string {
        return error instanceof Error ? error.message : String(error);
    }

    async function stopAndSaveQuickIdea() {
        clearTimer();
        try {
            await StopQuickIdeaRecording();
            toastStore.success("Idea de audio guardada.");
        } catch (error) {
            console.error("Failed to save quick idea recording:", error);
            toastStore.error(`No se pudo guardar la idea: ${errorMessage(error)}`);
        } finally {
            audioRecorderStore.setRecording(false);
            audioRecorderStore.setElapsed(0);
            audioRecorderStore.setPeakLevel(0);
        }
    }

    async function toggleQuickIdeaRecording() {
        if (isBusy) return;
        isBusy = true;

        try {
            if ($audioRecorderStore.isRecording) {
                await stopAndSaveQuickIdea();
                return;
            }

            await StartQuickIdeaRecording($audioRecorderStore.selectedDeviceID);
            audioRecorderStore.setElapsed(0);
            audioRecorderStore.setRecording(true);
            timerId = setInterval(() => {
                audioRecorderStore.setElapsed($audioRecorderStore.secondsElapsed + 1);
            }, 1000);
            toastStore.info("Grabación de idea iniciada. Pulsa de nuevo para guardar.", 2500);

            if (isDestroyed) {
                await stopAndSaveQuickIdea();
            }
        } catch (error) {
            console.error("Failed to toggle quick idea recording:", error);
            audioRecorderStore.setRecording(false);
            audioRecorderStore.setElapsed(0);
            audioRecorderStore.setPeakLevel(0);
            toastStore.error(`No se pudo completar la grabación: ${errorMessage(error)}`);
        } finally {
            isBusy = false;
        }
    }

    function formatElapsed(seconds: number): string {
        const minutes = Math.floor(seconds / 60).toString().padStart(2, "0");
        const remainingSeconds = (seconds % 60).toString().padStart(2, "0");
        return `${minutes}:${remainingSeconds}`;
    }

    onDestroy(() => {
        isDestroyed = true;
        clearTimer();
        if ($audioRecorderStore.isRecording && !isBusy) {
            void stopAndSaveQuickIdea();
        }
    });
</script>

<div class="layout-container">
    <aside class="sidebar">
        <div class="brand">
            <h2>Kizuna</h2>
        </div>

        <nav>
            <button class="nav-item active">🎵 Library</button>
            <button class="nav-item">💿 Albums (TODO)</button>
            <button class="nav-item">⚙️ Settings (TODO)</button>
        </nav>

        <footer>
            <button
                class:recording={$audioRecorderStore.isRecording}
                class="quick-idea"
                type="button"
                aria-pressed={$audioRecorderStore.isRecording}
                disabled={isBusy}
                on:click={toggleQuickIdeaRecording}
            >
                {#if $audioRecorderStore.isRecording}
                    <span class="recording-dot" aria-hidden="true"></span>
                    Detener y guardar ({formatElapsed($audioRecorderStore.secondsElapsed)})
                {:else}
                    ⚡ Record a Quick Idea
                {/if}
            </button>
        </footer>

        <div class="user-profile">
            <span>Local</span>
        </div>
    </aside>

    <main class="content-area">
        <slot />
    </main>
</div>

<style>
    .layout-container {
        display: flex;
        height: 100vh;
        width: 100vw;
        background-color: #1e1e1e;
        color: #e0e0e0;
    }

    /* SIDEBAR */
    .sidebar {
        width: 250px;
        background-color: #252526;
        border-right: 1px solid #333;
        display: flex;
        flex-direction: column;
        flex-shrink: 0;
    }

    .brand {
        padding: 20px;
        border-bottom: 1px solid #333;
    }
    .brand h2 {
        margin: 0;
        color: #4ec9b0;
        letter-spacing: 1px;
        font-size: 1.5rem;
    }

    nav {
        flex: 1;
        padding: 20px 10px;
        display: flex;
        flex-direction: column;
        gap: 5px;
    }

    .nav-item {
        background: transparent;
        border: none;
        color: #aaa;
        text-align: left;
        padding: 10px 15px;
        cursor: pointer;
        border-radius: 6px;
        font-size: 0.95rem;
        transition: all 0.2s;
    }
    .nav-item:hover {
        background-color: #2d2d2e;
        color: white;
    }
    .nav-item.active {
        background-color: #37373d;
        color: white;
        font-weight: bold;
        border-left: 3px solid #4ec9b0;
    }

    footer {
        padding: 15px;
        border-top: 1px solid #333;
    }

    .quick-idea {
        width: 100%;
        background-color: #0e639c;
        border: none;
        color: white;
        padding: 10px;
        font-size: 0.9rem;
        cursor: pointer;
        border-radius: 4px;
        transition: background 0.2s;
    }
    .quick-idea:disabled {
        cursor: wait;
        opacity: 0.75;
    }
    .quick-idea.recording {
        background-color: #a12626;
    }
    .quick-idea.recording:hover {
        background-color: #c43131;
    }
    .recording-dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        margin-right: 6px;
        border-radius: 50%;
        background: #fff;
        animation: pulse 1s infinite;
    }
    @keyframes pulse {
        50% { opacity: 0.35; }
    }

    .user-profile {
        padding: 15px;
        border-top: 1px solid #333;
        font-size: 0.85rem;
        color: #666;
        text-align: center;
    }

    /* CONTENT AREA */
    .content-area {
        flex: 1;
        overflow: hidden;
        position: relative;
        display: flex;
        flex-direction: column;
    }
</style>
