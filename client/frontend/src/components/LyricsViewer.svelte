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
    import { marked } from "marked";

    export let source: string = "";

    let renderedHtml: string = "";

    $: {
        if (typeof source === "string") {
            renderedHtml = marked(source);
        } else {
            renderedHtml = "<p>Invalid track data.</p>";
        }
    }
</script>

<div class="lyrics-container">
    {@html renderedHtml}
</div>

<style>
    .lyrics-container {
        padding: 20px;
        font-family: sans-serif;
        font-size: 1rem;
        line-height: 1.5;
        color: #333;
        background-color: #fff;
        /* Limitar ancho al contenedor padre y evitar desbordes horizontales */
        width: 100%;
        max-width: 100%;
        box-sizing: border-box;
        height: 100%;
        overflow-y: auto;
        overflow-x: hidden;
    }
    
    /* Evitar que contenido ancho (imágenes, tablas, bloques de código) genere scroll horizontal */
    .lyrics-container img,
    .lyrics-container table,
    .lyrics-container pre,
    .lyrics-container code {
        max-width: 100%;
        box-sizing: border-box;
    }
</style>
