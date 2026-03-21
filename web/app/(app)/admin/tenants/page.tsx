"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, Building2, ChevronLeft, ChevronRight } from "lucide-react";
import { useAuthStore } from "@/lib/stores/authStore";
import { superAdminApi, type SuperAdminWorkspace } from "@/lib/api/admin";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { formatDate } from "@/lib/utils/formatters";

const PLAN_COLORS: Record<string, string> = {
  starter:    "bg-gray-100 text-gray-700",
  business:   "bg-blue-100 text-blue-700",
  enterprise: "bg-purple-100 text-purple-700",
  free:       "bg-yellow-100 text-yellow-700",
};

const PAGE_SIZE = 50;

export default function TenantsPage() {
  const isSuperAdmin = useAuthStore((s) => s.user?.isSuperAdmin);

  if (!isSuperAdmin) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-muted-foreground">Acces restricționat — doar super-administratori.</p>
      </div>
    );
  }

  return <TenantsContent />;
}

function TenantsContent() {
  const [offset, setOffset] = useState(0);

  const { data, isPending } = useQuery({
    queryKey: ["superadmin", "workspaces", offset],
    queryFn: () => superAdminApi.listWorkspaces({ limit: PAGE_SIZE, offset }),
  });

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Tenants</h1>
          {data && (
            <p className="text-sm text-muted-foreground">{data.total} workspace-uri totale</p>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Building2 className="h-4 w-4" />
            Toate workspace-urile
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isPending ? (
            <div className="flex justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : !data?.workspaces?.length ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              Niciun workspace găsit.
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-4 py-3 text-left font-medium">Workspace</th>
                    <th className="px-4 py-3 text-left font-medium">CUI</th>
                    <th className="px-4 py-3 text-left font-medium">Plan</th>
                    <th className="px-4 py-3 text-left font-medium">Etichete</th>
                    <th className="px-4 py-3 text-left font-medium">Creat</th>
                  </tr>
                </thead>
                <tbody>
                  {data.workspaces.map((ws: SuperAdminWorkspace) => (
                    <tr key={ws.id} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3">
                        <div>
                          <p className="font-medium">{ws.name}</p>
                          <p className="text-xs text-muted-foreground font-mono">{ws.id}</p>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{ws.cui || "—"}</td>
                      <td className="px-4 py-3">
                        <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium capitalize ${PLAN_COLORS[ws.plan] ?? "bg-gray-100 text-gray-700"}`}>
                          {ws.plan}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={ws.labels_used >= ws.label_quota ? "text-destructive font-medium" : ""}>
                          {ws.labels_used}
                        </span>
                        <span className="text-muted-foreground"> / {ws.label_quota}</span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">{formatDate(ws.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
            disabled={offset === 0}
          >
            <ChevronLeft className="h-4 w-4" />
            Anterior
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            Pagina {currentPage} din {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setOffset((o) => o + PAGE_SIZE)}
            disabled={currentPage >= totalPages}
          >
            Următor
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      )}
    </div>
  );
}
