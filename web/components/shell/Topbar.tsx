"use client";

import { useRouter } from "next/navigation";
import { LogOut, User } from "lucide-react";
import { useAuthStore } from "@/lib/stores/authStore";
import { Button } from "@/components/ui/button";

export function Topbar() {
  const router = useRouter();
  const { user, clearSession } = useAuthStore();

  async function handleLogout() {
    // API route reads the httpOnly cookie and calls the backend
    await fetch("/api/auth/logout", { method: "POST" }).catch(() => {});
    clearSession();
    router.push("/login");
  }

  return (
    <header className="flex h-14 items-center justify-between border-b bg-background px-4">
      <div />
      <div className="flex items-center gap-3">
        <span className="flex items-center gap-2 text-sm text-muted-foreground">
          <User className="h-4 w-4" />
          {user?.email}
          {user?.role && (
            <span className="rounded bg-muted px-1.5 py-0.5 text-xs font-medium capitalize">
              {user.role}
            </span>
          )}
        </span>
        <Button variant="ghost" size="icon" onClick={handleLogout} title="Deconectare">
          <LogOut className="h-4 w-4" />
        </Button>
      </div>
    </header>
  );
}
