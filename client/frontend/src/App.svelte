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
    import { onMount } from "svelte";
    import { currentView } from "./stores/viewStore";
    import { instrumentStore } from "./stores/instrumentStore";

    // Load global data on startup
    onMount(() => {
        instrumentStore.load();
    });

    import Dashboard from "@/components/Dashboard.svelte";
    import DashboardLayout from "@/layouts/DashboardLayout.svelte";
    import EditorLayout from "@/layouts/EditorLayout.svelte";
    import "./styles/main.css";
</script>

<main>
    {#if $currentView === "dashboard"}
        <DashboardLayout>
            <Dashboard />
        </DashboardLayout>
    {:else}
        <EditorLayout />
    {/if}
</main>

<style>
    /* --- Global Styles & Resources --- */

    :global(body) {
        margin: 0;
        padding: 0;
        font-family:
            -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
            Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
        overflow: hidden; /* Prevent window scrolling; panels scroll independently */
    }

    main {
        height: 100vh;
        display: flex;
        background-color: #1e1e1e;
        color: white;
    }
</style>
