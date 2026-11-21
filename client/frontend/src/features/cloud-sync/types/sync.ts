/*
    Kizuna Editor - Cloud Sync Types (Issue #13, #14, #16, #22)
    Data structures for project synchronization and collaborative locking with Kizuna Cloud.
*/
import type { Track, AudioVersion } from "@shared/types/wails";

export interface ProjectLock {
    project_id: number;
    user_id: string;
    username: string;
    acquired_at: string;
    expires_at: string;
}

export interface CloudPushPayload {
    project_id: number;
    title: string;
    bpm: number;
    time_signature: string;
    key_signature: string;
    base_version: number;
    content_hash?: string;
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
    content_hash?: string;
    lock_released?: boolean;
}

export interface CloudProject {
    id: number;
    owner_id: string;
    collaborators?: string[];
    title: string;
    bpm: number;
    time_signature: string;
    key_signature: string;
    version: number;
    content_hash?: string;
    tracks?: Track[];
    audio_versions?: AudioVersion[];
    encrypted_payload?: string;
    encryption_algo?: string;
    is_encrypted?: boolean;
    active_lock?: ProjectLock | null;
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
