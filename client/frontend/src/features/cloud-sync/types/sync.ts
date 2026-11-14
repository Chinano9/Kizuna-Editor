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
    encrypted_payload?: string;
    encryption_algo?: string;
    is_encrypted?: boolean;
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
    encrypted_payload?: string;
    encryption_algo?: string;
    is_encrypted?: boolean;
    created_at?: string;
    updated_at?: string;
}

export interface SyncProgressPayload {
    operation: "push" | "pull";
    bytes_transferred: number;
    total_bytes: number;
    percentage: number;
    phase: "transferring" | "retrying" | "completed" | "failed";
    attempt: number;
    max_attempts: number;
}

export interface SyncRetryPayload {
    operation: "push" | "pull";
    attempt: number;
    max_attempts: number;
    wait_seconds: number;
    reason: string;
}
