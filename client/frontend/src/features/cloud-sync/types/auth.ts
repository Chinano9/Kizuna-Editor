/*
    Kizuna Editor - Cloud Auth Types
    Defines data structures for Kizuna Cloud authentication (Issue #12, #20).
*/

export interface AuthUser {
    id: string;
    username: string;
    email: string;
    created_at?: string;
}

export interface AuthResponseData {
    user: AuthUser;
    token: string;
}

export interface ApiErrorItem {
    code: number;
    message: string;
    details?: string;
}

export interface ApiResponse<T = unknown> {
    success: boolean;
    data?: T;
    error?: ApiErrorItem;
}

export interface LoginPayload {
    email: string;
    password: string;
}

export interface RegisterPayload {
    username: string;
    email: string;
    password: string;
}
