<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let id: string = `toggle-${Math.random().toString(36).substring(2, 8)}`;
    export let label: string = "";
    export let checked: boolean = false;
    export let disabled: boolean = false;

    const dispatch = createEventDispatcher<{ change: boolean }>();

    function handleChange(e: Event) {
        checked = (e.target as HTMLInputElement).checked;
        dispatch("change", checked);
    }
</script>

<div class="toggle-container" class:disabled>
    <label for={id} class="toggle-switch">
        <input
            {id}
            type="checkbox"
            {checked}
            {disabled}
            on:change={handleChange}
        />
        <span class="slider round"></span>
    </label>
    {#if label}
        <label for={id} class="toggle-text">{label}</label>
    {/if}
</div>

<style>
    .toggle-container {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        font-family: var(--lcd-font-ui, "VT323", monospace);
    }

    .toggle-text {
        font-size: 1.1rem;
        color: var(--lcd-text-soft, #a0e58d);
        cursor: pointer;
        user-select: none;
    }

    .toggle-switch {
        position: relative;
        display: inline-block;
        width: 38px;
        height: 20px;
    }

    .toggle-switch input {
        opacity: 0;
        width: 0;
        height: 0;
    }

    .slider {
        position: absolute;
        cursor: pointer;
        inset: 0;
        background-color: #0b1012;
        border: 1px solid var(--lcd-border, #243524);
        transition: 0.2s;
        border-radius: 20px;
    }

    .slider:before {
        position: absolute;
        content: "";
        height: 14px;
        width: 14px;
        left: 2px;
        bottom: 2px;
        background-color: var(--lcd-text-muted, #5a7c5a);
        transition: 0.2s;
        border-radius: 50%;
    }

    input:checked + .slider {
        background-color: #15110a;
        border-color: var(--lcd-button-border, #c3933b);
    }

    input:checked + .slider:before {
        transform: translateX(18px);
        background-color: var(--lcd-amber, #ffc966);
        box-shadow: 0 0 6px rgba(255, 201, 102, 0.8);
    }

    .disabled {
        opacity: 0.45;
        pointer-events: none;
    }
</style>
