"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Tag,
  Package,
  Building2,
  CreditCard,
  Settings,
} from "lucide-react";
import { useAuthStore } from "@/lib/stores/authStore";
import { hasRole } from "@/lib/utils/roleGuard";
import { cn } from "@/lib/utils/cn";

const NAV = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/labels", label: "Etichete", icon: Tag },
  { href: "/products", label: "Produse", icon: Package },
  { href: "/workspace", label: "Workspace", icon: Building2 },
  { href: "/billing", label: "Abonament", icon: CreditCard },
];

export function Sidebar() {
  const pathname = usePathname();
  const role = useAuthStore((s) => s.user?.role);

  return (
    <aside className="flex h-full w-56 flex-col border-r bg-sidebar">
      <div className="flex h-14 items-center px-4 border-b">
        <span className="font-semibold text-lg tracking-tight">EtiketAI</span>
      </div>

      <nav className="flex-1 overflow-y-auto p-2 space-y-0.5">
        {NAV.map(({ href, label, icon: Icon }) => {
          const active = href === "/" ? pathname === "/" : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                active
                  ? "bg-sidebar-primary text-sidebar-primary-foreground"
                  : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </Link>
          );
        })}

        {hasRole(role, "admin") && (
          <Link
            href="/admin"
            className={cn(
              "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              pathname.startsWith("/admin")
                ? "bg-sidebar-primary text-sidebar-primary-foreground"
                : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            )}
          >
            <Settings className="h-4 w-4 shrink-0" />
            Admin
          </Link>
        )}
      </nav>
    </aside>
  );
}
