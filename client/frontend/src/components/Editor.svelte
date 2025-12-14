<script lang="ts">
    import { trackSource } from "../stores/projectStore";
</script>

<div class="text-panel">
    <slot name="toolbar"></slot>

    <textarea
        bind:value={$trackSource}
        spellcheck="false"
        placeholder="Type your lyrics and chords here..."
    ></textarea>
</div>

<style>
    .text-panel {
        width: 40%;
        display: flex;
        flex-direction: column;
        border-right: 1px solid #333;
        height: 100%;
        min-width: 300px;
    }

    textarea {
        flex: 1;
        background-color: #1e1e1e;
        color: #d4d4d4;
        border: none;
        resize: none;
        padding: 15px;
        font-family: "Consolas", "Monaco", monospace;
        font-size: 14px;
        line-height: 1.6;
        outline: none;
    }

    textarea::placeholder {
        color: #555;
        font-style: italic;
    }
</style>
