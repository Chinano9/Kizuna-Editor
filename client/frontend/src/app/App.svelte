<script lang="ts">
    /*
        Kizuna Editor - Application Shell (src/app/App.svelte)
    */
    import { onMount } from "svelte";
    import { currentView } from "@shared/stores/viewStore";
    import { instrumentStore } from "@/stores/instrumentStore";

    import Dashboard from "@/components/Dashboard.svelte";
    import DashboardLayout from "@/layouts/DashboardLayout.svelte";
    import EditorLayout from "@/layouts/EditorLayout.svelte";
    import ToastContainer from "@shared/components/feedback/ToastContainer.svelte";
    import "@shared/styles/global.css";

    // Load global metadata on startup
    onMount(() => {
        instrumentStore.load();
    });
</script>

<main>
    {#if $currentView === "dashboard"}
        <DashboardLayout>
            <Dashboard />
        </DashboardLayout>
    {:else}
        <EditorLayout />
    {/if}

    <ToastContainer />
</main>

<style>
    :global(body) {
        margin: 0;
        padding: 0;
        overflow: hidden; /* Prevent window scrolling; panels scroll independently */
    }

    main {
        height: 100vh;
        width: 100%;
        display: flex;
        background-color: var(--app-bg, #050607);
        color: var(--lcd-text, #c8ff9f);
    }
</style>
