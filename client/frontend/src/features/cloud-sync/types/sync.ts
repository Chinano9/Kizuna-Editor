/*
    Kizuna Editor - Cloud Sync Types (Issue #13, #14)
    Data structures for project synchronization with Kizuna Cloud.
*/
import type { Track, AudioVersion } from "@shared/types/wails";

export interface CloudPushPayload {
    project_id: number;
    title: string;
    bpm: number;
    time_signature: string;
    key_signature: string;
    base_version: number;
    tracks: Track[];
    audio_versions: AudioVersion[];
}

export interface CloudPushResult {
    project_id: number;
    new_version: number;
    synced_at: string;
    status: string;
}

export interface CloudProject {
    id: number;
    owner_id: string;
    title: string;
    bpm: number;
    time_signature: string;
    key_signature: string;
    version: number;
    tracks?: Track[];
    audio_versions?: AudioVersion[];
    created_at?: string;
    updated_at?: string;
}
