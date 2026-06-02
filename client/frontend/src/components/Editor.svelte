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

    import "../styles/editor-theme.css";

    import type { main } from "../../../wailsjs/go/models";

    import Toolbar from "./Toolbar.svelte";

    // This component now recibe el objeto completo de track.
    export let track: main.Track;
</script>

<div class="editor-container">
    <div class="toolbar-subdivision">
        <Toolbar />
    </div>

    {#if track}
        <textarea
            class="editor-textarea"
            bind:value={track.data_content}
            spellcheck="false"
            placeholder="Type your lyrics and chords here..."
        />
    {:else}
        <textarea
            class="editor-textarea"
            disabled
            placeholder="No track selected."
        />
    {/if}
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

    .toolbar-subdivision {
        border-bottom: 1px solid var(--lcd-border);
        padding: 5px;
        background-color: var(--lcd-header-bg);
    }

    .editor-textarea {
        flex-grow: 1;
        background-color: var(--lcd-editor-bg);
        color: var(--lcd-text);
        border: none;
        padding: 16px 18px;
        font-family: var(--lcd-font-main);
        font-size: 1.2rem;
        line-height: 1.7;
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

    .editor-textarea:disabled {
        background-color: #050b0b;
        color: var(--lcd-text-muted);
        cursor: not-allowed;
    }
</style>
