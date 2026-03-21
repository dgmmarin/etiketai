"use client";

import { useAuthStore } from "@/lib/stores/authStore";
import { useWorkspaceProfile } from "@/lib/hooks/useWorkspace";
import { useLabels } from "@/lib/hooks/useLabels";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tag, Package, Users, TrendingUp } from "lucide-react";
import { PLAN_LABELS } from "@/lib/utils/formatters";

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const { data: ws } = useWorkspaceProfile();
  const { data: labels } = useLabels({ per_page: 5 });

  const quotaUsed = ws ? Math.round((ws.labels_used / ws.label_quota) * 100) : 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Bun venit, {user?.email?.split("@")[0]}</h1>
        <p className="text-muted-foreground">{ws?.name}</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Etichete totale</CardTitle>
            <Tag className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{labels?.total ?? "—"}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Utilizate / Limită</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {ws ? `${ws.labels_used} / ${ws.label_quota}` : "—"}
            </div>
            {ws && (
              <div className="mt-2 h-2 rounded-full bg-muted">
                <div
                  className="h-2 rounded-full bg-primary transition-all"
                  style={{ width: `${Math.min(quotaUsed, 100)}%` }}
                />
              </div>
            )}
            <p className="mt-1 text-xs text-muted-foreground">{quotaUsed}% din cotă</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Plan</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold capitalize">{ws?.plan ?? "—"}</div>
            <p className="text-xs text-muted-foreground">{ws ? PLAN_LABELS[ws.plan] ?? ws.plan : ""}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Rol</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold capitalize">{user?.role ?? "—"}</div>
          </CardContent>
        </Card>
      </div>

      {labels?.labels && labels.labels.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Etichete recente</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              {labels.labels.slice(0, 5).map((l) => (
                <li key={l.id} className="flex items-center justify-between text-sm">
                  <span className="truncate text-muted-foreground max-w-[200px]">{l.original_filename}</span>
                  <span className="ml-2 rounded-full bg-muted px-2 py-0.5 text-xs capitalize">{l.status}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
