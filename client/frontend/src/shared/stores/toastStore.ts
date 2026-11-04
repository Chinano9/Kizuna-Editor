/*
    Kizuna Editor - Toast Notification Store
*/
import { writable } from "svelte/store";

export type ToastType = "success" | "error" | "info" | "warning";

export interface ToastMessage {
    id: string;
    message: string;
    type: ToastType;
    durationMs: number;
}

const { subscribe, update } = writable<ToastMessage[]>([]);

export const toastStore = {
    subscribe,
    show: (message: string, type: ToastType = "info", durationMs = 3500): string => {
        const id = Math.random().toString(36).substring(2, 9);
        update((toasts) => [...toasts, { id, message, type, durationMs }]);

        if (durationMs > 0) {
            setTimeout(() => {
                toastStore.dismiss(id);
            }, durationMs);
        }
        return id;
    },
    success: (msg: string, durationMs = 3500): string => toastStore.show(msg, "success", durationMs),
    error: (msg: string, durationMs = 5000): string => toastStore.show(msg, "error", durationMs),
    warning: (msg: string, durationMs = 4000): string => toastStore.show(msg, "warning", durationMs),
    info: (msg: string, durationMs = 3500): string => toastStore.show(msg, "info", durationMs),
    dismiss: (id: string) => {
        update((toasts) => toasts.filter((t) => t.id !== id));
    },
    clear: () => {
        update(() => []);
    },
};
