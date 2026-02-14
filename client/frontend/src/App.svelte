<script lang="ts">
    import { onMount } from "svelte";
    import { currentView } from "./stores/viewStore";
    import { instrumentStore } from "./stores/instrumentStore";

    // Load global data on startup
    onMount(() => {
        instrumentStore.load();
    });

    // Importamos los dos grandes componentes
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
        /* Modern System Font Stack */
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
