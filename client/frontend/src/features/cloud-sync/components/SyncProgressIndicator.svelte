<script lang="ts">
    /*
        Kizuna Editor - Indicador No Bloqueante de Progreso de Sincronización (Issue #23)
        Escucha los eventos Wails "sync:progress" y "sync:retry" con estética LCD Studio.
    */
    import { onMount, onDestroy } from "svelte";
    import { syncStore } from "../stores/syncStore.ts";
    import { toastStore } from "@shared/stores/toastStore";
    import { formatBytes } from "@shared/lib/formatters";
    import type { SyncProgressPayload, SyncRetryPayload } from "../types/sync.ts";
    import { EventsOn } from "../../../../wailsjs/runtime/runtime";

    let unlistenProgress: (() => void) | null = null;
    let unlistenRetry: (() => void) | null = null;
    let fadeOutTimeout: ReturnType<typeof setTimeout> | null = null;
    let isRetrying = false;
    let retryAttempt = 0;
    let retryMax = 3;

    function handleProgress(payload: SyncProgressPayload) {
        if (!payload) return;
        syncStore.setProgress(payload);

        if (payload.phase === "transferring") {
            isRetrying = false;
        }

        if (fadeOutTimeout) {
            clearTimeout(fadeOutTimeout);
            fadeOutTimeout = null;
        }

        if (payload.phase === "completed" || payload.phase === "failed") {
            fadeOutTimeout = setTimeout(() => {
                syncStore.setProgress(null);
                isRetrying = false;
            }, 1800);
        }
    }

    function handleRetry(payload: SyncRetryPayload) {
        if (!payload) return;
        isRetrying = true;
        retryAttempt = payload.attempt;
        retryMax = payload.max_attempts;

        // Disparar warning toast tal como exige el Issue #23
        toastStore.warning("Conexión inestable, reintentando...");

        if (fadeOutTimeout) {
            clearTimeout(fadeOutTimeout);
            fadeOutTimeout = null;
        }
    }

    onMount(() => {
        try {
            if (typeof EventsOn === "function") {
                unlistenProgress = EventsOn("sync:progress", (data: any) => handleProgress(data));
                unlistenRetry = EventsOn("sync:retry", (data: any) => handleRetry(data));
            }
        } catch (e) {
            console.warn("Wails runtime not available for sync events:", e);
        }

        const handleCustomProgress = (e: Event) => {
            handleProgress((e as CustomEvent).detail);
        };
        const handleCustomRetry = (e: Event) => {
            handleRetry((e as CustomEvent).detail);
        };

        if (typeof window !== "undefined") {
            window.addEventListener("kizuna-sync-progress", handleCustomProgress);
            window.addEventListener("kizuna-sync-retry", handleCustomRetry);
        }

        return () => {
            if (unlistenProgress) unlistenProgress();
            if (unlistenRetry) unlistenRetry();
            if (typeof window !== "undefined") {
                window.removeEventListener("kizuna-sync-progress", handleCustomProgress);
                window.removeEventListener("kizuna-sync-retry", handleCustomRetry);
            }
            if (fadeOutTimeout) clearTimeout(fadeOutTimeout);
        };
    });

    onDestroy(() => {
        if (unlistenProgress) unlistenProgress();
        if (unlistenRetry) unlistenRetry();
        if (fadeOutTimeout) clearTimeout(fadeOutTimeout);
    });

    $: progress = $syncStore.syncProgress;
</script>

{#if progress}
    <div
        class="sync-indicator-card"
        class:is-retrying={isRetrying || progress.phase === "retrying"}
        class:is-completed={progress.phase === "completed"}
        class:is-failed={progress.phase === "failed"}
        role="status"
        aria-live="polite"
    >
        <div class="header-row">
            <div class="op-status">
                {#if isRetrying || progress.phase === "retrying"}
                    <span class="status-icon pulse-amber">⚠</span>
                    <span class="status-text lcd-amber-glow">
                        REINTENTANDO ({retryAttempt || progress.attempt}/{retryMax || progress.max_attempts})
                    </span>
                {:else if progress.phase === "completed"}
                    <span class="status-icon success-glow">✓</span>
                    <span class="status-text">
                        {progress.operation === "push" ? "RESPALDO COMPLETADO" : "DESCARGA COMPLETADA"}
                    </span>
                {:else if progress.phase === "failed"}
                    <span class="status-icon error-glow">✕</span>
                    <span class="status-text">ERROR EN SINCRONIZACIÓN</span>
                {:else}
                    <span class="status-icon spin-icon">⚡</span>
                    <span class="status-text">
                        {progress.operation === "push" ? "SUBIENDO A KIZUNA CLOUD" : "DESCARGANDO PROYECTO"}
                    </span>
                {/if}
            </div>
            <span class="percentage-badge">
                {Math.round(progress.percentage || 0)}%
            </span>
        </div>

        <div class="progress-track" aria-hidden="true">
            <div
                class="progress-bar-fill"
                style="width: {Math.min(100, Math.max(0, progress.percentage || 0))}%;"
            ></div>
        </div>

        <div class="footer-row">
            {#if isRetrying}
                <span class="footer-note">Conexión inestable...</span>
            {:else if progress.total_bytes > 0}
                <span class="footer-bytes">
                    {formatBytes(progress.bytes_transferred)} / {formatBytes(progress.total_bytes)}
                </span>
            {:else}
                <span class="footer-bytes">
                    {formatBytes(progress.bytes_transferred)} transferidos
                </span>
            {/if}
            <span class="footer-phase">
                {progress.phase.toUpperCase()}
            </span>
        </div>
    </div>
{/if}

<style>
    .sync-indicator-card {
        position: fixed;
        bottom: 95px;
        right: 20px;
        width: 320px;
        background-color: var(--lcd-deck-bg, #090d0e);
        border: 1.5px solid var(--lcd-border, #243524);
        border-radius: 4px;
        padding: 10px 14px;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.75);
        font-family: var(--lcd-font-ui, "VT323", monospace);
        color: var(--lcd-text, #c8ff9f);
        z-index: 9998;
        pointer-events: auto;
        animation: slide-up 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    @keyframes slide-up {
        from {
            transform: translateY(20px);
            opacity: 0;
        }
        to {
            transform: translateY(0);
            opacity: 1;
        }
    }

    .header-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 1.15rem;
        letter-spacing: 0.5px;
    }

    .op-status {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .status-icon {
        font-size: 1.1rem;
    }

    .spin-icon {
        display: inline-block;
        animation: pulse-icon 1s infinite alternate;
    }

    @keyframes pulse-icon {
        from { transform: scale(0.9); opacity: 0.7; }
        to { transform: scale(1.15); opacity: 1; }
    }

    .pulse-amber {
        color: var(--lcd-amber, #ffc966);
        animation: amber-blink 0.8s infinite;
    }

    @keyframes amber-blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.3; }
    }

    .success-glow {
        color: #5cf29a;
        text-shadow: 0 0 6px #5cf29a;
    }

    .error-glow {
        color: #ff4d4d;
        text-shadow: 0 0 6px #ff4d4d;
    }

    .lcd-amber-glow {
        color: var(--lcd-amber, #ffc966);
        text-shadow: var(--lcd-amber-glow-soft, 0 0 8px rgba(255, 201, 102, 0.4));
    }

    .percentage-badge {
        font-size: 1.25rem;
        font-weight: bold;
        color: var(--lcd-text-soft, #a0e58d);
    }

    .progress-track {
        height: 6px;
        background: #111b14;
        border: 1px solid var(--lcd-border, #243524);
        border-radius: 3px;
        overflow: hidden;
    }

    .progress-bar-fill {
        height: 100%;
        background: linear-gradient(90deg, #1f3d28, #5cf29a);
        box-shadow: 0 0 8px rgba(92, 242, 154, 0.4);
        transition: width 0.15s ease-out;
    }

    .footer-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 0.95rem;
        color: var(--lcd-text-muted, #5a7c5a);
    }

    /* Retry Styling */
    .sync-indicator-card.is-retrying {
        border-color: var(--lcd-amber, #ffc966);
        box-shadow: 0 0 12px rgba(255, 201, 102, 0.3);
    }

    .sync-indicator-card.is-retrying .progress-bar-fill {
        background: linear-gradient(90deg, #3d2c10, #ffc966);
        box-shadow: 0 0 8px rgba(255, 201, 102, 0.4);
    }

    .sync-indicator-card.is-retrying .percentage-badge {
        color: var(--lcd-amber, #ffc966);
    }

    /* Completed Styling */
    .sync-indicator-card.is-completed {
        border-color: #5cf29a;
        box-shadow: 0 0 10px rgba(92, 242, 154, 0.25);
    }

    /* Failed Styling */
    .sync-indicator-card.is-failed {
        border-color: #ff4d4d;
        box-shadow: 0 0 10px rgba(255, 77, 77, 0.3);
    }
</style>
