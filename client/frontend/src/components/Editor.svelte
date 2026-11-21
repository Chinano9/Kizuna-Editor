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

    import { createEventDispatcher, onDestroy } from "svelte";
    import "../styles/editor-theme.css";

    import type { Track } from "@shared/types/wails";
    import { song, activeSongStore, autoSaveStore } from "@/stores/projectStore";
    import { instrumentStore } from "@/stores/instrumentStore";
    import { looksLikeAlphaTex } from "@shared/lib/musicUtils";
    import { syncStore, isSongReadOnly } from "@/features/cloud-sync/stores/syncStore";

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

    $: activeCloudId = Number(
        $activeSongStore?.cloud_id || ($song?.id ? syncStore.getCloudIdForSong($song.id) : 0) || 0
    );
    $: isReadOnly = isSongReadOnly($syncStore, activeCloudId);

    function handleInput() {
        if (!track) return;
        activeSongStore.updateTrack(track);
        autoSaveStore.scheduleSave();
        dispatch("contentChange", track.data_content);
    }

    function toggleFocusMode() {
        distractionFree = !distractionFree;
        dispatch("toggleDistractionFree", distractionFree);
        textareaEl?.focus();
    }

    function insertSnippet(snippet: string) {
        if (!track || !textareaEl) return;
        const current = track.data_content ?? "";
        let start = textareaEl.selectionStart ?? current.length;
        let end = textareaEl.selectionEnd ?? current.length;

        if (instrumentName !== "Vocals") {
            const dotMatch = /(?:^|\n)[ \t]*\.[ \t]*(?=\n|$)/.exec(current);
            if (dotMatch) {
                const dotEndIndex = dotMatch.index + dotMatch[0].length;
                if (start <= dotEndIndex) {
                    start = dotEndIndex;
                    end = Math.max(end, dotEndIndex);
                }
            }
        }

        const needsLeadingNewline =
            instrumentName !== "Vocals" &&
            start > 0 &&
            current[start - 1] === ".";
        const effectiveSnippet = (needsLeadingNewline ? "\n" : "") + snippet;

        track.data_content = current.slice(0, start) + effectiveSnippet + current.slice(end);
        activeSongStore.updateTrack(track);
        autoSaveStore.scheduleSave();
        dispatch("contentChange", track.data_content);

        requestAnimationFrame(() => {
            if (!textareaEl) return;
            textareaEl.focus();
            const cursor = start + effectiveSnippet.length;
            textareaEl.setSelectionRange(cursor, cursor);
        });
    }

    function handlePaste(event: ClipboardEvent) {
        if (!track || !textareaEl || instrumentName === "Vocals") return;
        const pasted = event.clipboardData?.getData("text/plain") ?? "";
        if (!pasted.trim()) return;

        const current = track.data_content ?? "";
        const isDefaultHeaderOnly = /^\\title\s+"[^"]*"\s*\r?\n\s*\.\s*$/.test(current.trim());
        const pastedHasOwnHeaderOrDot =
            /\\(?:title|subtitle|tempo|ts|track|instrument)\b/i.test(pasted) ||
            /(?:^|\r?\n)\s*\.\s*(?:\r?\n|$)/.test(pasted);

        if (isDefaultHeaderOnly && pastedHasOwnHeaderOrDot) {
            event.preventDefault();
            track.data_content = pasted;
            activeSongStore.updateTrack(track);
            autoSaveStore.scheduleSave();
            dispatch("contentChange", track.data_content);
            requestAnimationFrame(() => {
                if (!textareaEl) return;
                const cursor = track.data_content.length;
                textareaEl.setSelectionRange(cursor, cursor);
            });
        }
    }

    onDestroy(() => {
        void autoSaveStore.flushIfPending();
    });

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

    {#if $syncStore.isLockedByOther}
        <div class="lock-warning-banner" role="alert">
            <span>
                ⚠️ <strong>Proyecto bloqueado por {$syncStore.lockedBy || "otro colaborador"}:</strong>
                Edición deshabilitada temporalmente para evitar colisiones.
            </span>
        </div>
    {:else if activeCloudId > 0 && $syncStore.isOffline}
        <div class="lock-offline-banner" role="status">
            <span>
                ⚡ <strong>Modo Local-First (Sin conexión a Kizuna Cloud):</strong>
                Edición local habilitada. Tus cambios se validarán al reconectar y sincronizar.
            </span>
        </div>
    {:else if isReadOnly}
        <div class="lock-readonly-banner" role="status">
            <span>
                🔒 <strong>Modo solo lectura (Candado liberado):</strong>
                Haz clic en <strong>"🔒 Editar (Check-out)"</strong> en la barra superior para adquirir el candado y poder editar.
            </span>
        </div>
    {/if}

    {#if track}
        <textarea
            bind:this={textareaEl}
            class="editor-textarea"
            bind:value={track.data_content}
            disabled={isReadOnly}
            on:input={handleInput}
            on:paste={handlePaste}
            on:keydown={handleKeydown}
            spellcheck="false"
            placeholder={isReadOnly
                ? $syncStore.isLockedByOther
                    ? `Bloqueado por ${$syncStore.lockedBy || "otro colaborador"} (Modo solo lectura)...`
                    : "Haz clic en '🔒 Editar (Check-out)' para adquirir el candado y editar..."
                : instrumentName === "Vocals"
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
        <span class="autosave-indicator {$autoSaveStore.status}">
            {#if $autoSaveStore.status === "pending"}
                ⏳ Guardando al pausar...
            {:else if $autoSaveStore.status === "saving"}
                💾 Autoguardando...
            {:else if $autoSaveStore.status === "saved"}
                ✓ Guardado localmente
            {:else if $autoSaveStore.status === "error"}
                ⚠️ Error al autoguardar
            {/if}
        </span>
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

    .lock-warning-banner {
        background: rgba(255, 85, 85, 0.14);
        border-bottom: 1px solid #ff6b6b;
        color: #ff9b9b;
        padding: 8px 14px;
        font-family: var(--lcd-font-ui);
        font-size: 0.88rem;
        line-height: 1.4;
        letter-spacing: 0.03em;
    }

    .lock-warning-banner strong {
        color: #ffc966;
    }

    .lock-readonly-banner {
        background: rgba(255, 184, 108, 0.12);
        border-bottom: 1px solid #ffb86c;
        color: #ffd8a8;
        padding: 8px 14px;
        font-family: var(--lcd-font-ui);
        font-size: 0.88rem;
        line-height: 1.4;
        letter-spacing: 0.03em;
    }

    .lock-readonly-banner strong {
        color: #ffb86c;
    }

    .lock-offline-banner {
        background: rgba(200, 255, 159, 0.1);
        border-bottom: 1px solid var(--lcd-border-strong, #4a7856);
        color: var(--lcd-text, #c8ff9f);
        padding: 8px 14px;
        font-family: var(--lcd-font-ui);
        font-size: 0.88rem;
        line-height: 1.4;
        letter-spacing: 0.03em;
    }

    .lock-offline-banner strong {
        color: var(--lcd-amber, #ffc966);
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

    .autosave-indicator {
        font-size: 0.7rem;
        transition: color 0.2s ease;
    }

    .autosave-indicator.pending,
    .autosave-indicator.saving {
        color: var(--lcd-amber, #ffc966);
    }

    .autosave-indicator.saved {
        color: #50fa7b;
    }

    .autosave-indicator.error {
        color: #ff6b6b;
    }
</style>
