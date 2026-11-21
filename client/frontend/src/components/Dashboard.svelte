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

    import { GetRecentSongs } from "wailsjs/go/main/App";

    import type { models } from "wailsjs/go/models";

    import { songId, activeSongStore } from "../stores/projectStore";
    import { activeTrackIndex } from "@/stores/trackStore";
    import { currentView } from "../stores/viewStore";
    import {
        syncStore,
        computeProjectSyncStatus,
        deduplicateRecentSongs,
        type ProjectSyncComparison,
        type CloudProjectMeta,
    } from "@/features/cloud-sync/stores/syncStore";
    import type { CloudProject } from "@/features/cloud-sync/types/sync";
    import { formatDate } from "@shared/lib/formatters";

    let recentSongs: models.Song[] = [];
    let cloudProjectsById: Record<number, CloudProject> = {};
    let isCheckingCloud = false;

    $: uniqueRecentSongs = deduplicateRecentSongs(recentSongs, $syncStore.cloudProjectMappings);

    async function loadDashboardData() {
        try {
            recentSongs = (await GetRecentSongs()) ?? [];
        } catch (err) {
            console.error("Failed to load recent songs:", err);
            recentSongs = [];
        }

        if ($syncStore.isAuthenticated) {
            await refreshCloudComparison();
        } else {
            cloudProjectsById = {};
        }
    }

    async function refreshCloudComparison() {
        if (!$syncStore.isAuthenticated || isCheckingCloud) return;
        isCheckingCloud = true;
        try {
            const list = await syncStore.listCloudProjects();
            const map: Record<number, CloudProject> = {};
            for (const cp of list) {
                map[cp.id] = cp;
            }
            cloudProjectsById = map;
        } catch (err) {
            // Local-First resilience: if offline, keep local comparison active without crashing
            console.warn("Offline or unable to fetch cloud projects for comparison:", err);
        } finally {
            isCheckingCloud = false;
        }
    }

    function getSongSyncStatus(
        songItem: models.Song,
        mappings: typeof $syncStore.cloudProjectMappings,
        cloudMap: Record<number, CloudProject>
    ): ProjectSyncComparison {
        const mapping = mappings[songItem.id] ?? null;
        const cloudProj = mapping ? cloudMap[mapping.cloudProjectId] ?? null : null;
        return computeProjectSyncStatus(songItem, mapping, cloudProj);
    }

    $: if ($syncStore.isAuthenticated) {
        void refreshCloudComparison();
    } else {
        cloudProjectsById = {};
    }

    onMount(async () => {
        await loadDashboardData();
    });

    function newSong() {
        activeSongStore.createNewSong();
        activeTrackIndex.set(0);
        currentView.set("editor");
    }

    function openSong(songToOpen: models.Song) {
        songId.set(songToOpen.id);

        currentView.set("editor");
    }
</script>

<div class="dashboard-container">
    <header>
        <h1>Kizuna Editor</h1>

        <p>Your local-first songwriting environment.</p>
    </header>

    <div class="main-actions">
        <button class="action-btn" on:click={newSong}>+ New Idea</button>
    </div>

    <section class="recent-work">
        <div class="section-header">
            <h2>Recent Work</h2>
            {#if $syncStore.isAuthenticated}
                <button
                    type="button"
                    class="check-cloud-btn"
                    disabled={isCheckingCloud}
                    on:click={refreshCloudComparison}
                    title="Comparar fechas y checksums de SQLite contra Kizuna Cloud (Issue #21)"
                >
                    <span class:spinning={isCheckingCloud}>🔄</span>
                    <span>{isCheckingCloud ? "Verificando..." : "Verificar nube"}</span>
                </button>
            {/if}
        </div>

        {#if !uniqueRecentSongs || uniqueRecentSongs.length === 0}
            <p>No recent songs found. Start a new idea!</p>
        {:else}
            <div class="grid">
                {#each uniqueRecentSongs as song (song.id)}
                    {@const syncInfo = getSongSyncStatus(
                        song,
                        $syncStore.cloudProjectMappings,
                        cloudProjectsById
                    )}
                    <div
                        class="card"
                        class:status-local-ahead={syncInfo.status === "local_ahead"}
                        class:status-cloud-ahead={syncInfo.status === "cloud_ahead"}
                        class:status-conflict={syncInfo.status === "conflict"}
                        on:click={() => openSong(song)}
                        on:keydown={(e) => e.key === "Enter" && openSong(song)}
                        role="button"
                        tabindex="0"
                    >
                        <div class="card-info">
                            <div class="card-top-row">
                                <h3 title={song.title}>{song.title}</h3>
                                {#if song.content_hash}
                                    <span
                                        class="hash-pill"
                                        title={`Checksum SHA-256 local: ${song.content_hash}`}
                                    >
                                        #{song.content_hash.slice(0, 7)}
                                    </span>
                                {/if}
                            </div>

                            <div class="sync-badge-row">
                                <span
                                    class="sync-status-badge {syncInfo.status}"
                                    title={syncInfo.status === "local_ahead"
                                        ? "SQLite tiene modificaciones o grabaciones más recientes que el último respaldo en la nube"
                                        : syncInfo.status === "cloud_ahead"
                                        ? "Kizuna Cloud tiene una versión más reciente lista para descargar"
                                        : syncInfo.status === "conflict"
                                        ? "Tanto SQLite local como Kizuna Cloud tienen cambios nuevos"
                                        : syncInfo.status === "synced"
                                        ? "El proyecto local coincide con la versión respaldada en Kizuna Cloud"
                                        : "Este proyecto solo existe en tu base de datos SQLite local"}
                                >
                                    <span class="badge-icon">{syncInfo.icon}</span>
                                    <span>{syncInfo.label}</span>
                                    {#if syncInfo.cloudVersion && syncInfo.status !== "local_only"}
                                        <span class="badge-version">v{syncInfo.cloudVersion}</span>
                                    {/if}
                                </span>
                            </div>

                            <div class="dates-comparison">
                                <span class="date">
                                    SQLite: {formatDate(song.updated_at)}
                                </span>
                                {#if syncInfo.cloudUpdatedAt}
                                    <span class="date cloud-date">
                                        Nube: {formatDate(syncInfo.cloudUpdatedAt)}
                                    </span>
                                {:else if syncInfo.lastSyncedAt}
                                    <span class="date cloud-date">
                                        Último sync: {formatDate(syncInfo.lastSyncedAt)}
                                    </span>
                                {/if}
                            </div>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </section>
</div>

<style>
    .dashboard-container {
        width: 100%;

        max-width: 960px;

        margin: 0 auto;

        padding: 40px 20px;

        color: var(--lcd-text);
        background: radial-gradient(
            circle at top,
            #0f1f1a 0,
            var(--app-bg) 45%
        );
        font-family: var(--lcd-font-ui);
        text-shadow: var(--lcd-glow-soft);
    }

    header {
        text-align: center;

        margin-bottom: 40px;
    }

    header h1 {
        font-size: 2.5rem;

        margin: 0;

        color: var(--lcd-text);
        letter-spacing: 0.08em;
    }

    header p {
        font-size: 1.1rem;

        color: var(--lcd-text-muted);
    }

    .main-actions {
        text-align: center;

        margin-bottom: 50px;
    }

    .action-btn {
        position: relative;

        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.45) 0,
            var(--lcd-button-bg) 60%
        );
        color: var(--lcd-amber);

        border: 1px solid var(--lcd-button-border);

        padding: 10px 26px;
        font-size: 1.05rem;
        font-weight: bold;

        border-radius: 4px;
        cursor: pointer;

        text-transform: uppercase;

        letter-spacing: 0.12em;

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.7),
            var(--lcd-glow-soft);

        text-shadow:
            0 0 3px rgba(0, 0, 0, 0.9),
            var(--lcd-glow-soft);

        transition:
            background-color 0.15s ease,
            border-color 0.15s ease,
            transform 0.1s ease,
            box-shadow 0.15s ease,
            filter 0.15s ease;
    }

    .action-btn::before {
        content: "";

        position: absolute;

        inset: 2px 2px auto 2px;
        height: 35%;
        background: linear-gradient(
            to bottom,

            rgba(255, 220, 170, 0.18),
            transparent
        );

        border-radius: 3px 3px 1px 1px;
        pointer-events: none;

        opacity: 0.8;
    }

    .action-btn:hover {
        background: radial-gradient(
            circle at 50% 0,
            rgba(255, 201, 102, 0.55) 0,
            var(--lcd-button-bg-soft) 65%
        );
        border-color: var(--lcd-button-hover-border);

        transform: translateY(-1px);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.7),
            var(--lcd-glow);
        filter: brightness(1.05);
    }

    .action-btn:active {
        transform: translateY(1px);

        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.9),
            0 0 6px rgba(255, 201, 102, 0.5);

        filter: brightness(0.92);
    }

    .recent-work h2 {
        margin: 0;
        color: var(--lcd-text-soft);
        text-transform: uppercase;
        letter-spacing: 0.1em;
        font-size: 0.9rem;
    }

    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom: 1px solid var(--lcd-border);
        padding-bottom: 10px;
        margin-bottom: 20px;
    }

    .check-cloud-btn {
        background: #0a1615;
        border: 1px solid var(--lcd-border);
        color: var(--lcd-text-soft);
        font-family: var(--lcd-font-ui);
        font-size: 0.78rem;
        padding: 4px 10px;
        border-radius: 4px;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 6px;
        transition: all 0.15s ease;
    }

    .check-cloud-btn:hover:not(:disabled) {
        border-color: var(--lcd-text);
        color: var(--lcd-text);
        box-shadow: var(--lcd-glow-soft);
    }

    .check-cloud-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .spinning {
        display: inline-block;
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    .grid {
        display: grid;

        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));

        gap: 20px;
    }

    .card {
        background: linear-gradient(135deg, #050b0b, #060f10);
        border: 1px solid var(--lcd-border);
        border-radius: 5px;

        padding: 16px 18px;
        cursor: pointer;

        transition:
            transform 0.15s ease,
            background-color 0.15s ease,
            border-color 0.15s ease,
            box-shadow 0.15s ease;
        box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.7);
    }

    .card:hover {
        transform: translateY(-2px);
        background-color: #071616;
        border-color: var(--lcd-border-strong);
        box-shadow: var(--lcd-glow);
    }

    .card-info h3 {
        margin: 0;
        color: var(--lcd-text);
        font-size: 1.05rem;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .card-top-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
        margin-bottom: 10px;
    }

    .hash-pill {
        font-family: var(--lcd-font-main, monospace);
        font-size: 0.72rem;
        color: var(--lcd-text-muted);
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid rgba(255, 255, 255, 0.1);
        padding: 1px 6px;
        border-radius: 3px;
        flex-shrink: 0;
    }

    .sync-badge-row {
        margin-bottom: 10px;
    }

    .sync-status-badge {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        padding: 3px 8px;
        border-radius: 4px;
        font-size: 0.78rem;
        font-weight: 600;
        letter-spacing: 0.03em;
        border: 1px solid transparent;
    }

    .sync-status-badge.local_only {
        background: rgba(138, 163, 153, 0.1);
        border-color: rgba(138, 163, 153, 0.3);
        color: var(--lcd-text-muted);
    }

    .sync-status-badge.synced {
        background: rgba(80, 250, 123, 0.1);
        border-color: rgba(80, 250, 123, 0.4);
        color: #50fa7b;
    }

    .sync-status-badge.local_ahead {
        background: rgba(255, 201, 102, 0.14);
        border-color: #ffc966;
        color: #ffc966;
        box-shadow: 0 0 6px rgba(255, 201, 102, 0.2);
    }

    .sync-status-badge.cloud_ahead {
        background: rgba(112, 214, 255, 0.14);
        border-color: #70d6ff;
        color: #70d6ff;
        box-shadow: 0 0 6px rgba(112, 214, 255, 0.2);
    }

    .sync-status-badge.conflict {
        background: rgba(255, 107, 107, 0.16);
        border-color: #ff6b6b;
        color: #ff8787;
        box-shadow: 0 0 6px rgba(255, 107, 107, 0.25);
    }

    .badge-version {
        opacity: 0.85;
        font-size: 0.72rem;
        margin-left: 2px;
        padding-left: 4px;
        border-left: 1px solid currentColor;
    }

    .card.status-local-ahead {
        border-color: rgba(255, 201, 102, 0.45);
    }

    .card.status-cloud-ahead {
        border-color: rgba(112, 214, 255, 0.45);
    }

    .card.status-conflict {
        border-color: rgba(255, 107, 107, 0.55);
    }

    .dates-comparison {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .card-info .date {
        font-size: 0.78rem;
        color: var(--lcd-text-muted);
    }

    .card-info .cloud-date {
        color: var(--lcd-text-soft);
        opacity: 0.85;
    }
</style>
