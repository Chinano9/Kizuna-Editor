/*
    Kizuna Editor - Cloud Auth API Service
    Handles HTTP communication with Kizuna Cloud backend and session token persistence (Issue #12, #20).
*/

import type {
    AuthUser,
    AuthResponseData,
    ApiResponse,
    LoginPayload,
    RegisterPayload,
} from "../types/auth.ts";

export const DEFAULT_CLOUD_URL = "http://localhost:8080";

const STORAGE_KEYS = {
    TOKEN: "kizuna_cloud_token",
    USER: "kizuna_cloud_user",
    URL: "kizuna_cloud_url",
} as const;

export class AuthApiError extends Error {
    public code: number;
    public details?: string;

    constructor(message: string, code: number = 400, details?: string) {
        super(message);
        this.name = "AuthApiError";
        this.code = code;
        this.details = details;
    }
}

// Safe storage access (guards against SSR or non-browser environments)
export const authStorage = {
    getToken(): string | null {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                return localStorage.getItem(STORAGE_KEYS.TOKEN);
            }
        } catch {
            // Ignore storage access errors
        }
        return null;
    },

    getUser(): AuthUser | null {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                const raw = localStorage.getItem(STORAGE_KEYS.USER);
                if (raw) return JSON.parse(raw) as AuthUser;
            }
        } catch {
            // Ignore parse errors
        }
        return null;
    },

    getCloudUrl(): string {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                const url = localStorage.getItem(STORAGE_KEYS.URL);
                if (url) return url;
            }
        } catch {
            // Ignore storage errors
        }
        return DEFAULT_CLOUD_URL;
    },

    saveSession(token: string, user: AuthUser, cloudUrl?: string): void {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                localStorage.setItem(STORAGE_KEYS.TOKEN, token);
                localStorage.setItem(STORAGE_KEYS.USER, JSON.stringify(user));
                if (cloudUrl) {
                    localStorage.setItem(STORAGE_KEYS.URL, cloudUrl);
                }
            }
        } catch (err) {
            console.error("Failed to persist auth session to storage:", err);
        }
    },

    clearSession(): void {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                localStorage.removeItem(STORAGE_KEYS.TOKEN);
                localStorage.removeItem(STORAGE_KEYS.USER);
            }
        } catch (err) {
            console.error("Failed to clear auth session from storage:", err);
        }
    },

    setCloudUrl(url: string): void {
        try {
            if (typeof window !== "undefined" && window.localStorage) {
                localStorage.setItem(STORAGE_KEYS.URL, url);
            }
        } catch {
            // Ignore storage errors
        }
    },
};

/**
 * Execute HTTP requests to the Kizuna Cloud API with error handling and envelope decoding
 */
async function request<T>(
    endpoint: string,
    options: RequestInit = {},
    customBaseUrl?: string
): Promise<T> {
    const baseUrl = customBaseUrl || authStorage.getCloudUrl();
    const url = `${baseUrl.replace(/\/+$/, "")}${endpoint}`;

    const headers: Record<string, string> = {
        "Content-Type": "application/json",
        Accept: "application/json",
        ...(options.headers as Record<string, string>),
    };

    let response: Response;
    try {
        response = await fetch(url, {
            ...options,
            headers,
        });
    } catch (networkError) {
        throw new AuthApiError(
            `Unable to connect to Kizuna Cloud at ${baseUrl}. Ensure the server is running.`,
            0,
            networkError instanceof Error ? networkError.message : String(networkError)
        );
    }

    let payload: ApiResponse<T>;
    try {
        payload = (await response.json()) as ApiResponse<T>;
    } catch {
        throw new AuthApiError(
            `Invalid server response (${response.status} ${response.statusText})`,
            response.status
        );
    }

    if (!response.ok || !payload.success) {
        const errorItem = payload.error;
        const message = errorItem?.message || `Request failed with status ${response.status}`;
        throw new AuthApiError(message, response.status, errorItem?.details);
    }

    return payload.data as T;
}

export const authApi = {
    /**
     * Authenticate an existing user with email and password (Issue #12)
     */
    async login(payload: LoginPayload, baseUrl?: string): Promise<AuthResponseData> {
        return request<AuthResponseData>(
            "/api/v1/auth/login",
            {
                method: "POST",
                body: JSON.stringify(payload),
            },
            baseUrl
        );
    },

    /**
     * Register a new user account on Kizuna Cloud (Issue #12)
     */
    async register(payload: RegisterPayload, baseUrl?: string): Promise<AuthResponseData> {
        return request<AuthResponseData>(
            "/api/v1/auth/register",
            {
                method: "POST",
                body: JSON.stringify(payload),
            },
            baseUrl
        );
    },

    /**
     * Verify current session token and fetch user profile (Issue #12, #20)
     */
    async getMe(token: string, baseUrl?: string): Promise<AuthUser> {
        return request<AuthUser>(
            "/api/v1/auth/me",
            {
                method: "GET",
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            },
            baseUrl
        );
    },
};
