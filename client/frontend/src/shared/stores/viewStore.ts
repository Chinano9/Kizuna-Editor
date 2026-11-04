/*
    Kizuna Editor - View Navigation Store
*/
import { writable } from "svelte/store";
import type { ViewType } from "@app/routes";

export type { ViewType } from "@app/routes";

export const currentView = writable<ViewType>("dashboard");

export function navigateTo(view: ViewType) {
    currentView.set(view);
}
