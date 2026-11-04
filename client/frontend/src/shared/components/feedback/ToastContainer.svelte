<script lang="ts">
    import { toastStore } from "@shared/stores/toastStore";
    import Toast from "./Toast.svelte";

    function handleDismiss(e: CustomEvent<string>) {
        toastStore.dismiss(e.detail);
    }
</script>

<aside class="toast-container" aria-label="Notifications" role="region">
    {#each $toastStore as toast (toast.id)}
        <Toast
            id={toast.id}
            message={toast.message}
            type={toast.type}
            on:dismiss={handleDismiss}
        />
    {/each}
</aside>

<style>
    .toast-container {
        position: fixed;
        bottom: 95px; /* Above the 85px Workbar */
        right: 20px;
        display: flex;
        flex-direction: column-reverse;
        gap: 10px;
        z-index: 9999;
        pointer-events: auto;
    }
</style>
