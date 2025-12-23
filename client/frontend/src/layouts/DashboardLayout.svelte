<script lang="ts">
    // Side bar layout for the dashboard view logic pending
</script>

<div class="layout-container">
    <aside class="sidebar">
        <div class="brand">
            <h2>Kizuna</h2>
        </div>

        <nav>
            <button class="nav-item active">🎵 Library</button>
            <button class="nav-item">💿 Albums (TODO)</button>
            <button class="nav-item">⚙️ Settings (TODO)</button>
        </nav>

        <footer>
            <button class="quick-idea">⚡️Quick idea (TODO)</button>
        </footer>

        <div class="user-profile">
            <span>Local</span>
        </div>
    </aside>

    <main class="content-area">
        <slot />
    </main>
</div>

<style>
    .layout-container {
        display: flex;
        height: 100vh;
        width: 100vw;
        background-color: #1e1e1e;
        color: #e0e0e0;
    }

    /* SIDEBAR */
    .sidebar {
        width: 250px;
        background-color: #252526;
        border-right: 1px solid #333;
        display: flex;
        flex-direction: column;
        flex-shrink: 0;
    }

    .brand {
        padding: 20px;
        border-bottom: 1px solid #333;
    }
    .brand h2 {
        margin: 0;
        color: #4ec9b0;
        letter-spacing: 1px;
        font-size: 1.5rem;
    }

    nav {
        flex: 1;
        padding: 20px 10px;
        display: flex;
        flex-direction: column;
        gap: 5px;
    }

    .nav-item {
        background: transparent;
        border: none;
        color: #aaa;
        text-align: left;
        padding: 10px 15px;
        cursor: pointer;
        border-radius: 6px;
        font-size: 0.95rem;
        transition: all 0.2s;
    }
    .nav-item:hover {
        background-color: #2d2d2e;
        color: white;
    }
    .nav-item.active {
        background-color: #37373d;
        color: white;
        font-weight: bold;
        border-left: 3px solid #4ec9b0;
    }

    footer {
        padding: 15px;
        border-top: 1px solid #333;
    }

    .quick-idea {
        width: 100%;
        background-color: #0e639c;
        border: none;
        color: white;
        padding: 10px;
        font-size: 0.9rem;
        cursor: pointer;
        border-radius: 4px;
        transition: background 0.2s;
    }

    .user-profile {
        padding: 15px;
        border-top: 1px solid #333;
        font-size: 0.85rem;
        color: #666;
        text-align: center;
    }

    /* CONTENT AREA */
    .content-area {
        flex: 1;
        overflow: hidden;
        position: relative;
        display: flex;
        flex-direction: column;
    }
</style>
