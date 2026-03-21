import { create } from "zustand";
import { registerTokenAccessors } from "@/lib/api/client";

export type Role = "admin" | "operator" | "viewer";

export interface AuthUser {
  id: string;
  email: string;
  workspaceId: string;
  role: Role;
  isSuperAdmin?: boolean;
}

interface AuthState {
  accessToken: string | null;
  user: AuthUser | null;
  setSession: (token: string, user: AuthUser) => void;
  setToken: (token: string) => void;
  clearSession: () => void;
}

export const useAuthStore = create<AuthState>((set, get) => {
  // Register token accessors so client.ts can read/write tokens
  // without a circular import or runtime require()
  registerTokenAccessors(
    () => get().accessToken,
    (token) => set({ accessToken: token })
  );

  return {
    accessToken: null,
    user: null,

    setSession: (token, user) => {
      set({ accessToken: token, user });
    },

    setToken: (token) => {
      set({ accessToken: token });
    },

    clearSession: () => {
      set({ accessToken: null, user: null });
    },
  };
});
