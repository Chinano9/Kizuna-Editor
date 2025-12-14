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
