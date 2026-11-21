<script lang="ts">
    /*
        Kizuna Editor - Mis Proyectos en la Nube (Issue #14)
        Lists remote projects from Kizuna Cloud and enables pulling / importing to local SQLite.
    */
    import { onMount } from "svelte";
    import { syncStore, computeProjectSyncStatus } from "../stores/syncStore.ts";
    import type { CloudProject } from "../types/sync.ts";
    import { songId } from "@/stores/projectStore";
    import { currentView } from "@shared/stores/viewStore";
    import { toastStore } from "@shared/stores/toastStore";
    import { formatDate } from "@shared/lib/formatters";
    import Modal from "@shared/components/ui/Modal.svelte";
    import Button from "@shared/components/ui/Button.svelte";
    import { GetSong } from "wailsjs/go/main/App";
    import type { models } from "wailsjs/go/models";

    export let onOpenAuthModal: () => void = () => {};

    let projects: CloudProject[] = [];
    let isLoading = false;
    let errorMessage = "";
    let downloadingId: number | null = null;

    let showOverwriteModal = false;
    let pendingProject: CloudProject | null = null;
    let pendingLocalSongId: number | null = null;

    async function fetchCloudProjects() {
        if (!$syncStore.isAuthenticated) {
            projects = [];
            return;
        }

        isLoading = true;
        errorMessage = "";

        try {
            projects = await syncStore.listCloudProjects();
        } catch (err) {
            console.error("Failed to list cloud projects:", err);
            errorMessage = err instanceof Error ? err.message : "Error al obtener proyectos de la nube.";
            toastStore.error(errorMessage);
        } finally {
            isLoading = false;
        }
    }

    async function handleDownload(project: CloudProject) {
        if (downloadingId !== null) return;

        // Check if this cloud project is already mapped locally
        const existingEntry = Object.entries($syncStore.cloudProjectMappings).find(
            ([_, meta]) => meta.cloudProjectId === project.id
        );
        const existingLocalId = existingEntry ? Number(existingEntry[0]) : 0;

        if (existingLocalId > 0) {
            let localSong: models.Song | null = null;
            try {
                if (typeof GetSong === "function") {
                    localSong = await GetSong(existingLocalId);
                }
            } catch {
                localSong = null;
            }

            if (localSong) {
                const mapping = $syncStore.cloudProjectMappings[existingLocalId];
                const syncInfo = computeProjectSyncStatus(localSong, mapping, project);

                // If local project has unpushed modifications or conflict, prompt the user for control
                if (syncInfo.status === "local_ahead" || syncInfo.status === "conflict") {
                    pendingProject = project;
                    pendingLocalSongId = existingLocalId;
                    showOverwriteModal = true;
                    return;
                }
            }
        }

        await executeDownload(project);
    }

    async function executeDownload(project: CloudProject) {
        downloadingId = project.id;

        try {
            toastStore.info(`Descargando "${project.title}" desde la nube...`, 2000);
            const localSongId = await syncStore.pullAndImportProject(project.id);
            
            toastStore.success(`"${project.title}" descargado e importado con éxito.`);
            
            // Set as active song and navigate to the editor
            songId.set(localSongId);
            currentView.set("editor");
        } catch (err) {
            console.error("Failed to download project:", err);
            const msg = err instanceof Error ? err.message : "Error desconocido al descargar.";
            toastStore.error(`Error al descargar: ${msg}`);
        } finally {
            downloadingId = null;
        }
    }

    async function confirmOverwrite() {
        showOverwriteModal = false;
        if (pendingProject) {
            const proj = pendingProject;
            pendingProject = null;
            pendingLocalSongId = null;
            await executeDownload(proj);
        }
    }

    function goToEditorToPush() {
        showOverwriteModal = false;
        if (pendingLocalSongId) {
            songId.set(pendingLocalSongId);
            currentView.set("editor");
            toastStore.info("Abre el proyecto para hacer Push de tus cambios locales pendientes.");
        }
        pendingProject = null;
        pendingLocalSongId = null;
    }

    function cancelOverwrite() {
        showOverwriteModal = false;
        pendingProject = null;
        pendingLocalSongId = null;
    }

    $: if ($syncStore.isAuthenticated) {
        fetchCloudProjects();
    } else {
        projects = [];
    }

    onMount(() => {
        if ($syncStore.isAuthenticated) {
            fetchCloudProjects();
        }
    });
</script>

<div class="cloud-view-container">
    <header class="cloud-header">
        <div class="title-section">
            <h1 class="lcd-glow">☁️ Mis Proyectos en la Nube</h1>
            <p class="subtitle">
                Proyectos y maquetas respaldados en Kizuna Cloud listos para sincronizar o continuar trabajando en esta máquina.
            </p>
        </div>

        {#if $syncStore.isAuthenticated}
            <div class="header-actions">
                <button
                    type="button"
                    class="lcd-btn refresh-btn"
                    disabled={isLoading}
                    on:click={fetchCloudProjects}
                    title="Actualizar lista"
                >
                    <span class="btn-icon" class:spinning={isLoading}>🔄</span>
                    <span>Actualizar</span>
                </button>
            </div>
        {/if}
    </header>

    {#if !$syncStore.isAuthenticated}
        <!-- Unauthenticated Prompt -->
        <div class="lcd-card empty-card auth-prompt">
            <div class="prompt-icon">☁️ 🔒</div>
            <h2>Conexión a la Nube Requerida</h2>
            <p>
                Inicia sesión con tu cuenta de Kizuna Cloud para explorar y descargar tus maquetas y pistas respaldadas.
            </p>
            <button
                type="button"
                class="lcd-btn primary-btn"
                on:click={onOpenAuthModal}
            >
                Conectar Nube
            </button>
        </div>
    {:else if isLoading && projects.length === 0}
        <!-- Loading Indicator -->
        <div class="lcd-card empty-card">
            <div class="loading-spinner"></div>
            <p class="loading-text">Consultando catálogo en Kizuna Cloud...</p>
        </div>
    {:else if errorMessage && projects.length === 0}
        <!-- Error State -->
        <div class="lcd-card empty-card error-state">
            <span class="warning-icon">⚠️</span>
            <h2>No se pudo conectar a la nube</h2>
            <p>{errorMessage}</p>
            <button
                type="button"
                class="lcd-btn"
                on:click={fetchCloudProjects}
            >
                Reintentar
            </button>
        </div>
    {:else if projects.length === 0}
        <!-- Empty State -->
        <div class="lcd-card empty-card">
            <div class="empty-icon">📁</div>
            <h2>No tienes proyectos respaldados todavía</h2>
            <p>
                Abre cualquier canción en el Editor y haz clic en <strong>"☁️ Respaldar Nube"</strong> en la barra superior para subir tu primera maqueta.
            </p>
        </div>
    {:else}
        <!-- Grid of Cloud Projects -->
        <div class="projects-grid">
            {#each projects as project (project.id)}
                <div class="lcd-card project-card">
                    <div class="card-header">
                        <h2 class="project-title" title={project.title}>
                            {project.title || "Sin título"}
                        </h2>
                        <div class="badges-group">
                            {#if project.active_lock}
                                <span
                                    class="lock-status-badge"
                                    title={`Bloqueado temporalmente por ${project.active_lock.username} (Issue #16)`}
                                >
                                    🔒 {project.active_lock.username}
                                </span>
                            {/if}
                            <span class="version-badge" title="Versión en la nube">
                                v{project.version || 1}
                            </span>
                        </div>
                    </div>

                    <div class="project-meta">
                        <div class="meta-row">
                            <span class="meta-label">BPM:</span>
                            <span class="meta-val">{project.bpm || 120}</span>
                            <span class="meta-separator">•</span>
                            <span class="meta-label">Compás:</span>
                            <span class="meta-val">{project.time_signature || "4/4"}</span>
                            {#if project.key_signature}
                                <span class="meta-separator">•</span>
                                <span class="meta-label">Tono:</span>
                                <span class="meta-val">{project.key_signature}</span>
                            {/if}
                        </div>

                        <div class="meta-row stats">
                            <span>🎼 {project.tracks ? project.tracks.length : 0} pistas</span>
                            {#if project.audio_versions && project.audio_versions.length > 0}
                                <span class="meta-separator">•</span>
                                <span>🎙 {project.audio_versions.length} tomas</span>
                            {/if}
                        </div>

                        {#if project.updated_at}
                            <div class="meta-row updated-at">
                                <span class="meta-label">Último respaldo:</span>
                                <span class="meta-val-date">{formatDate(project.updated_at)}</span>
                            </div>
                        {/if}
                    </div>

                    <div class="card-footer">
                        <button
                            type="button"
                            class="lcd-btn pull-btn"
                            disabled={downloadingId !== null}
                            on:click={() => handleDownload(project)}
                        >
                            {#if downloadingId === project.id}
                                <span class="btn-spinner"></span>
                                <span>Descargando...</span>
                            {:else}
                                <span>⬇️ Descargar (Pull)</span>
                            {/if}
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}

    <Modal
        isOpen={showOverwriteModal}
        title="⚠️ Cambios locales pendientes"
        maxWidth="540px"
        on:close={cancelOverwrite}
    >
        <div class="overwrite-modal-content">
            <p class="modal-text">
                El proyecto <strong>"{pendingProject?.title || "Sin título"}"</strong> tiene modificaciones locales en esta máquina que aún no has respaldado en Kizuna Cloud.
            </p>
            <div class="warning-callout">
                <span class="callout-icon">⚠️</span>
                <span>
                    Si descargas la versión de la nube (v{pendingProject?.version || 1}), tus cambios locales y pistas no respaldadas serán sobrescritos.
                </span>
            </div>
            <p class="tip-callout">
                ¿Deseas sobrescribir y descargar la versión de la nube, o prefieres ir al proyecto para hacer un <strong>Push</strong> primero?
            </p>
        </div>

        <div slot="footer" class="modal-footer-actions">
            <Button variant="danger" on:click={confirmOverwrite}>
                Sobreescribir y Descargar
            </Button>
            <Button variant="secondary" on:click={goToEditorToPush}>
                Ir a hacer Push
            </Button>
            <Button variant="ghost" on:click={cancelOverwrite}>
                Cancelar
            </Button>
        </div>
    </Modal>
</div>

<style>
    .cloud-view-container {
        padding: 24px;
        color: var(--lcd-text, #c8ff9f);
        height: 100%;
        overflow-y: auto;
        box-sizing: border-box;
    }

    .cloud-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 24px;
        border-bottom: 1px solid rgba(200, 255, 159, 0.15);
        padding-bottom: 16px;
    }

    .title-section h1 {
        margin: 0 0 6px 0;
        font-size: 1.6rem;
        letter-spacing: 0.05em;
        font-family: var(--font-lcd, "Courier New", monospace);
    }

    .lcd-glow {
        text-shadow: 0 0 8px rgba(200, 255, 159, 0.4);
    }

    .subtitle {
        margin: 0;
        font-size: 0.9rem;
        color: var(--lcd-dim, #7aa35f);
    }

    .header-actions {
        display: flex;
        gap: 10px;
    }

    .lcd-btn {
        background: #111b14;
        border: 1px solid var(--lcd-border, #2d4531);
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--font-lcd, monospace);
        padding: 8px 14px;
        border-radius: 4px;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 0.85rem;
        transition: all 0.15s ease-in-out;
    }

    .lcd-btn:hover:not(:disabled) {
        background: #1a2a1f;
        border-color: var(--lcd-text, #c8ff9f);
        box-shadow: 0 0 6px rgba(200, 255, 159, 0.3);
    }

    .lcd-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .primary-btn {
        background: #1a3322;
        border-color: var(--lcd-text, #c8ff9f);
        font-weight: bold;
    }

    .pull-btn {
        width: 100%;
        justify-content: center;
        font-weight: bold;
    }

    .pull-btn:hover:not(:disabled) {
        background: #1f3d28;
    }

    .btn-icon.spinning {
        animation: spin 1s linear infinite;
    }

    .lcd-card {
        background: #0a0f0c;
        border: 1px solid var(--lcd-border, #1f3324);
        border-radius: 6px;
        padding: 18px;
        box-shadow: inset 0 0 10px rgba(0, 0, 0, 0.5);
    }

    .empty-card {
        text-align: center;
        padding: 48px 24px;
        max-width: 520px;
        margin: 40px auto;
    }

    .empty-card h2 {
        margin: 12px 0 8px 0;
        font-size: 1.25rem;
    }

    .empty-card p {
        color: var(--lcd-dim, #7aa35f);
        font-size: 0.9rem;
        line-height: 1.5;
        margin-bottom: 20px;
    }

    .prompt-icon, .empty-icon {
        font-size: 2.5rem;
        margin-bottom: 8px;
    }

    .warning-icon {
        font-size: 2rem;
        color: var(--lcd-amber, #ffc966);
    }

    .projects-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 16px;
    }

    .project-card {
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        transition: transform 0.15s ease, border-color 0.15s ease;
    }

    .project-card:hover {
        border-color: var(--lcd-text, #c8ff9f);
        transform: translateY(-2px);
    }

    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
    }

    .project-title {
        margin: 0;
        font-size: 1.1rem;
        font-weight: 600;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 200px;
    }

    .version-badge {
        background: #142419;
        color: var(--lcd-amber, #ffc966);
        border: 1px solid #2b452f;
        font-size: 0.75rem;
        font-weight: bold;
        padding: 2px 6px;
        border-radius: 3px;
        font-family: var(--font-lcd, monospace);
    }

    .badges-group {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .lock-status-badge {
        background: rgba(255, 107, 107, 0.14);
        color: #ff8787;
        border: 1px solid #ff6b6b;
        font-size: 0.72rem;
        font-weight: bold;
        padding: 2px 6px;
        border-radius: 3px;
        font-family: var(--font-lcd, monospace);
        white-space: nowrap;
    }

    .project-meta {
        font-size: 0.85rem;
        color: var(--lcd-dim, #7aa35f);
        display: flex;
        flex-direction: column;
        gap: 6px;
        margin-bottom: 16px;
    }

    .meta-row {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 6px;
    }

    .meta-val {
        color: var(--lcd-text, #c8ff9f);
        font-weight: 500;
    }

    .meta-separator {
        opacity: 0.4;
    }

    .meta-val-date {
        color: #9ecc85;
        font-size: 0.8rem;
    }

    .card-footer {
        margin-top: auto;
    }

    .loading-spinner, .btn-spinner {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(200, 255, 159, 0.2);
        border-top-color: var(--lcd-text, #c8ff9f);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        display: inline-block;
    }

    .loading-spinner {
        width: 32px;
        height: 32px;
        border-width: 3px;
        margin: 0 auto 12px auto;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    .overwrite-modal-content {
        display: flex;
        flex-direction: column;
        gap: 14px;
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--font-lcd, "Courier New", monospace);
        font-size: 0.95rem;
        line-height: 1.5;
    }

    .modal-text {
        margin: 0;
    }

    .warning-callout {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        padding: 12px 14px;
        background: rgba(255, 176, 0, 0.12);
        border: 1px solid rgba(255, 176, 0, 0.4);
        border-radius: 4px;
        color: #ffd066;
        font-size: 0.9rem;
    }

    .callout-icon {
        font-size: 1.2rem;
        line-height: 1;
    }

    .tip-callout {
        margin: 0;
        color: var(--lcd-dim, #7aa35f);
        font-size: 0.88rem;
    }

    .modal-footer-actions {
        display: flex;
        justify-content: flex-end;
        align-items: center;
        gap: 10px;
        width: 100%;
        flex-wrap: wrap;
    }
</style>