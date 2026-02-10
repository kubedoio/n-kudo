import { ApiClient, ApiConfig } from '@/api/client';

export interface AuthState {
    config: ApiConfig | null;
    client: ApiClient | null;
}

let authState: AuthState = {
    config: null,
    client: null,
};

const listeners = new Set<(state: AuthState) => void>();

export const authStore = {
    getState: () => authState,
    subscribe: (listener: (state: AuthState) => void) => {
        listeners.add(listener);
        return () => listeners.delete(listener);
    },
    setConfig: (config: ApiConfig | null) => {
        authState = {
            config,
            client: config ? new ApiClient(config) : null,
        };
        listeners.forEach((l) => l(authState));

        // Optional persistence in sessionStorage if enabled
        if (config) {
            sessionStorage.setItem('nk_connection', JSON.stringify(config));
        } else {
            sessionStorage.removeItem('nk_connection');
        }
    },
    initFromSession: () => {
        const saved = sessionStorage.getItem('nk_connection');
        if (saved) {
            try {
                const config = JSON.parse(saved);
                authStore.setConfig(config);
            } catch (e) {
                console.error('Failed to load connection from session', e);
            }
        }
    }
};
