<script lang="ts">
    import { currentView } from "./stores/viewStore";

    // Importamos los dos grandes componentes
    import Dashboard from "@/components/Dashboard.svelte";
    import DashboardLayout from "@/layouts/DashboardLayout.svelte";
    import EditorLayout from "@/layouts/EditorLayout.svelte";
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

    /* Register Bravura font.
       Essential for AlphaTab to render musical symbols correctly.
    */
    @font-face {
        font-family: "alphaTab";
        src: url("/font/Bravura.woff2?v=1") format("woff2");
        font-weight: normal;
        font-style: normal;
    }

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
