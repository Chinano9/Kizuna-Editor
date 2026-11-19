<!--
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
-->
<script lang="ts">
    /*
        Kizuna Editor - Share Project Modal (Issue #15)
        Invites collaborators to a cloud-backed project by email or username.
    */
    import { createEventDispatcher } from "svelte";
    import Modal from "@shared/components/ui/Modal.svelte";
    import Button from "@shared/components/ui/Button.svelte";
    import { toastStore } from "@shared/stores/toastStore";
    import { ShareProject } from "@wails/go/main/App";

    export let isOpen: boolean = false;
    export let cloudId: string | number = "";
    export let projectTitle: string = "Proyecto";

    let collaboratorIdentifier: string = "";
    let isSubmitting: boolean = false;

    const dispatch = createEventDispatcher<{
        close: void;
        shared: { cloudId: string | number; collaborator: string };
    }>();

    function close() {
        if (isSubmitting) return;
        collaboratorIdentifier = "";
        isOpen = false;
        dispatch("close");
    }

    export async function submitShare() {
        const identifier = collaboratorIdentifier.trim();
        if (!identifier) {
            toastStore.warning("Ingresa el correo o nombre de usuario del colaborador.");
            return;
        }

        if (!cloudId || String(cloudId).trim() === "" || String(cloudId) === "0") {
            toastStore.warning("Debes respaldar el proyecto en la nube antes de invitar colaboradores");
            return;
        }

        isSubmitting = true;
        try {
            await ShareProject(String(cloudId), identifier);
            toastStore.success(`¡Invitación enviada a ${identifier} con éxito!`);
            dispatch("shared", { cloudId, collaborator: identifier });
            close();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : String(err);
            toastStore.error(message);
        } finally {
            isSubmitting = false;
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "Enter" && !isSubmitting) {
            submitShare();
        }
    }
</script>

<Modal {isOpen} title="Compartir Proyecto" maxWidth="440px" on:close={close}>
    <div class="share-dialog">
        <p class="description">
            Invita a otro músico a colaborar en <strong class="project-name">"{projectTitle}"</strong>.
            Tendrá permisos de lectura y descarga en Kizuna Cloud.
        </p>

        <div class="input-group">
            <label for="collab-input">Email o Nombre de Usuario:</label>
            <input
                id="collab-input"
                type="text"
                class="lcd-input"
                placeholder="ej. colega@kizuna.io o usuario"
                bind:value={collaboratorIdentifier}
                on:keydown={handleKeydown}
                disabled={isSubmitting}
                autocomplete="off"
            />
        </div>
    </div>

    <svelte:fragment slot="footer">
        <Button variant="ghost" size="md" disabled={isSubmitting} on:click={close}>
            Cancelar
        </Button>
        <Button variant="primary" size="md" disabled={isSubmitting} on:click={submitShare}>
            {isSubmitting ? "Enviando..." : "Invitar Colaborador"}
        </Button>
    </svelte:fragment>
</Modal>

<style>
    .share-dialog {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .description {
        font-size: 1rem;
        margin: 0;
        color: var(--lcd-text, #c8ff9f);
        line-height: 1.4;
    }

    .project-name {
        color: var(--lcd-amber, #ffc966);
    }

    .input-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    label {
        font-size: 0.9rem;
        color: var(--lcd-text-soft, #a0e58d);
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .lcd-input {
        background-color: var(--app-bg, #050607);
        border: 1px solid var(--lcd-border, #243524);
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 1.15rem;
        padding: 8px 12px;
        border-radius: 4px;
        outline: none;
        transition: border-color 0.2s ease, box-shadow 0.2s ease;
    }

    .lcd-input:focus {
        border-color: var(--lcd-button-border, #c3933b);
        box-shadow: 0 0 8px rgba(255, 201, 102, 0.3);
    }

    .lcd-input:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }
</style>
