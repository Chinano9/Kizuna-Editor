/*
    Kizuna Editor - Definición de Rutas y Navegación
*/

export type ViewType = "dashboard" | "editor" | "albums" | "settings";

export interface RouteConfig {
    id: ViewType;
    label: string;
    icon?: string;
}

export const APP_ROUTES: RouteConfig[] = [
    { id: "dashboard", label: "Dashboard", icon: "home" },
    { id: "editor", label: "Editor", icon: "music" },
    { id: "albums", label: "Albums", icon: "disc" },
    { id: "settings", label: "Settings", icon: "settings" },
];
