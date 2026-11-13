
<script lang="ts">
    import { onMount } from "svelte";
    import { CreateAlbum, ListAlbums } from "@wails/go/bindings/AlbumHandler";
    import { albumStore } from "../stores/albumStore";
    import { toastStore } from "@shared/stores/toastStore";

    let isModalOpen = false;
    let isSaving = false;
    let isLoading = true;

    let title = "";
    let artist = "";
    let description = "";

    onMount(() => {
        void loadAlbums();
    });

    async function loadAlbums() {
        isLoading = true;

        try {
            const albums = await ListAlbums();
            albumStore.setAlbums(albums ?? []);
        } catch (error) {
            console.error("Error loading albums:", error);
            toastStore.error("No se pudieron cargar los álbumes.");
        } finally {
            isLoading = false;
        }
    }

    function openModal() {
        title = "";
        artist = "";
        description = "";
        isModalOpen = true;
    }

    function closeModal() {
        if (isSaving) return;
        isModalOpen = false;
    }

    async function saveAlbum() {
        if (isSaving) return;

        if (!title.trim()) {
            toastStore.warning("El nombre del álbum es obligatorio.");
            return;
        }

        isSaving = true;

        try {
            const album = await CreateAlbum(
                title.trim(),
                artist.trim(),
                description.trim()
            );

            albumStore.addAlbum(album);
            isModalOpen = false;

            toastStore.success("Álbum creado correctamente.");
        } catch (error) {
            console.error("Error creating album:", error);
            toastStore.error("No se pudo crear el álbum.");
        } finally {
            isSaving = false;
        }
    }
</script>

<section class="albums-view">
    <header class="albums-header">
        <div>
            <h1>Álbumes</h1>
            <p>Organiza tus proyectos musicales.</p>
        </div>

        <button type="button" class="primary-button" on:click={openModal}>+ Nuevo álbum</button>
    </header>

    {#if isLoading}
        <p class="status-text">Cargando álbumes...</p>
    {:else if $albumStore.length === 0}
        <div class="empty-state">
            <div class="empty-icon">💿</div>
            <h2>Aún no tienes álbumes</h2>
            <p>Crea tu primer álbum para comenzar.</p>
            <button type="button" class="primary-button" on:click={openModal}>Crear álbum</button>
        </div>
    {:else}
        <div class="album-grid">
            {#each $albumStore as album (album.id)}
                <article class="album-card">
                    <div class="album-cover">💿</div>
                    <h3>{album.title}</h3>
                    <p>{album.artist || "Artista sin definir"}</p>
                    <span class="album-info">Álbum creado</span>
                </article>
            {/each}
        </div>
    {/if}
</section>

{#if isModalOpen}
    <div class="modal-overlay" role="presentation">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="album-modal-title">
            <h2 id="album-modal-title">Nuevo álbum</h2>
            <p>Ingresa la información de tu álbum.</p>

            <form on:submit|preventDefault={saveAlbum}>
                <label for="album-title">Nombre del álbum *</label>
                <input id="album-title" type="text" bind:value={title} maxlength="150" placeholder="Mi primer álbum" required/>

                <label for="album-artist">Artista</label>
                <input id="album-artist" type="text" bind:value={artist} placeholder="Nombre del artista"/>

                <label for="album-description">Descripción</label>
                <textarea id="album-description" bind:value={description} placeholder="Descripción opcional" rows="3"></textarea>

                <div class="modal-actions">
                    <button type="button" class="secondary-button" disabled={isSaving} on:click={closeModal}>Cancelar</button>
                    <button type="submit" class="primary-button" disabled={isSaving}>{isSaving ? "Guardando..." : "Crear álbum"}</button>
                </div>
            </form>
        </div>
    </div>
{/if}

<style>
    .albums-view {
        height: 100%;
        overflow-y: auto;
        padding: 32px;
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--lcd-font-ui, sans-serif);
    }

    .albums-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 20px;
        margin-bottom: 32px;
    }

    h1 {
        margin: 0;
        font-size: 2rem;
    }

    .albums-header p,
    .modal p {
        color: #a0ada5;
    }

    .primary-button {
        background: #b8ec9a;
        color: #182518;
        border: none;
        padding: 11px 18px;
        border-radius: 6px;
        cursor: pointer;
        font-weight: 700;
    }

    .primary-button:hover {
        background: #d0ffb5;
    }

    .primary-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .secondary-button {
        background: #353b38;
        color: #ffffff;
        border: 1px solid #526157;
        padding: 11px 18px;
        border-radius: 6px;
        cursor: pointer;
    }

    .secondary-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .album-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: 20px;
    }

    .album-card {
        min-width: 0;
        padding: 14px;
        background: #17211e;
        border: 1px solid #35483d;
        border-radius: 9px;
    }

    .album-cover {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 150px;
        background: #263a31;
        border-radius: 6px;
        font-size: 3.5rem;
    }

    .album-card h3 {
        margin: 12px 0 4px;
        overflow-wrap: anywhere;
    }

    .album-card p {
        margin: 0;
        color: #a5b2a8;
    }

    .album-info {
        display: inline-block;
        margin-top: 12px;
        font-size: 0.8rem;
        color: #81968a;
    }

    .empty-state {
        text-align: center;
        padding: 65px 20px;
        border: 1px dashed #35483d;
        border-radius: 10px;
    }

    .empty-icon {
        font-size: 4rem;
    }

    .status-text {
        color: #a5b2a8;
    }

    .modal-overlay {
        position: fixed;
        inset: 0;
        z-index: 1000;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 20px;
        background: rgba(0, 0, 0, 0.75);
    }

    .modal {
        width: 100%;
        max-width: 440px;
        max-height: 90vh;
        overflow-y: auto;
        box-sizing: border-box;
        padding: 26px;
        background: #202823;
        border: 1px solid #49634f;
        border-radius: 10px;
        color: #e6f3e8;
    }

    .modal h2 {
        margin-top: 0;
    }

    form {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    label {
        font-weight: 600;
        font-size: 0.9rem;
    }

    input, textarea {
        box-sizing: border-box;
        width: 100%;
        padding: 11px;
        margin-bottom: 8px;
        border: 1px solid #49634f;
        border-radius: 5px;
        background: #111914;
        color: #ffffff;
        font: inherit;
    }

    input:focus, textarea:focus {
        outline: 2px solid #b8ec9a;
        outline-offset: 1px;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 12px;
    }
</style>
