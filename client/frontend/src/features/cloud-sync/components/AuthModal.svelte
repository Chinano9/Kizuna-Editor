<script lang="ts">
    /*
        Kizuna Editor - Cloud Authentication Modal (Issue #12, #20)
        Provides Login and Registration dialogs with Kizuna Cloud backend.
    */
    import { syncStore } from "../stores/syncStore";
    import Modal from "@shared/components/ui/Modal.svelte";
    import Button from "@shared/components/ui/Button.svelte";
    import { toastStore } from "@shared/stores/toastStore";

    export let isOpen: boolean = false;

    let mode: "login" | "register" = "login";
    let isLoading = false;
    let errorMessage = "";

    // Form fields
    let loginEmail = "";
    let loginPassword = "";

    let registerUsername = "";
    let registerEmail = "";
    let registerPassword = "";
    let registerConfirmPassword = "";

    // Advanced server URL settings
    let showSettings = false;
    let serverUrl = $syncStore.cloudUrl;

    $: if (!isOpen) {
        // Reset transient error when closing
        errorMessage = "";
    }

    function switchMode(newMode: "login" | "register") {
        mode = newMode;
        errorMessage = "";
    }

    function handleClose() {
        isOpen = false;
        errorMessage = "";
    }

    async function handleLogin() {
        errorMessage = "";
        const email = loginEmail.trim();
        const password = loginPassword;

        if (!email) {
            errorMessage = "Ingresa tu correo electrónico.";
            return;
        }
        if (!password) {
            errorMessage = "Ingresa tu contraseña.";
            return;
        }

        isLoading = true;
        try {
            if (serverUrl.trim() && serverUrl !== $syncStore.cloudUrl) {
                syncStore.setCloudUrl(serverUrl.trim());
            }

            await syncStore.login({ email, password });
            toastStore.success(`Bienvenido a Kizuna Cloud (${email})`);
            loginEmail = "";
            loginPassword = "";
            isOpen = false;
        } catch (err) {
            errorMessage = err instanceof Error ? err.message : "Error al iniciar sesión.";
        } finally {
            isLoading = false;
        }
    }

    async function handleRegister() {
        errorMessage = "";
        const username = registerUsername.trim();
        const email = registerEmail.trim();
        const password = registerPassword;
        const confirmPassword = registerConfirmPassword;

        if (!username) {
            errorMessage = "Ingresa un nombre de usuario.";
            return;
        }
        if (!email || !email.includes("@")) {
            errorMessage = "Ingresa un correo electrónico válido.";
            return;
        }
        if (!password || password.length < 6) {
            errorMessage = "La contraseña debe tener al menos 6 caracteres.";
            return;
        }
        if (password !== confirmPassword) {
            errorMessage = "Las contraseñas no coinciden.";
            return;
        }

        isLoading = true;
        try {
            if (serverUrl.trim() && serverUrl !== $syncStore.cloudUrl) {
                syncStore.setCloudUrl(serverUrl.trim());
            }

            await syncStore.register({ username, email, password });
            toastStore.success(`Cuenta creada exitosamente. ¡Bienvenido, ${username}!`);
            registerUsername = "";
            registerEmail = "";
            registerPassword = "";
            registerConfirmPassword = "";
            isOpen = false;
        } catch (err) {
            errorMessage = err instanceof Error ? err.message : "Error al registrar la cuenta.";
        } finally {
            isLoading = false;
        }
    }
</script>

<Modal
    {isOpen}
    title={mode === "login" ? "KIZUNA CLOUD - INICIAR SESIÓN" : "KIZUNA CLOUD - CREAR CUENTA"}
    maxWidth="460px"
    on:close={handleClose}
>
    <div class="auth-container">
        <!-- Tabs -->
        <div class="auth-tabs" role="tablist">
            <button
                type="button"
                role="tab"
                class="tab-btn"
                class:active={mode === "login"}
                aria-selected={mode === "login"}
                on:click={() => switchMode("login")}
            >
                Iniciar Sesión
            </button>
            <button
                type="button"
                role="tab"
                class="tab-btn"
                class:active={mode === "register"}
                aria-selected={mode === "register"}
                on:click={() => switchMode("register")}
            >
                Crear Cuenta
            </button>
        </div>

        {#if errorMessage}
            <div class="error-banner" role="alert">
                <span class="error-icon">⚠️</span>
                <span>{errorMessage}</span>
            </div>
        {/if}

        {#if mode === "login"}
            <!-- Login Form -->
            <form on:submit|preventDefault={handleLogin} class="auth-form">
                <div class="form-group">
                    <label for="login-email">Correo Electrónico</label>
                    <input
                        id="login-email"
                        type="email"
                        bind:value={loginEmail}
                        placeholder="tu@correo.com"
                        autocomplete="email"
                        disabled={isLoading}
                        required
                    />
                </div>

                <div class="form-group">
                    <label for="login-password">Contraseña</label>
                    <input
                        id="login-password"
                        type="password"
                        bind:value={loginPassword}
                        placeholder="••••••••"
                        autocomplete="current-password"
                        disabled={isLoading}
                        required
                    />
                </div>

                <div class="actions-row">
                    <Button
                        type="submit"
                        variant="primary"
                        disabled={isLoading}
                    >
                        {isLoading ? "Conectando..." : "Entrar a Kizuna Cloud"}
                    </Button>
                </div>
            </form>
        {:else}
            <!-- Register Form -->
            <form on:submit|preventDefault={handleRegister} class="auth-form">
                <div class="form-group">
                    <label for="reg-username">Nombre de Usuario</label>
                    <input
                        id="reg-username"
                        type="text"
                        bind:value={registerUsername}
                        placeholder="ej. jimi_hendrix"
                        autocomplete="username"
                        disabled={isLoading}
                        required
                    />
                </div>

                <div class="form-group">
                    <label for="reg-email">Correo Electrónico</label>
                    <input
                        id="reg-email"
                        type="email"
                        bind:value={registerEmail}
                        placeholder="tu@correo.com"
                        autocomplete="email"
                        disabled={isLoading}
                        required
                    />
                </div>

                <div class="form-group">
                    <label for="reg-password">Contraseña (Mín. 6 caracteres)</label>
                    <input
                        id="reg-password"
                        type="password"
                        bind:value={registerPassword}
                        placeholder="••••••••"
                        autocomplete="new-password"
                        disabled={isLoading}
                        required
                    />
                </div>

                <div class="form-group">
                    <label for="reg-confirm">Confirmar Contraseña</label>
                    <input
                        id="reg-confirm"
                        type="password"
                        bind:value={registerConfirmPassword}
                        placeholder="••••••••"
                        autocomplete="new-password"
                        disabled={isLoading}
                        required
                    />
                </div>

                <div class="actions-row">
                    <Button
                        type="submit"
                        variant="primary"
                        disabled={isLoading}
                    >
                        {isLoading ? "Creando cuenta..." : "Registrar y Conectar"}
                    </Button>
                </div>
            </form>
        {/if}

        <!-- Advanced Server URL settings toggle -->
        <div class="server-config">
            <button
                type="button"
                class="settings-toggle"
                on:click={() => (showSettings = !showSettings)}
            >
                ⚙️ {showSettings ? "Ocultar ajustes de servidor" : "Configurar servidor remoto"}
            </button>

            {#if showSettings}
                <div class="server-input-group">
                    <label for="cloud-server-url">URL de Kizuna Cloud</label>
                    <input
                        id="cloud-server-url"
                        type="url"
                        bind:value={serverUrl}
                        placeholder="http://localhost:8080"
                        disabled={isLoading}
                    />
                    <small>Por defecto se conecta al servidor local en el puerto 8080.</small>
                </div>
            {/if}
        </div>
    </div>
</Modal>

<style>
    .auth-container {
        display: flex;
        flex-direction: column;
        gap: 16px;
        color: var(--lcd-text, #c8ff9f);
    }

    /* Tabs */
    .auth-tabs {
        display: flex;
        border-bottom: 1px solid var(--lcd-border, #2d382e);
        gap: 8px;
    }

    .tab-btn {
        background: transparent;
        border: none;
        border-bottom: 2px solid transparent;
        color: #888;
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 1.1rem;
        padding: 8px 16px;
        cursor: pointer;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        transition: all 0.15s ease;
    }

    .tab-btn:hover {
        color: var(--lcd-text, #c8ff9f);
    }

    .tab-btn.active {
        color: var(--lcd-text, #c8ff9f);
        border-bottom-color: var(--lcd-text, #c8ff9f);
        text-shadow: 0 0 6px rgba(200, 255, 159, 0.4);
    }

    /* Error banner */
    .error-banner {
        display: flex;
        align-items: center;
        gap: 10px;
        background: rgba(255, 85, 85, 0.15);
        border: 1px solid #ff5555;
        color: #ff8888;
        padding: 10px 14px;
        border-radius: 4px;
        font-size: 0.95rem;
    }

    .error-icon {
        font-size: 1.1rem;
    }

    /* Form */
    .auth-form {
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .form-group label {
        font-family: var(--lcd-font-ui, "VT323", monospace);
        font-size: 1rem;
        text-transform: uppercase;
        color: #aaa;
        letter-spacing: 0.5px;
    }

    .form-group input {
        background-color: var(--app-bg-alt, #0f1311);
        border: 1px solid var(--lcd-border, #2d382e);
        color: var(--lcd-text, #c8ff9f);
        font-family: var(--lcd-font-main, monospace);
        font-size: 1rem;
        padding: 9px 12px;
        border-radius: 4px;
        outline: none;
        transition: border-color 0.15s ease, box-shadow 0.15s ease;
    }

    .form-group input:focus {
        border-color: var(--lcd-text, #c8ff9f);
        box-shadow: 0 0 8px rgba(200, 255, 159, 0.25);
    }

    .form-group input:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .actions-row {
        margin-top: 8px;
        display: flex;
        justify-content: flex-end;
    }

    /* Server URL configuration */
    .server-config {
        border-top: 1px dashed var(--lcd-border, #2d382e);
        padding-top: 12px;
        margin-top: 4px;
    }

    .settings-toggle {
        background: transparent;
        border: none;
        color: #777;
        font-size: 0.85rem;
        cursor: pointer;
        padding: 0;
        text-align: left;
        transition: color 0.15s ease;
    }

    .settings-toggle:hover {
        color: #aaa;
    }

    .server-input-group {
        display: flex;
        flex-direction: column;
        gap: 4px;
        margin-top: 8px;
    }

    .server-input-group label {
        font-size: 0.85rem;
        color: #888;
    }

    .server-input-group input {
        background-color: var(--app-bg-alt, #0f1311);
        border: 1px solid var(--lcd-border, #2d382e);
        color: var(--lcd-text, #c8ff9f);
        font-family: monospace;
        font-size: 0.9rem;
        padding: 6px 10px;
        border-radius: 4px;
        outline: none;
    }

    .server-input-group small {
        color: #666;
        font-size: 0.8rem;
    }
</style>
