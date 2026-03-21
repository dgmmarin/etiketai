"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/stores/authStore";

/**
 * If the access token is already in Zustand memory, render immediately.
 * Otherwise attempt a silent refresh via the httpOnly cookie.
 * If the cookie is absent or expired, redirect to /login.
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { accessToken, setToken, clearSession } = useAuthStore();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (accessToken) {
      setReady(true);
      return;
    }

    // No token in memory — try silent refresh from httpOnly cookie.
    // Works for: hard refresh, new tab, direct URL navigation.
    fetch("/api/auth/refresh", { method: "POST" })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (data?.access_token) {
          setToken(data.access_token);
          setReady(true);
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
