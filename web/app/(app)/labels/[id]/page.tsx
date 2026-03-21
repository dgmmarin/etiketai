"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, CheckCircle, Printer, Loader2, AlertCircle } from "lucide-react";
import {
  useLabelStatus,
  useLabelCompliance,
  useUpdateFields,
  useConfirmLabel,
  useCreatePrintJob,
} from "@/lib/hooks/useLabels";
import { usePrintStream } from "@/lib/hooks/usePrintStream";
import { useAuthStore } from "@/lib/stores/authStore";
import { hasRole } from "@/lib/utils/roleGuard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { STATUS_COLORS } from "@/lib/utils/formatters";

export default function LabelDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const role = useAuthStore((s) => s.user?.role);

  const { data: label, isPending } = useLabelStatus(id);
  const { data: compliance } = useLabelCompliance(id, label?.status === "needs_review" || label?.status === "confirmed");

  const updateFields = useUpdateFields();
  const confirmLabel = useConfirmLabel();
  const createPrintJob = useCreatePrintJob();

  const [editedFields, setEditedFields] = useState<Record<string, string>>({});
  const [printJobId, setPrintJobId] = useState<string | null>(null);
  const { job: printJob, error: streamError, isDone } = usePrintStream(id, printJobId);

  if (isPending) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!label) {
    return <p className="text-muted-foreground">Eticheta nu a fost găsită.</p>;
  }

  const isProcessing = label.status === "processing" || label.status === "pending";

  async function handleSave(draft = true) {
    await updateFields.mutateAsync({ id, fields: editedFields, draft });
    setEditedFields({});
  }

  async function handleConfirm() {
    if (Object.keys(editedFields).length > 0) await handleSave(false);
    await confirmLabel.mutateAsync(id);
  }

  async function handlePrint() {
    const result = await createPrintJob.mutateAsync({ id, opts: { copies: 1 } });
    setPrintJobId(result.job_id);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-xl font-bold">{label.id}</h1>
          <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[label.status] ?? ""}`}>
            {label.status}
          </span>
        </div>
      </div>

      {isProcessing && (
        <Card>
          <CardContent className="flex items-center gap-3 py-4">
            <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
            <span className="text-sm">Procesare AI în curs...</span>
          </CardContent>
        </Card>
      )}

      {/* Fields editor */}
      {label.fields && Object.keys(label.fields).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Câmpuri detectate</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 sm:grid-cols-2">
              {Object.entries(label.fields).map(([key, fv]) => (
                <div key={key} className="space-y-1">
                  <label className="text-xs font-medium text-muted-foreground capitalize">
                    {key.replace(/_/g, " ")}
                    <span className="ml-1 text-[10px] opacity-60">({Math.round(fv.confidence * 100)}%)</span>
                  </label>
                  <Input
                    value={editedFields[key] ?? fv.value}
                    onChange={(e) => setEditedFields((prev) => ({ ...prev, [key]: e.target.value }))}
                    className="h-8 text-sm"
                    disabled={label.status === "confirmed"}
                  />
                </div>
              ))}
            </div>

            {label.status !== "confirmed" && Object.keys(editedFields).length > 0 && (
              <div className="mt-4 flex gap-2">
                <Button variant="outline" size="sm" onClick={() => handleSave(true)} disabled={updateFields.isPending}>
                  Salvează ciornă
                </Button>
                <Button size="sm" onClick={handleConfirm} disabled={confirmLabel.isPending}>
                  <CheckCircle className="h-4 w-4" />
                  Confirmă și scade din cotă
                </Button>
              </div>
            )}

            {label.status === "needs_review" && Object.keys(editedFields).length === 0 && (
              <Button className="mt-4" size="sm" onClick={handleConfirm} disabled={confirmLabel.isPending}>
                <CheckCircle className="h-4 w-4" />
                Confirmă eticheta
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Compliance */}
      {compliance && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              Conformitate
              <span className={`text-sm font-bold ${compliance.score >= 80 ? "text-green-600" : compliance.score >= 50 ? "text-orange-500" : "text-red-500"}`}>
                {compliance.score}/100
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {compliance.missing_fields.length > 0 && (
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Câmpuri lipsă</p>
                <div className="flex flex-wrap gap-1">
                  {compliance.missing_fields.map((f) => (
                    <span key={f} className="rounded bg-destructive/10 px-2 py-0.5 text-xs text-destructive">
                      {f.replace(/_/g, " ")}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {compliance.warnings.length > 0 && (
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Avertismente</p>
                <ul className="space-y-1">
                  {compliance.warnings.map((w, i) => (
                    <li key={i} className="flex items-start gap-1.5 text-xs text-orange-600">
                      <AlertCircle className="h-3 w-3 mt-0.5 shrink-0" />
                      {w}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <p className="text-xs text-muted-foreground">Regulament: {compliance.regulation}</p>
          </CardContent>
        </Card>
      )}

      {/* Print */}
      {hasRole(role, "operator") && label.status === "confirmed" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Tipărire PDF</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {!printJobId ? (
              <Button onClick={handlePrint} disabled={createPrintJob.isPending}>
                {createPrintJob.isPending ? <Loader2 className="animate-spin" /> : <Printer />}
                Generează PDF
              </Button>
            ) : (
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  {!isDone && <Loader2 className="h-4 w-4 animate-spin text-blue-500" />}
                  <span>Status: <strong>{printJob?.status ?? "așteptare..."}</strong></span>
                </div>
                {streamError && <p className="text-xs text-destructive">{streamError}</p>}
                {printJob?.status === "ready" && printJob.pdf_url && (
                  <a
                    href={printJob.pdf_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"
                  >
                    Descarcă PDF
                  </a>
                )}
                {printJob?.status === "failed" && (
                  <p className="text-sm text-destructive">{printJob.error ?? "Tipărire eșuată"}</p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
