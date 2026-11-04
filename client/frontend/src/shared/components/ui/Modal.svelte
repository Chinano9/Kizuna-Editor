<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";

    export let isOpen: boolean = false;
    export let title: string = "";
    export let maxWidth: string = "500px";

    const dispatch = createEventDispatcher<{ close: void }>();

    function close() {
        dispatch("close");
    }

    function handleKeyDown(e: KeyboardEvent) {
        if (e.key === "Escape" && isOpen) {
            close();
        }
    }
</script>

<svelte:window on:keydown={handleKeyDown} />

{#if isOpen}
    <div
        class="modal-backdrop"
        role="presentation"
        on:click|self={close}
    >
        <div
            class="modal-window"
            style="max-width: {maxWidth};"
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
        >
            <div class="modal-header">
                <h3 id="modal-title" class="modal-title">{title}</h3>
                <button
                    type="button"
                    class="close-button"
                    aria-label="Close dialog"
                    on:click={close}
                >
                    ✕
                </button>
            </div>

            <div class="modal-body">
                <slot />
            </div>

            {#if $$slots.footer}
                <div class="modal-footer">
                    <slot name="footer" />
                </div>
            {/if}
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background-color: rgba(0, 0, 0, 0.75);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        backdrop-filter: blur(2px);
    }

    .modal-window {
        width: 90%;
        background-color: var(--lcd-deck-bg, #090d0e);
        border: 2px solid var(--lcd-button-border, #c3933b);
        border-radius: 6px;
        box-shadow: 0 0 20px rgba(0, 0, 0, 0.8), 0 0 10px rgba(255, 201, 102, 0.3);
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--lcd-font-ui, "VT323", monospace);
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 16px;
        background-color: var(--lcd-header-bg, #0e1214);
        border-bottom: 1px solid var(--lcd-border, #243524);
    }

    .modal-title {
        margin: 0;
        font-size: 1.3rem;
        color: var(--lcd-amber, #ffc966);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .close-button {
        background: transparent;
        border: none;
        color: var(--lcd-text-muted, #5a7c5a);
        font-size: 1.2rem;
        cursor: pointer;
        padding: 2px 6px;
        border-radius: 4px;
    }
    .close-button:hover {
        color: #ff6b6b;
    }

    .modal-body {
        padding: 16px;
        overflow-y: auto;
        font-size: 1.1rem;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        padding: 10px 16px;
        background-color: var(--lcd-header-bg, #0e1214);
        border-top: 1px solid var(--lcd-border, #243524);
    }
</style>
