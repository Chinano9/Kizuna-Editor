<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import {
        StartQuickIdeaRecording,
        StopQuickIdeaRecording,
    } from "@wails/go/main/App";
    import { audioRecorderStore } from "@/features/audio-deck/stores/audioRecorderStore";
    import QuickIdeasInbox from "@/features/audio-deck/components/QuickIdeasInbox.svelte";
    import { syncStore } from "@/features/cloud-sync/stores/syncStore";
    import AuthModal from "@/features/cloud-sync/components/AuthModal.svelte";
    import CloudProjectsView from "@/features/cloud-sync/components/CloudProjectsView.svelte";
    import { toastStore } from "@shared/stores/toastStore";

    let activePanel: "library" | "ideas" | "cloud" = "library";
    let isBusy = false;
    let timerId: ReturnType<typeof setInterval> | undefined;
    let isDestroyed = false;
    let isAuthModalOpen = false;

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
            const idea = await StopQuickIdeaRecording();
            toastStore.success("Audio idea saved.");
            window.dispatchEvent(new CustomEvent("quick-idea-saved", { detail: idea }));
        } catch (error) {
            console.error("Failed to save quick idea recording:", error);
            toastStore.error(`Could not save the idea: ${errorMessage(error)}`);
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
            toastStore.info("Idea recording started. Click again to save.", 2500);

            if (isDestroyed) {
                await stopAndSaveQuickIdea();
            }
        } catch (error) {
            console.error("Failed to toggle quick idea recording:", error);
            audioRecorderStore.setRecording(false);
            audioRecorderStore.setElapsed(0);
            audioRecorderStore.setPeakLevel(0);
            toastStore.error(`Could not complete the recording: ${errorMessage(error)}`);
        } finally {
            isBusy = false;
        }
    }

    function formatElapsed(seconds: number): string {
        const minutes = Math.floor(seconds / 60).toString().padStart(2, "0");
        const remainingSeconds = (seconds % 60).toString().padStart(2, "0");
        return `${minutes}:${remainingSeconds}`;
    }

    onMount(() => {
        const handleSessionExpired = () => {
            isAuthModalOpen = true;
            toastStore.warning("Tu sesión ha expirado. Por favor inicia sesión de nuevo.");
        };
        window.addEventListener("kizuna-session-expired", handleSessionExpired);
        return () => {
            window.removeEventListener("kizuna-session-expired", handleSessionExpired);
        };
    });

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
            <button class="nav-item" class:active={activePanel === "library"} on:click={() => activePanel = "library"}>
                🎵 Library
            </button>
            <button class="nav-item" class:active={activePanel === "ideas"} on:click={() => activePanel = "ideas"}>
                🎙 Ideas Inbox
            </button>
            <button class="nav-item" class:active={activePanel === "cloud"} on:click={() => activePanel = "cloud"}>
                ☁️ Nube
            </button>
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
                    Stop and save ({formatElapsed($audioRecorderStore.secondsElapsed)})
                {:else}
                    ⚡ Record a Quick Idea
                {/if}
            </button>
        </footer>

        <div class="user-profile">
            {#if $syncStore.isAuthenticated}
                <div class="cloud-user-badge">
                    <span class="status-indicator online" title="Conectado a Kizuna Cloud"></span>
                    <span class="username" title={$syncStore.userEmail || ""}>
                        {$syncStore.user?.username || $syncStore.userEmail?.split("@")[0] || "Usuario"}
                    </span>
                    <button
                        type="button"
                        class="logout-link"
                        title="Cerrar sesión"
                        on:click={() => {
                            syncStore.clearAuth();
                            toastStore.info("Sesión cerrada.");
                        }}
                    >
                        Salir
                    </button>
                </div>
            {:else}
                <button
                    type="button"
                    class="login-trigger-btn"
                    on:click={() => (isAuthModalOpen = true)}
                >
                    <span class="cloud-icon">☁️</span>
                    <span>Conectar Nube</span>
                </button>
            {/if}
        </div>
    </aside>

    <main class="content-area">
        {#if activePanel === "library"}
            <slot />
        {:else if activePanel === "ideas"}
            <QuickIdeasInbox />
        {:else if activePanel === "cloud"}
            <CloudProjectsView onOpenAuthModal={() => (isAuthModalOpen = true)} />
        {/if}
    </main>

    <AuthModal bind:isOpen={isAuthModalOpen} />
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
        padding: 12px 14px;
        border-top: 1px solid #333;
        font-size: 0.85rem;
        color: #888;
        background: rgba(0, 0, 0, 0.2);
    }

    .login-trigger-btn {
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        background: transparent;
        border: 1px dashed #444;
        color: var(--lcd-text, #c8ff9f);
        padding: 8px 10px;
        border-radius: 4px;
        cursor: pointer;
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 1rem;
        transition: all 0.2s ease;
    }

    .login-trigger-btn:hover {
        border-color: var(--lcd-text, #c8ff9f);
        background: rgba(200, 255, 159, 0.08);
        box-shadow: 0 0 8px rgba(200, 255, 159, 0.15);
    }

    .cloud-icon {
        font-size: 1.1rem;
    }

    .cloud-user-badge {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 6px;
        width: 100%;
    }

    .status-indicator {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        flex-shrink: 0;
    }

    .status-indicator.online {
        background: #50fa7b;
        box-shadow: 0 0 6px #50fa7b;
    }

    .username {
        color: #e0e0e0;
        font-weight: 500;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-family: var(--lcd-font-main, monospace);
        font-size: 0.85rem;
        flex: 1;
        text-align: left;
    }

    .logout-link {
        background: transparent;
        border: none;
        color: #888;
        font-size: 0.8rem;
        cursor: pointer;
        padding: 2px 6px;
        border-radius: 3px;
        transition: all 0.15s ease;
    }

    .logout-link:hover {
        color: #ff5555;
        background: rgba(255, 85, 85, 0.1);
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
