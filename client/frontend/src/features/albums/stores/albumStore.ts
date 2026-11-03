/*
    Kizuna Editor - Album Store (Feature: Albums - Dev 3)
*/
import { writable } from "svelte/store";
import type { Song } from "@shared/types/wails";

export interface AlbumData {
    id: number;
    title: string;
    artist?: string;
    description?: string;
    cover_image_path?: string;
    created_at?: string;
    songs?: Song[];
}

const { subscribe, set, update } = writable<AlbumData[]>([]);

export const albumStore = {
    subscribe,
    setAlbums: (albums: AlbumData[]) => set(albums),
    addAlbum: (album: AlbumData) => update((items) => [album, ...items]),
    removeAlbum: (id: number) => update((items) => items.filter((a) => a.id !== id)),
};
