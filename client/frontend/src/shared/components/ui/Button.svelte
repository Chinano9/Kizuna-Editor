<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let variant: "primary" | "secondary" | "danger" | "ghost" = "primary";
    export let size: "sm" | "md" | "lg" = "md";
    export let disabled: boolean = false;
    export let type: "button" | "submit" | "reset" = "button";
    export let title: string = "";

    const dispatch = createEventDispatcher<{ click: MouseEvent }>();

    function handleClick(e: MouseEvent) {
        if (!disabled) {
            dispatch("click", e);
        }
    }
</script>

<button
    {type}
    {title}
    class="lcd-button variant-{variant} size-{size}"
    class:disabled
    {disabled}
    on:click={handleClick}
>
    <slot />
</button>

<style>
    .lcd-button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        font-family: var(--lcd-font-ui, "VT323", monospace);
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.15s ease;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        outline: none;
    }

    /* Sizes */
    .size-sm {
        padding: 2px 8px;
        font-size: 0.95rem;
        height: 24px;
    }
    .size-md {
        padding: 4px 14px;
        font-size: 1.15rem;
        height: 32px;
    }
    .size-lg {
        padding: 6px 20px;
        font-size: 1.35rem;
        height: 42px;
    }

    /* Variants */
    .variant-primary {
        background-color: var(--lcd-button-bg, #15110a);
        color: var(--lcd-amber, #ffc966);
        border: 1px solid var(--lcd-button-border, #c3933b);
        box-shadow: 0 0 6px rgba(255, 201, 102, 0.2);
    }
    .variant-primary:hover:not(.disabled) {
        background-color: var(--lcd-button-bg-soft, #1c160a);
        border-color: #ffd685;
        box-shadow: 0 0 10px rgba(255, 201, 102, 0.5);
    }
    .variant-primary:active:not(.disabled) {
        transform: translateY(1px);
        box-shadow: 0 0 4px rgba(255, 201, 102, 0.3);
    }

    .variant-secondary {
        background-color: #0b120f;
        color: var(--lcd-text-soft, #a0e58d);
        border: 1px solid var(--lcd-border, #243524);
    }
    .variant-secondary:hover:not(.disabled) {
        border-color: var(--lcd-border-strong, #5cf29a);
        color: var(--lcd-text, #c8ff9f);
        box-shadow: 0 0 8px rgba(92, 242, 154, 0.3);
    }

    .variant-danger {
        background-color: #220808;
        color: #ff6b6b;
        border: 1px solid #aa2b2b;
        box-shadow: 0 0 6px rgba(255, 75, 75, 0.2);
    }
    .variant-danger:hover:not(.disabled) {
        background-color: #2f0b0b;
        border-color: #ff4d4d;
        box-shadow: 0 0 10px rgba(255, 75, 75, 0.5);
    }

    .variant-ghost {
        background: transparent;
        color: var(--lcd-text-muted, #5a7c5a);
        border: 1px solid transparent;
    }
    .variant-ghost:hover:not(.disabled) {
        color: var(--lcd-text, #c8ff9f);
        background: rgba(92, 242, 154, 0.08);
    }

    .disabled {
        opacity: 0.4;
        cursor: not-allowed;
        box-shadow: none !important;
    }
</style>
