import { writable } from 'svelte/store';

// Tipos de vista permitidos
export type ViewType = 'dashboard' | 'editor';

// Por defecto arrancamos en el Dashboard
export const currentView = writable<ViewType>('dashboard');
