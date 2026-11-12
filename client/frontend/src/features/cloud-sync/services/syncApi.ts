/*
    Kizuna Editor - Cloud Sync API Service (Issue #13, #14)
    Handles project push/pull communication with Kizuna Cloud.
*/
import type { CloudPushPayload, CloudPushResult } from "../types/sync.ts";
import { authStorage, DEFAULT_CLOUD_URL, AuthApiError } from "./authApi.ts";
import type { ApiResponse } from "../types/auth.ts";

export const syncApi = {
    /**
     * Push a project snapshot to Kizuna Cloud (Issue #13)
     */
    async pushProject(
        payload: CloudPushPayload,
        token: string,
        customBaseUrl?: string
    ): Promise<CloudPushResult> {
        if (!token) {
            throw new AuthApiError("Authentication token is required to backup to the cloud", 401);
        }

        const baseUrl = customBaseUrl || authStorage.getCloudUrl() || DEFAULT_CLOUD_URL;

        // 1. If running natively in Wails and binding exists, we can use the Go backend or direct fetch
        const url = `${baseUrl.replace(/\/+$/, "")}/api/v1/sync/push`;

        let response: Response;
        try {
            response = await fetch(url, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Accept: "application/json",
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify(payload),
            });
        } catch (networkError) {
            throw new AuthApiError(
                `Unable to connect to Kizuna Cloud at ${baseUrl}. Ensure the server is online.`,
                0,
                networkError instanceof Error ? networkError.message : String(networkError)
            );
        }

        let body: ApiResponse<CloudPushResult>;
        try {
            body = (await response.json()) as ApiResponse<CloudPushResult>;
        } catch {
            throw new AuthApiError(
                `Invalid server response (${response.status} ${response.statusText})`,
                response.status
            );
        }

        if (!response.ok || !body.success) {
            const errorItem = body.error;
            const message = errorItem?.message || `Sync push rejected with status ${response.status}`;
            throw new AuthApiError(message, response.status, errorItem?.details);
        }

        if (!body.data) {
            throw new AuthApiError("Server returned an empty synchronization response", response.status);
        }

        return body.data;
    },
};
