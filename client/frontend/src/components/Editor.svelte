<script lang="ts">
    /*
      Kizuna Editor - A local-first songwriting environment.
      Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

      This program is free software: you can redistribute it and/or modify
      it under the terms of the GNU Affero General Public License as published by
      the Free Software Foundation, either version 3 of the License, or
      (at your option) any later version.

      This program is distributed in the hope that it will be useful,
      but WITHOUT ANY WARRANTY; without even the implied warranty of
      MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
      GNU Affero General Public License for more details.

      You should have received a copy of the GNU Affero General Public License
      along with this program.  If not, see <https://www.gnu.org/licenses/>.
    */

    import { createEventDispatcher } from "svelte";
    import "../styles/editor-theme.css";

    import type { Track } from "@shared/types/wails";
    import { activeSongStore } from "@/stores/projectStore";
    import { instrumentStore } from "@/stores/instrumentStore";
    import { looksLikeAlphaTex } from "@shared/lib/musicUtils";

    import Toolbar from "./Toolbar.svelte";

    export let track: Track;
    export let distractionFree: boolean = false;

    const dispatch = createEventDispatcher<{
        contentChange: string;
        toggleDistractionFree: boolean;
    }>();

    let textareaEl: HTMLTextAreaElement;
    let instrumentName = "";

    $: {
        const _catalog = $instrumentStore;
        instrumentName = track ? instrumentStore.getNameById(track.instrument_id) : "";
    }

    $: content = track?.data_content ?? "";
    $: isAlphaTexMode = instrumentName !== "Vocals" && looksLikeAlphaTex(content);
    $: lineCount = content ? content.split("\n").length : 1;
    $: wordCount = content.trim() ? content.trim().split(/\s+/).length : 0;

    function handleInput() {
        if (!track) return;
        activeSongStore.updateTrack(track);
        dispatch("contentChange", track.data_content);
    }

    function toggleFocusMode() {
        distractionFree = !distractionFree;
        dispatch("toggleDistractionFree", distractionFree);
        textareaEl?.focus();
    }

    function insertSnippet(snippet: string) {
        if (!track || !textareaEl) return;
        const start = textareaEl.selectionStart ?? track.data_content.length;
        const end = textareaEl.selectionEnd ?? track.data_content.length;
        const current = track.data_content ?? "";
        track.data_content = current.slice(0, start) + snippet + current.slice(end);
        activeSongStore.updateTrack(track);
        dispatch("contentChange", track.data_content);

        requestAnimationFrame(() => {
            if (!textareaEl) return;
            textareaEl.focus();
            const cursor = start + snippet.length;
            textareaEl.setSelectionRange(cursor, cursor);
        });
    }

    function handleKeydown(event: KeyboardEvent) {
        if (event.altKey && event.key.toLowerCase() === "z") {
            event.preventDefault();
            toggleFocusMode();
            return;
        }

        if (event.key === "Tab" && track && textareaEl) {
            event.preventDefault();
            insertSnippet("  ");
        }
    }
</script>

<div class="editor-container" class:distraction-free={distractionFree}>
    {#if !distractionFree}
        <div class="toolbar-subdivision">
            <Toolbar />
        </div>
    {/if}

    <div class="writing-subbar">
        <div class="mode-indicator">
            <span class="mode-dot" class:alphatex={isAlphaTexMode} />
            <span class="mode-text">
                {#if instrumentName === "Vocals"}
                    Letras y Acordes
                {:else if isAlphaTexMode}
                    Notación AlphaTex ({instrumentName || "Instrumento"})
                {:else}
                    Cuaderno Libre / AlphaTex
                {/if}
            </span>
        </div>

        {#if !distractionFree && track}
            <div class="quick-snippets" aria-label="Atajos de notación">
                {#if instrumentName === "Vocals"}
                    <button
                        type="button"
                        class="snippet-chip"
                        on:click={() => insertSnippet("[Am] ")}
                        title="Insertar acorde"
                    >
                        [Acorde]
                    </button>
                    <button
                        type="button"
                        class="snippet-chip"
                        on:click={() => insertSnippet("\n## Coro\n")}
                        title="Insertar sección de coro"
                    >
                        Sección
                    </button>
                {:else}
                    <button
                        type="button"
                        class="snippet-chip"
                        on:click={() => insertSnippet(":4 0.6 2.6 3.6 5.6 ")}
                        title="Insertar compás de ejemplo en AlphaTex"
                    >
                        + Compás
                    </button>
                    <button
                        type="button"
                        class="snippet-chip"
                        on:click={() => insertSnippet("(0.1 1.2 0.3 2.4 3.5) ")}
                        title="Insertar acorde en AlphaTex"
                    >
                        + Acorde
                    </button>
                    <button
                        type="button"
                        class="snippet-chip"
                        on:click={() => insertSnippet("| ")}
                        title="Insertar barra de compás"
                    >
                        | Barra
                    </button>
                {/if}
            </div>
        {/if}

        <button
            type="button"
            class="focus-toggle-btn"
            class:active={distractionFree}
            on:click={toggleFocusMode}
            title="Modo sin distracciones (Alt+Z)"
        >
            {distractionFree ? "Salir de Enfoque" : "Sin Distracciones"}
        </button>
    </div>

    {#if track}
        <textarea
            bind:this={textareaEl}
            class="editor-textarea"
            bind:value={track.data_content}
            on:input={handleInput}
            on:keydown={handleKeydown}
            spellcheck="false"
            placeholder={instrumentName === "Vocals"
                ? "Escribe tu letra y acordes sin distracciones (ej. [Am] Bajo el cielo gris)..."
                : 'Escribe AlphaTex o acordes aquí (ej. \\title "Mi Canción" \\tempo 120 . :4 0.6 2.6 3.6 5.6)...'}
        />
    {:else}
        <textarea
            class="editor-textarea"
            disabled
            placeholder="No track selected."
        />
    {/if}

    <div class="editor-status-footer">
        <span>Líneas: {lineCount}</span>
        <span>Tokens: {wordCount}</span>
        {#if distractionFree}
            <span class="focus-hint">Modo sin distracciones activo (Alt+Z)</span>
        {/if}
    </div>
</div>

<style>
    .editor-container {
        width: 40%;
        min-width: 300px;
        display: flex;
        flex-direction: column;
        background-color: var(--app-bg-alt);
        border-left: 1px solid var(--lcd-border);
        transition: width 0.2s ease;
    }

    .editor-container.distraction-free {
        width: 50%;
    }

    .toolbar-subdivision {
        border-bottom: 1px solid var(--lcd-border);
        padding: 5px;
        background-color: var(--lcd-header-bg);
    }

    .writing-subbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
        padding: 6px 14px;
        background-color: #071011;
        border-bottom: 1px solid rgba(255, 201, 102, 0.14);
        font-family: var(--lcd-font-ui, monospace);
        font-size: 0.78rem;
        flex-shrink: 0;
    }

    .mode-indicator {
        display: flex;
        align-items: center;
        gap: 6px;
        color: var(--lcd-text-soft, #b8e6a8);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .mode-dot {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        background-color: #ffc966;
        box-shadow: 0 0 6px rgba(255, 201, 102, 0.6);
        flex-shrink: 0;
    }

    .mode-dot.alphatex {
        background-color: #50fa7b;
        box-shadow: 0 0 6px rgba(80, 250, 123, 0.7);
    }

    .quick-snippets {
        display: flex;
        align-items: center;
        gap: 4px;
        flex-wrap: nowrap;
    }

    .snippet-chip {
        background: rgba(255, 201, 102, 0.08);
        color: var(--lcd-amber, #ffc966);
        border: 1px solid rgba(255, 201, 102, 0.28);
        border-radius: 3px;
        padding: 2px 7px;
        font-family: var(--lcd-font-ui, monospace);
        font-size: 0.72rem;
        cursor: pointer;
        white-space: nowrap;
        transition: background-color 0.15s ease, border-color 0.15s ease;
    }

    .snippet-chip:hover {
        background: rgba(255, 201, 102, 0.2);
        border-color: rgba(255, 201, 102, 0.55);
    }

    .focus-toggle-btn {
        background: transparent;
        color: var(--lcd-text-muted, #6c8c80);
        border: 1px solid rgba(255, 255, 255, 0.14);
        border-radius: 3px;
        padding: 2px 8px;
        font-family: var(--lcd-font-ui, monospace);
        font-size: 0.72rem;
        cursor: pointer;
        white-space: nowrap;
        transition: all 0.15s ease;
    }

    .focus-toggle-btn:hover,
    .focus-toggle-btn.active {
        color: var(--lcd-amber, #ffc966);
        border-color: var(--lcd-amber, #ffc966);
        background: rgba(255, 201, 102, 0.12);
    }

    .editor-textarea {
        flex-grow: 1;
        background-color: var(--lcd-editor-bg);
        color: var(--lcd-text);
        border: none;
        padding: 20px 22px;
        font-family: var(--lcd-font-main);
        font-size: 1.2rem;
        line-height: 1.75;
        resize: none;
        outline: none;
        text-shadow: var(--lcd-glow-soft);
        /* Scanlines suaves tipo pantalla LCD */
        background-image: linear-gradient(
            rgba(255, 255, 255, 0.03) 1px,
            transparent 1px
        );
        background-size: 100% 3px;
    }

    .editor-container.distraction-free .editor-textarea {
        padding: 28px 36px;
        font-size: 1.28rem;
        line-height: 1.85;
    }

    .editor-textarea:disabled {
        background-color: #050b0b;
        color: var(--lcd-text-muted);
        cursor: not-allowed;
    }

    .editor-status-footer {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 4px 14px;
        background-color: #050a0b;
        border-top: 1px solid rgba(255, 255, 255, 0.06);
        color: var(--lcd-text-muted, #6c8c80);
        font-family: var(--lcd-font-ui, monospace);
        font-size: 0.72rem;
        flex-shrink: 0;
    }

    .focus-hint {
        margin-left: auto;
        color: var(--lcd-amber, #ffc966);
    }
</style>
