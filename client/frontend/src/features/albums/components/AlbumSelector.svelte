<script lang="ts">
    import { onMount } from "svelte";
    import { ListAlbums, AssignSongToAlbum } from "@wails/go/bindings/AlbumHandler";
    import type { models } from "@wails/go/models";
    import { song } from "@/stores/projectStore";
    import { Song } from "@shared/types/wails";
    import { toastStore } from "@shared/stores/toastStore";

    let albums: models.Album[] = [];
    let errorMessage = "";
    let isLoading = true;
    let isSaving = false;
    onMount(() => {
        void loadAlbums();
    });

    async function loadAlbums() {
        isLoading = true;
        try {
            albums = (await ListAlbums()) ?? [];
        } catch (error) {
            console.error("[AlbumSelector] ListAlbums failed:", error);
            errorMessage = "No se pudieron cargar los álbumes.";
            toastStore.error(errorMessage);
        } finally {
            isLoading = false;
        }
    }

    async function assignAlbum(event: Event) {
        const select = event.currentTarget as HTMLSelectElement;
        const albumId = Number(select.value);
        const currentSong = $song;
        const previousValue = String(currentSong?.album_id ?? "");

        if (isSaving) {
            select.value = previousValue;
            return;
        }

        errorMessage = "";
        if (!currentSong || currentSong.id <= 0) {
            select.value = previousValue;
            errorMessage = "Guarda el proyecto antes de asignarlo.";
            toastStore.warning(errorMessage);
            return;
        }
        if (!Number.isInteger(albumId) || albumId <= 0) {
            select.value = previousValue;
            errorMessage = "Selecciona un álbum válido.";
            toastStore.warning(errorMessage);
            return;
        }

        const songId = currentSong.id;
        console.log("[AlbumSelector] Selection:", {
            songId,
            previousAlbumId: currentSong.album_id,
            selectedAlbumId: albumId,
        });

        if (Number(currentSong.album_id) === albumId) {
            select.value = previousValue;
            return;
        }

        isSaving = true;
        try {
            await AssignSongToAlbum(songId, albumId);
            song.update((current) => {
                if (!current || current.id !== songId) return current;
                return new Song({ ...current, album_id: albumId });
            });
            console.log("[AlbumSelector] Saved:", { songId, albumId });
            toastStore.success("Proyecto asignado al álbum.");
        } catch (error) {
            console.error("[AlbumSelector] Assignment failed:", error);
            errorMessage = `No se pudo asignar el proyecto: ${String(error)}`;
            select.value = previousValue;
            toastStore.error(errorMessage);
        } finally {
            isSaving = false;
        }
    }
</script>

<div class="album-selector">
    <label for="project-album">Álbum</label>
    <!-- Recreate the native select when the loaded song or its stored album changes. Also recreate it after the async album list arrives. -->
    {#key `${$song?.id ?? 0}:${$song?.album_id ?? ""}:${albums.length}`}
        <select id="project-album" on:change={assignAlbum} disabled={isLoading || isSaving || !$song || $song.id <= 0 || albums.length === 0} title={$song?.id === 0 ? "Guarda primero el proyecto" : "Selecciona un álbum"}>
            <option value="" selected={$song?.album_id == null} disabled>Sin álbum</option>
            {#each albums as album (album.id)}
                <option value={String(album.id)} selected={Number($song?.album_id) === Number(album.id)}>
                    {album.title}
                </option>
            {/each}
        </select>
    {/key}

    {#if errorMessage}
        <span class="status error">{errorMessage}</span>
    {:else if isSaving}
        <span class="status">Guardando...</span>
    {:else if $song?.id === 0}
        <span class="status">Guarda el proyecto primero</span>
    {:else if !isLoading && albums.length === 0}
        <span class="status">No hay álbumes</span>
    {/if}
</div>

<style>
    .album-selector {
        display: flex;
        align-items: center;
        gap: 9px; min-width: 0;
        font-family: var(--lcd-font-ui, sans-serif);
    }
    label {
        font-size: 0.85rem;
        color: var(--lcd-text-muted, #a0ada5);
        white-space: nowrap;
    }
    select {
        min-width: 145px;
        max-width: 230px; padding: 6px 9px;
        border: 1px solid var(--lcd-border, #35483d);
        border-radius: 5px;
        background: #17211e;
        color: var(--lcd-text, #c8ff9f);
        font: inherit;
        font-size: 0.85rem;
        cursor: pointer;
    }
    select:focus {
        outline: 1px solid #b8ec9a;
    }
    select:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    .status {
        color: #a0ada5;
        font-size: 0.75rem;
        white-space: nowrap;
    }
    .error {
        color: #ff9b9b;
    }
</style>