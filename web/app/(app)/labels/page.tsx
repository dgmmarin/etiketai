"use client";

import { useState, useRef } from "react";
import Link from "next/link";
import { Upload, Loader2, Trash2, Download } from "lucide-react";
import { useLabels, useUploadLabel, useDeleteLabel } from "@/lib/hooks/useLabels";
import { useAuthStore } from "@/lib/stores/authStore";
import { hasRole } from "@/lib/utils/roleGuard";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { STATUS_COLORS, formatDate } from "@/lib/utils/formatters";
import { API_URL } from "@/lib/api/client";

const STATUS_OPTIONS = ["pending", "processing", "needs_review", "confirmed", "failed"];

export default function LabelsPage() {
  const role = useAuthStore((s) => s.user?.role);
  const token = useAuthStore((s) => s.accessToken);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data, isPending } = useLabels({
    page,
    per_page: 20,
    status: statusFilter || undefined,
  });
  const uploadMut = useUploadLabel();
  const deleteMut = useDeleteLabel();

  async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 10 * 1024 * 1024) {
      alert("Fișierul depășește 10 MB");
      return;
    }
    await uploadMut.mutateAsync(file);
    e.target.value = "";
  }

  async function handleExport() {
    setExporting(true);
    try {
      const res = await fetch(`${API_URL}/v1/labels/export`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Export eșuat");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `etichete-export-${new Date().toISOString().slice(0, 10)}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      alert("Exportul a eșuat.");
    } finally {
      setExporting(false);
    }
  }

  function handleStatusChange(v: string) {
    setStatusFilter(v === "all" ? "" : v);
    setPage(1);
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Etichete</h1>
        <div className="flex items-center gap-2">
          {hasRole(role, "admin") && (
            <Button variant="outline" onClick={handleExport} disabled={exporting}>
              {exporting ? <Loader2 className="animate-spin" /> : <Download />}
              Export CSV
            </Button>
          )}
          {hasRole(role, "operator") && (
            <>
              <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={handleFileChange} />
              <Button onClick={() => fileInputRef.current?.click()} disabled={uploadMut.isPending}>
                {uploadMut.isPending ? <Loader2 className="animate-spin" /> : <Upload />}
                Încarcă etichetă
              </Button>
            </>
          )}
        </div>
      </div>

      {uploadMut.isError && (
        <p className="text-sm text-destructive">Eroare la încărcare: {uploadMut.error.message}</p>
      )}

      {/* Filters */}
      <div className="flex items-center gap-3">
        <Select value={statusFilter || "all"} onValueChange={handleStatusChange}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="Toate statusurile" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Toate statusurile</SelectItem>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s} value={s}>{s}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            {data ? `${data.total} etichete` : "Etichete"}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isPending ? (
            <div className="flex justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : !data?.labels?.length ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              {statusFilter ? "Nicio etichetă cu acest status." : "Nicio etichetă. Încarcă prima ta etichetă."}
            </p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">Fișier</th>
                  <th className="px-4 py-3 text-left font-medium">Status</th>
                  <th className="px-4 py-3 text-left font-medium">Data</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {data.labels.map((label) => (
                  <tr key={label.id} className="border-b last:border-0 hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3">
                      <Link href={`/labels/${label.id}`} className="text-primary hover:underline truncate block max-w-xs">
                        {label.original_filename}
                      </Link>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[label.status] ?? "bg-gray-100 text-gray-700"}`}>
                        {label.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{formatDate(label.created_at)}</td>
                    <td className="px-4 py-3 text-right">
                      {hasRole(role, "admin") && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-destructive hover:text-destructive"
                          onClick={() => setDeleteId(label.id)}
                          disabled={deleteMut.isPending}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={!!deleteId} onOpenChange={(o) => !o && setDeleteId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Ștergi eticheta?</AlertDialogTitle>
            <AlertDialogDescription>
              Această acțiune este ireversibilă. Eticheta și toate datele asociate vor fi șterse permanent.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Anulează</AlertDialogCancel>
            <AlertDialogAction onClick={() => { deleteMut.mutate(deleteId!); setDeleteId(null); }}>
              Șterge
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {data && data.total > 20 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1}>
            Anterior
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            Pagina {page} din {Math.ceil(data.total / 20)}
          </span>
          <Button variant="outline" size="sm" onClick={() => setPage((p) => p + 1)} disabled={page >= Math.ceil(data.total / 20)}>
            Următor
          </Button>
        </div>
      )}
    </div>
  );
}
