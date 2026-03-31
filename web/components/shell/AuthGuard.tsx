"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore, type AuthUser, type Role } from "@/lib/stores/authStore";
import { toAuthUser } from "@/lib/api/auth";

interface JwtPayload {
  sub?: string;
  wid?: string;
  role?: string;
  email?: string;
  sadm?: boolean;
}

function parseJwtPayload(token: string): JwtPayload | null {
  try {
    const part = token.split(".")[1];
    const padded = part + "=".repeat((4 - (part.length % 4)) % 4);
    return JSON.parse(atob(padded.replace(/-/g, "+").replace(/_/g, "/")));
  } catch {
    return null;
  }
}

function userFromToken(token: string): AuthUser | null {
  const p = parseJwtPayload(token);
  if (!p?.sub || !p?.wid || !p?.role) return null;
  return {
    id: p.sub,
    email: p.email ?? "",
    workspaceId: p.wid,
    role: p.role as Role,
    isSuperAdmin: p.sadm ?? false,
  };
}

/**
 * If the access token is already in Zustand memory, render immediately.
 * Otherwise attempt a silent refresh via the httpOnly cookie.
 * If the cookie is absent or expired, redirect to /login.
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { accessToken, user, setSession, clearSession } = useAuthStore();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (accessToken && user) {
      setReady(true);
      return;
    }

    // No token (or no user) in memory — try silent refresh from httpOnly cookie.
    // Decodes the JWT payload to restore the full user object.
    // Works for: hard refresh, new tab, direct URL navigation.
    fetch("/api/auth/refresh", { method: "POST" })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (data?.access_token) {
          // Prefer user object from response (has wid+role from workspace-svc).
          // Fall back to JWT decode for backwards-compat.
          const restoredUser = data.user ? toAuthUser(data.user) : userFromToken(data.access_token);
          if (restoredUser) {
            setSession(data.access_token, restoredUser);
            setReady(true);
          } else {
            clearSession();
            router.replace("/login");
          }
        } else {
          clearSession();
          router.replace("/login");
        }
      })
      .catch(() => {
        clearSession();
        router.replace("/login");
      });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // run once on mount — token changes handled by re-renders

  if (!ready) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  return <>{children}</>;
}
