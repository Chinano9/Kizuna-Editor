<script lang="ts">
    import { createEventDispatcher } from "svelte";

    export let id: string = `slider-${Math.random().toString(36).substring(2, 8)}`;
    export let label: string = "";
    export let min: number = 0;
    export let max: number = 1;
    export let step: number = 0.01;
    export let value: number = 1;
    export let disabled: boolean = false;
    export let showValue: boolean = false;
    export let valueFormatter: (val: number) => string = (v) => `${Math.round(v * 100)}%`;

    const dispatch = createEventDispatcher<{ input: number; change: number }>();

    function handleInput(e: Event) {
        const val = parseFloat((e.target as HTMLInputElement).value);
        value = val;
        dispatch("input", val);
    }

    function handleChange(e: Event) {
        const val = parseFloat((e.target as HTMLInputElement).value);
        value = val;
        dispatch("change", val);
    }
</script>

<div class="slider-group">
    {#if label || showValue}
        <div class="slider-header">
            {#if label}
                <label for={id} class="slider-label">{label}</label>
            {/if}
            {#if showValue}
                <span class="slider-display">{valueFormatter(value)}</span>
            {/if}
        </div>
    {/if}

    <input
        {id}
        type="range"
        {min}
        {max}
        {step}
        {value}
        {disabled}
        class="lcd-range"
        on:input={handleInput}
        on:change={handleChange}
    />
</div>

<style>
    .slider-group {
        display: flex;
        flex-direction: column;
        gap: 3px;
        width: 100%;
    }

    .slider-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-family: var(--lcd-font-ui, "VT323", monospace);
    }

    .slider-label {
        font-size: 0.95rem;
        color: var(--lcd-text-muted, #5a7c5a);
        text-transform: uppercase;
    }

    .slider-display {
        font-size: 1rem;
        color: var(--lcd-amber, #ffc966);
    }

    .lcd-range {
        -webkit-appearance: none;
        appearance: none;
        width: 100%;
        height: 6px;
        background: #0b1012;
        border: 1px solid var(--lcd-border, #243524);
        border-radius: 3px;
        outline: none;
    }

    .lcd-range::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 14px;
        height: 14px;
        background: var(--lcd-amber, #ffc966);
        border: 1px solid var(--lcd-button-border, #c3933b);
        border-radius: 50%;
        cursor: pointer;
        box-shadow: 0 0 5px rgba(255, 201, 102, 0.5);
    }

    .lcd-range::-webkit-slider-thumb:hover {
        box-shadow: 0 0 8px rgba(255, 201, 102, 0.8);
    }

    .lcd-range:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }
</style>
