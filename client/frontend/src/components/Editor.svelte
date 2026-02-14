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

    // This component now receives the entire track object.
    // The 'export let' makes it a bindable property.
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
    }

    .toolbar-subdivision {
        border-bottom: 1px solid #333;
        padding: 5px;
        background-color: #181818;
    }

    .editor-textarea {
        flex-grow: 1; /* Make textarea fill the available space */
        background-color: #1e1e1e;
        color: #d4d4d4;
        border: none;
        padding: 15px;
        font-family: "Fira Code", "Courier New", monospace;
        font-size: 1rem;
        line-height: 1.6;
        resize: none;
        outline: none;
    }

    .editor-textarea:disabled {
        background-color: #2a2a2a;
        cursor: not-allowed;
    }
</style>
