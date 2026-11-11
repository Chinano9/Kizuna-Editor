<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import type { ToastType } from "@shared/stores/toastStore";

    export let id: string;
    export let message: string;
    export let type: ToastType = "info";

    const dispatch = createEventDispatcher<{ dismiss: string }>();

    function dismiss() {
        dispatch("dismiss", id);
    }

    const typeIcons: Record<ToastType, string> = {
        success: "✓",
        error: "✕",
        warning: "⚠",
        info: "ℹ",
    };
</script>

<div class="toast-item type-{type}" role="alert" aria-live="polite">
    <span class="toast-icon">{typeIcons[type] || "•"}</span>
    <span class="toast-message">{message}</span>
    <button
        type="button"
        class="toast-close"
        on:click={dismiss}
        aria-label="Dismiss notification"
    >
        ✕
    </button>
</div>

<style>
    .toast-item {
        display: flex;
        align-items: center;
        gap: 10px;
        min-width: 280px;
        max-width: 440px;
        padding: 8px 14px;
        background-color: var(--lcd-deck-bg, #090d0e);
        border-radius: 4px;
        box-shadow: 0 4px 14px rgba(0, 0, 0, 0.75);
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 1.15rem;
        letter-spacing: 0.5px;
        animation: slide-in 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        user-select: text;
    }

    @keyframes slide-in {
        from {
            transform: translateX(30px);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }

    /* Type styles with studio glows */
    .type-success {
        border: 1.5px solid #33cc66;
        color: #c8ff9f;
        box-shadow: 0 0 10px rgba(51, 204, 102, 0.25);
    }
    .type-success .toast-icon {
        color: #33cc66;
    }

    .type-error {
        border: 1.5px solid #ff4d4d;
        color: #ff9999;
        box-shadow: 0 0 10px rgba(255, 77, 77, 0.25);
    }
    .type-error .toast-icon {
        color: #ff4d4d;
    }

    .type-warning {
        border: 1.5px solid #ffc966;
        color: #ffdd99;
        box-shadow: 0 0 10px rgba(255, 201, 102, 0.25);
    }
    .type-warning .toast-icon {
        color: #ffc966;
    }

    .type-info {
        border: 1.5px solid #5cf29a;
        color: #c8ff9f;
        box-shadow: 0 0 10px rgba(92, 242, 154, 0.25);
    }
    .type-info .toast-icon {
        color: #5cf29a;
    }

    .toast-icon {
        font-weight: bold;
        font-size: 1.25rem;
        flex-shrink: 0;
    }

    .toast-message {
        flex: 1;
        line-height: 1.3;
    }

    .toast-close {
        background: transparent;
        border: none;
        color: inherit;
        opacity: 0.6;
        cursor: pointer;
        padding: 2px 4px;
        font-size: 0.95rem;
        border-radius: 2px;
        flex-shrink: 0;
        transition: opacity 0.15s ease;
    }
    .toast-close:hover {
        opacity: 1;
    }
</style>
