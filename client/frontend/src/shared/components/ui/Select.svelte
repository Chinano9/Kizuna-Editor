<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let id: string = `select-${Math.random().toString(36).substring(2, 8)}`;
    export let label: string = "";
    export let value: string | number = "";
    export let disabled: boolean = false;
    export let width: string = "100%";

    const dispatch = createEventDispatcher<{ change: Event }>();

    function handleChange(e: Event) {
        dispatch("change", e);
    }
</script>

<div class="select-container" style="width: {width};">
    {#if label}
        <label for={id} class="select-label">{label}</label>
    {/if}
    <select
        {id}
        bind:value
        {disabled}
        class="lcd-select"
        on:change={handleChange}
    >
        <slot />
    </select>
</div>

<style>
    .select-container {
        display: flex;
        flex-direction: column;
        gap: 3px;
    }

    .select-label {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 0.95rem;
        color: var(--lcd-text-muted, #5a7c5a);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .lcd-select {
        background-color: #0b1012;
        color: var(--lcd-amber, #ffc966);
        border: 1px solid var(--lcd-border, #243524);
        border-radius: 4px;
        padding: 4px 8px;
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 1.05rem;
        outline: none;
        cursor: pointer;
        transition: border-color 0.15s ease, box-shadow 0.15s ease;
    }

    .lcd-select:focus {
        border-color: var(--lcd-border-strong, #5cf29a);
        box-shadow: 0 0 8px rgba(92, 242, 154, 0.25);
    }

    .lcd-select:disabled {
        opacity: 0.45;
        cursor: not-allowed;
    }
</style>
