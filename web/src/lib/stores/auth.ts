import { writable } from "svelte/store";
import type { User } from "$api/types";
import apiClient from "$api/client";

interface AuthState {
  user: User | null;
  token: string | null;
  refreshToken: string | null;
  loading: boolean;
  error: string | null;
}

function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>({
    user: null,
    token: null,
    refreshToken: null,
    loading: false,
    error: null,
  });

  // Initialize auth state from localStorage
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("token");
    const refreshToken = localStorage.getItem("refresh_token");
    const userStr = localStorage.getItem("user");

    if (token && userStr) {
      try {
        const user = JSON.parse(userStr);
        apiClient.setToken(token);
        update((state) => ({ ...state, user, token, refreshToken }));
      } catch (e) {
        // Invalid stored data, clear it
        localStorage.clear();
      }
    }
  }

  return {
    subscribe,

    async login(username: string, password: string): Promise<boolean> {
      update((state) => ({ ...state, loading: true, error: null }));

      try {
        const response = await apiClient.login({ username, password });

        if (typeof window !== "undefined") {
          localStorage.setItem("token", response.token);
          localStorage.setItem("refresh_token", response.refresh_token);
          localStorage.setItem("user", JSON.stringify(response.user));
        }

        update((state) => ({
          ...state,
          user: response.user,
          token: response.token,
          refreshToken: response.refresh_token,
          loading: false,
          error: null,
        }));

        return true;
      } catch (error: any) {
        update((state) => ({
          ...state,
          loading: false,
          error: error.message || "Login failed",
        }));
        return false;
      }
    },

    async refreshAuth(): Promise<boolean> {
      let currentState: AuthState;
      const unsubscribe = subscribe((s) => {
        currentState = s;
      });
      unsubscribe();

      if (!currentState!.refreshToken) {
        return false;
      }

      try {
        const response = await apiClient.refreshToken(currentState!.refreshToken);

        if (typeof window !== "undefined") {
          localStorage.setItem("token", response.token);
        }

        update((s) => ({
          ...s,
          token: response.token,
        }));

        return true;
      } catch (error) {
        // Refresh failed, logout
        this.logout();
        return false;
      }
    },

    logout() {
      apiClient.logout();

      if (typeof window !== "undefined") {
        localStorage.removeItem("token");
        localStorage.removeItem("refresh_token");
        localStorage.removeItem("user");
      }

      set({
        user: null,
        token: null,
        refreshToken: null,
        loading: false,
        error: null,
      });
    },

    clearError() {
      update((state) => ({ ...state, error: null }));
    },
  };
}

export const auth = createAuthStore();
