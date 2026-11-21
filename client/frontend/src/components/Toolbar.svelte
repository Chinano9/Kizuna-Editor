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

    import { get } from "svelte/store";
    import { onMount } from "svelte";
    import { SaveSong, GetAudioVersionsForSong } from "../../wailsjs/go/main/App";
    import { song, autoBar, autoSaveStore, activeSongStore } from "../stores/projectStore";
    import { syncStore, isSongReadOnly } from "@/features/cloud-sync/stores/syncStore";
    import AuthModal from "@/features/cloud-sync/components/AuthModal.svelte";
    import ShareProjectModal from "@/features/cloud-sync/components/ShareProjectModal.svelte";
    import { toastStore } from "@shared/stores/toastStore";
    import type { AudioVersion } from "@shared/types/wails";

    let isAuthModalOpen = false;
    let isShareModalOpen = false;

    $: activeCloudId = Number(
        $activeSongStore?.cloud_id || ($song?.id ? syncStore.getCloudIdForSong($song.id) : 0) || 0
    );
    $: isReadOnly = isSongReadOnly($syncStore, activeCloudId);
    $: canPushToCloud =
        !$syncStore.isSyncing &&
        !$syncStore.isLockedByOther &&
        (activeCloudId === 0 || $syncStore.hasActiveCheckout || $syncStore.isOffline);

    onMount(() => {
        const handleSessionExpired = () => {
            isAuthModalOpen = true;
            toastStore.warning("Tu sesión ha expirado. Por favor inicia sesión de nuevo.");
        };
        window.addEventListener("kizuna-session-expired", handleSessionExpired);
        return () => {
            window.removeEventListener("kizuna-session-expired", handleSessionExpired);
        };
    });

    function handleTitleInput() {
        autoSaveStore.scheduleSave();
    }

    async function save() {
        if ($syncStore.isLockedByOther) {
            toastStore.warning(
                `Edición deshabilitada: el proyecto está bloqueado por ${$syncStore.lockedBy || "otro colaborador"}.`
            );
            return;
        }

        if (isReadOnly) {
            toastStore.warning(
                "Proyecto en modo solo lectura: haz clic en '🔒 Editar (Check-out)' para adquirir el candado antes de editar."
            );
            return;
        }

        const currentSong = get(song);

        if (!currentSong) {
            toastStore.warning("No song data to save.");
            return;
        }

        try {
            const updatedSong = await autoSaveStore.flush();
            if (updatedSong) {
                song.set(updatedSong);
                toastStore.success(`Project saved successfully (ID: ${updatedSong.id})`);
            }
        } catch (err) {
            console.error("Failed to save project:", err);
            toastStore.error("Failed to save project.");
        }
    }

    async function backupToCloud() {
        if ($syncStore.isSyncing) return;

        if (!$syncStore.isAuthenticated) {
            toastStore.warning("Inicia sesión en Kizuna Cloud antes de respaldar en la nube.");
            isAuthModalOpen = true;
            return;
        }

        if ($syncStore.isLockedByOther) {
            toastStore.warning(
                `No puedes sincronizar: el proyecto está bloqueado por ${$syncStore.lockedBy || "otro colaborador"}.`
            );
            return;
        }

        let currentSong = get(song);
        if (!currentSong) {
            toastStore.warning("No hay ninguna canción activa para respaldar.");
            return;
        }

        const cloudId =
            Number($activeSongStore?.cloud_id) ||
            (currentSong.id > 0 ? syncStore.getCloudIdForSong(currentSong.id) : 0) ||
            0;

        // Issue #16: When online and project is already in cloud, user must hold the checkout lock to push updates
        if (cloudId > 0 && !$syncStore.isOffline && !$syncStore.hasActiveCheckout) {
            toastStore.warning(
                "🔒 Debes adquirir el candado ('🔒 Editar (Check-out)') antes de modificar o sincronizar este proyecto."
            );
            return;
        }

        // Issue #16: Before syncing (Push), verify with the cloud if the project is locked by another user
        if (cloudId > 0) {
            try {
                const lockStatus = await syncStore.verifyProjectLock(cloudId);
                if (lockStatus.isLockedByOther) {
                    toastStore.warning(
                        `🔒 No puedes sincronizar: el proyecto está siendo editado por ${lockStatus.lockedBy || "otro colaborador"}.`
                    );
                    return;
                }
                if (lockStatus.isOffline) {
                    toastStore.error(
                        "No se pudo conectar con Kizuna Cloud para sincronizar. Tus cambios siguen guardados localmente."
                    );
                    return;
                }
            } catch (err) {
                console.warn("Lock status verification skipped or failed before push:", err);
            }
        }

        // Flush any pending local autosave or initial save (ID === 0) before cloud push
        try {
            const flushed = await autoSaveStore.flush();
            if (flushed) {
                currentSong = flushed;
            }
        } catch (err) {
            toastStore.error("Debes guardar la canción localmente antes de respaldar.");
            return;
        }

        // Gather all takes/audio versions associated with this project
        let audioVersions: AudioVersion[] = [];
        try {
            audioVersions = (await GetAudioVersionsForSong(currentSong.id)) || [];
        } catch {
            audioVersions = [];
        }

        try {
            const res = await syncStore.pushProject(currentSong, audioVersions);
            toastStore.success(
                `☁️ Proyecto respaldado en Kizuna Cloud (v${res.new_version}) — Candado liberado (Solo lectura)`
            );
        } catch (err) {
            console.error("Failed cloud backup:", err);
            toastStore.error(err instanceof Error ? err.message : "Error al respaldar en la nube.");
        }
    }

    async function handleFinishEditing() {
        if ($syncStore.isCheckingLock || !activeCloudId) return;

        try {
            await syncStore.releaseProjectLock(activeCloudId);
            toastStore.success(
                "🔓 Edición terminada: candado liberado. El editor pasó a modo solo lectura."
            );
        } catch (err) {
            console.error("Failed to release project lock:", err);
            toastStore.error(
                err instanceof Error ? err.message : "No se pudo liberar el candado del proyecto."
            );
        }
    }

    async function handleRequestCheckout() {
        if ($syncStore.isCheckingLock || !activeCloudId) return;

        try {
            await syncStore.checkoutProject(activeCloudId);
            toastStore.success("🔒 Candado adquirido. Edición habilitada.");
        } catch (err) {
            const msg = err instanceof Error ? err.message : "El proyecto sigue bloqueado.";
            if ($syncStore.isOffline) {
                toastStore.info(`⚡ ${msg}`);
            } else {
                toastStore.warning(`🔒 ${msg}`);
            }
        }
    }

    function handleShareClick() {
        const currentSong = get(activeSongStore);
        let cloudId = currentSong?.cloud_id;
        if (!cloudId && currentSong && currentSong.id > 0) {
            cloudId = syncStore.getCloudIdForSong(currentSong.id);
        }

        if (!cloudId || String(cloudId).trim() === "" || String(cloudId) === "0") {
            toastStore.warning("Debes respaldar el proyecto en la nube antes de invitar colaboradores");
            return;
        }

        isShareModalOpen = true;
    }
</script>

<div class="panel-header control-bar">
    {#if $song}
        <input
            type="text"
            class="title-input"
            bind:value={$song.title}
            on:input={handleTitleInput}
            placeholder="Song Title..."
            disabled={isReadOnly}
        />
    {/if}

    <div class="actions">
        {#if $syncStore.isAuthenticated && activeCloudId > 0}
            {#if $syncStore.isLockedByOther}
                <div
                    class="lock-badge locked-by-other"
                    title="Otro colaborador tiene el candado activo. Edición deshabilitada temporalmente (Issue #16)"
                >
                    <span>🔒 Bloqueado por {$syncStore.lockedBy || "otro usuario"}</span>
                    <button
                        type="button"
                        class="lock-action-btn"
                        disabled={$syncStore.isCheckingLock}
                        on:click={handleRequestCheckout}
                        title="Consultar a la nube si el candado ya fue liberado"
                    >
                        🔄
                    </button>
                </div>
            {:else if $syncStore.isOffline}
                <div
                    class="lock-badge offline-badge"
                    title="Sin conexión a Kizuna Cloud: puedes editar y guardar localmente (Local-First)"
                >
                    <span>⚡ Modo Offline (Local)</span>
                    <button
                        type="button"
                        class="lock-action-btn"
                        disabled={$syncStore.isCheckingLock}
                        on:click={handleRequestCheckout}
                        title="Intentar reconectar con Kizuna Cloud y adquirir candado"
                    >
                        🔄
                    </button>
                </div>
            {:else if $syncStore.hasActiveCheckout}
                <button
                    type="button"
                    class="release-lock-btn"
                    disabled={$syncStore.isCheckingLock || $syncStore.isSyncing}
                    on:click={handleFinishEditing}
                    title="Liberar el candado en Kizuna Cloud y bloquear edición local (Issue #22)"
                >
                    🔓 Terminar edición
                </button>
            {:else}
                <button
                    type="button"
                    class="checkout-lock-btn"
                    disabled={$syncStore.isCheckingLock || $syncStore.isSyncing}
                    on:click={handleRequestCheckout}
                    title="Adquirir candado de edición en Kizuna Cloud para desbloquear el editor (Issue #16)"
                >
                    🔒 Editar (Check-out)
                </button>
            {/if}
        {/if}

        <label class="toggle-switch">
            <input
                type="checkbox"
                bind:checked={$autoBar}
                disabled={isReadOnly}
            />

            <span>Auto |</span>
        </label>

        <button
            class="save-btn"
            disabled={isReadOnly}
            on:click={save}
            title={isReadOnly
                ? $syncStore.isLockedByOther
                    ? `Edición bloqueada por ${$syncStore.lockedBy || "otro colaborador"}`
                    : "Haz clic en '🔒 Editar (Check-out)' para desbloquear la edición"
                : "Guardar cambios localmente"}
        >
            💾 Save
        </button>

        <button
            class="cloud-btn"
            class:syncing={$syncStore.isSyncing}
            disabled={!canPushToCloud}
            on:click={backupToCloud}
            title={!canPushToCloud
                ? $syncStore.isLockedByOther
                    ? `Sincronización bloqueada por ${$syncStore.lockedBy || "otro colaborador"}`
                    : "Adquiere primero el candado ('🔒 Editar') para modificar y sincronizar este proyecto"
                : "Respaldar este proyecto en Kizuna Cloud y liberar candado (Issues #13, #16, #22)"}
        >
            {#if $syncStore.isSyncing}
                ⏳ Subiendo...
            {:else}
                ☁️ Respaldar Nube
            {/if}
        </button>

        <button
            class="share-btn"
            on:click={handleShareClick}
            title="Compartir proyecto con colaboradores (Issue #15)"
        >
            👥 Compartir
        </button>
    </div>

    <AuthModal bind:isOpen={isAuthModalOpen} />
    <ShareProjectModal
        bind:isOpen={isShareModalOpen}
        cloudId={$activeSongStore?.cloud_id || ($song?.id ? syncStore.getCloudIdForSong($song.id) : "") || ""}
        projectTitle={$song?.title || "Proyecto"}
    />
</div>

<style>
    .panel-header {
        background-color: var(--lcd-header-bg);
        color: var(--lcd-text);
        padding: 10px 15px;

        border-bottom: 1px solid var(--lcd-header-border);
        font-family: var(--lcd-font-ui);
        flex-shrink: 0;
        min-height: 40px;
        display: flex;
        flex-direction: row;
        justify-content: space-between;
        align-items: center;
        box-shadow: var(--lcd-header-shadow);
    }

    .control-bar {
        display: flex;
        justify-content: space-between;

        align-items: center;

        gap: 10px;

        width: 100%;
    }

    .title-input {
        background: transparent;

        border: none;

        color: var(--lcd-text);
        text-shadow: var(--lcd-glow-soft);
        font-size: 1rem;
        font-weight: bold;

        width: 100%;

        outline: none;

        border-bottom: 1px solid var(--lcd-border);
    }

    .title-input:focus {
        border-bottom-color: var(--lcd-border-strong);
    }

    .title-input:disabled {
        opacity: 0.6;
        cursor: not-allowed;
        color: var(--lcd-text-muted);
    }

    .save-btn:disabled {
        opacity: 0.45;
        cursor: not-allowed;
        transform: none;
        filter: grayscale(0.4);
    }

    .lock-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 3px 10px;
        border-radius: 4px;
        font-family: var(--lcd-font-ui);
        font-size: 0.82rem;
        white-space: nowrap;
        letter-spacing: 0.04em;
    }

    .lock-badge.locked-by-other {
        background: rgba(255, 85, 85, 0.16);
        border: 1px solid #ff6b6b;
        color: #ff8787;
        text-shadow: 0 0 6px rgba(255, 107, 107, 0.35);
    }

    .lock-badge.offline-badge {
        background: rgba(255, 201, 102, 0.14);
        border: 1px solid #ffc966;
        color: #ffc966;
        text-shadow: 0 0 6px rgba(255, 201, 102, 0.35);
    }

    .lock-action-btn {
        background: transparent;
        border: 1px solid rgba(255, 135, 135, 0.4);
        color: inherit;
        border-radius: 3px;
        cursor: pointer;
        padding: 1px 5px;
        font-size: 0.78rem;
        transition: all 0.15s ease;
    }

    .lock-action-btn:hover:not(:disabled) {
        background: rgba(255, 107, 107, 0.25);
        border-color: #ff8787;
    }

    .release-lock-btn,
    .checkout-lock-btn {
        position: relative;
        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 184, 108, 0.38) 0,
            var(--lcd-button-bg) 60%
        );
        color: #ffb86c;
        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        border: 1px solid #ffb86c;
        padding: 4px 10px;
        border-radius: 4px;
        cursor: pointer;
        white-space: nowrap;
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        font-family: var(--lcd-font-ui);
        font-size: 0.85rem;
        letter-spacing: 0.06em;
        text-transform: uppercase;
        transition: all 0.15s ease;
    }

    .release-lock-btn:hover:not(:disabled),
    .checkout-lock-btn:hover:not(:disabled) {
        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 184, 108, 0.58) 0,
            var(--lcd-button-bg-soft) 65%
        );
        box-shadow: 0 0 8px rgba(255, 184, 108, 0.45);
        transform: translateY(-1px);
    }

    .release-lock-btn:disabled,
    .checkout-lock-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .actions {
        display: flex;
        align-items: center;

        gap: 10px;
    }

    .save-btn {
        position: relative;

        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.45) 0,
            var(--lcd-button-bg) 60%
        );
        color: var(--lcd-amber);

        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        border: 1px solid var(--lcd-button-border);

        padding: 4px 12px;
        border-radius: 4px;
        cursor: pointer;

        white-space: nowrap;

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        font-family: var(--lcd-font-ui);

        font-size: 0.9rem;
        letter-spacing: 0.08em;

        text-transform: uppercase;

        transition:
            background-color 0.15s ease,
            border-color 0.15s ease,
            transform 0.1s ease,
            box-shadow 0.15s ease,
            filter 0.15s ease;
    }

    .save-btn::before {
        content: "";

        position: absolute;

        inset: 2px 2px auto 2px;
        height: 40%;
        background: linear-gradient(
            to bottom,

            rgba(255, 220, 170, 0.2),
            transparent
        );

        border-radius: 3px 3px 1px 1px;
        pointer-events: none;

        opacity: 0.85;
    }

    .save-btn:hover {
        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.6) 0,
            var(--lcd-button-bg-soft) 65%
        );
        border-color: var(--lcd-button-hover-border);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow);
        transform: translateY(-1px);

        filter: brightness(1.05);
    }

    .save-btn:active {
        transform: translateY(1px);

        filter: brightness(0.9);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.9),
            0 0 6px rgba(255, 201, 102, 0.45);
    }

    .toggle-switch {
        display: flex;

        align-items: center;

        cursor: pointer;

        font-size: 0.8rem;

        color: var(--lcd-text-soft);

        text-shadow: var(--lcd-glow-soft);

        white-space: nowrap;

        gap: 4px;
    }

    .toggle-switch input {
        margin-right: 5px;

        accent-color: var(--lcd-amber);
    }

    .toggle-switch span {
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--lcd-amber-soft);
    }

    .cloud-btn {
        position: relative;
        background: radial-gradient(
            circle at 50% 0,
            rgba(80, 250, 123, 0.4) 0,
            var(--lcd-button-bg) 60%
        );
        color: #50fa7b;
        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        border: 1px solid #50fa7b;
        padding: 4px 12px;
        border-radius: 4px;
        cursor: pointer;
        white-space: nowrap;
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        font-family: var(--lcd-font-ui);
        font-size: 0.9rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        transition: all 0.15s ease;
    }

    .cloud-btn:hover:not(:disabled) {
        background: radial-gradient(
            circle at 50% 0,
            rgba(80, 250, 123, 0.6) 0,
            var(--lcd-button-bg-soft) 65%
        );
        box-shadow: 0 0 8px rgba(80, 250, 123, 0.45);
        transform: translateY(-1px);
        filter: brightness(1.05);
    }

    .cloud-btn:active:not(:disabled) {
        transform: translateY(1px);
    }

    .cloud-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .share-btn {
        position: relative;
        background: radial-gradient(
            circle at 50% 0,
            rgba(0, 200, 255, 0.35) 0,
            var(--lcd-button-bg) 60%
        );
        color: #70d6ff;
        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        border: 1px solid #70d6ff;
        padding: 4px 12px;
        border-radius: 4px;
        cursor: pointer;
        white-space: nowrap;
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.8),
            var(--lcd-glow-soft);
        font-family: var(--lcd-font-ui);
        font-size: 0.9rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        transition: all 0.15s ease;
    }

    .share-btn:hover {
        background: radial-gradient(
            circle at 50% 0,
            rgba(0, 200, 255, 0.55) 0,
            var(--lcd-button-bg-soft) 65%
        );
        box-shadow: 0 0 8px rgba(0, 200, 255, 0.45);
        transform: translateY(-1px);
        filter: brightness(1.05);
    }

    .share-btn:active {
        transform: translateY(1px);
    }
</style>
