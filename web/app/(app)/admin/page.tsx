"use client";

import { useState } from "react";
import { Loader2, Save } from "lucide-react";
import { useAgentConfig, useUpdateAgentConfig, useAgentLogs, useMetrics, useRateLimits } from "@/lib/hooks/useAdmin";
import { adminApi } from "@/lib/api/admin";
import { useAuthStore } from "@/lib/stores/authStore";
import { hasRole } from "@/lib/utils/roleGuard";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { formatDate } from "@/lib/utils/formatters";
import type { AgentConfig } from "@/lib/api/admin";

const PROVIDERS = ["anthropic", "ollama", "rules_engine"];

function ProviderRow({
  label,
  value,
  onChange,
}: {
  label: string;
  value: { provider: string; model?: string; endpoint?: string };
  onChange: (v: { provider: string; model?: string; endpoint?: string }) => void;
}) {
  return (
    <div className="rounded-md border p-3 space-y-2">
      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">{label}</p>
      <div className="grid gap-2 sm:grid-cols-3">
        <div className="space-y-1">
          <Label className="text-xs">Provider</Label>
          <Select value={value.provider} onValueChange={(v) => onChange({ ...value, provider: v })}>
            <SelectTrigger className="h-8 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PROVIDERS.map((p) => <SelectItem key={p} value={p}>{p}</SelectItem>)}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Model</Label>
          <Input className="h-8 text-xs" value={value.model ?? ""} onChange={(e) => onChange({ ...value, model: e.target.value })} placeholder="claude-sonnet-4-6" />
        </div>
        {value.provider === "ollama" && (
          <div className="space-y-1">
            <Label className="text-xs">Endpoint</Label>
            <Input className="h-8 text-xs" value={value.endpoint ?? ""} onChange={(e) => onChange({ ...value, endpoint: e.target.value })} placeholder="http://localhost:11434" />
          </div>
        )}
      </div>
    </div>
  );
}

export default function AdminPage() {
  const role = useAuthStore((s) => s.user?.role);

  if (!hasRole(role, "admin")) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-muted-foreground">Acces restricționat — doar administratori.</p>
      </div>
    );
  }

  return <AdminContent />;
}

function AdminContent() {
  const { data: config, isPending: configPending } = useAgentConfig();
  const updateConfig = useUpdateAgentConfig();
  const { data: logsData, isPending: logsPending } = useAgentLogs();
  const { data: metrics } = useMetrics();
  const { data: rateLimits, setLimits } = useRateLimits();

  const [draft, setDraft] = useState<Partial<AgentConfig> | null>(null);
  const effective = draft ?? config;
  const workspaceId = useAuthStore((s) => s.user?.workspaceId ?? "");
  const [testResult, setTestResult] = useState<Record<string, string>>({});
  const [testLoading, setTestLoading] = useState<string | null>(null);

  async function handleTest(type: "vision" | "translation" | "validation") {
    setTestLoading(type);
    setTestResult((prev) => ({ ...prev, [type]: "" }));
    try {
      const res = await adminApi.testAgentConfig(workspaceId, type);
      setTestResult((prev) => ({
        ...prev,
        [type]: res.success
          ? `✓ OK${res.latency_ms ? ` — ${res.latency_ms}ms` : ""}`
          : `✗ ${res.error ?? "Eșuat"}`,
      }));
    } catch (e: unknown) {
      setTestResult((prev) => ({ ...prev, [type]: `✗ ${e instanceof Error ? e.message : "Eroare"}` }));
    } finally {
      setTestLoading(null);
    }
  }

  const [rlUpm, setRlUpm] = useState<string>("");
  const [rlPpd, setRlPpd] = useState<string>("");

  async function handleSaveConfig() {
    if (!draft) return;
    await updateConfig.mutateAsync(draft);
    setDraft(null);
  }

  async function handleSaveRateLimits(e: React.FormEvent) {
    e.preventDefault();
    await setLimits.mutateAsync({
      uploads_per_minute: rlUpm ? parseInt(rlUpm) : undefined,
      prints_per_day: rlPpd ? parseInt(rlPpd) : undefined,
    });
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Admin</h1>

      {/* Metrics strip */}
      {metrics && (
        <div className="grid gap-3 sm:grid-cols-4">
          {[
            { label: "Etichete totale", value: metrics.total_labels },
            { label: "Confirmate", value: metrics.confirmed_labels },
            { label: "Eșuate", value: metrics.failed_labels },
            { label: "Încredere medie", value: `${Math.round(metrics.avg_confidence * 100)}%` },
          ].map(({ label, value }) => (
            <Card key={label}>
              <CardContent className="pt-4">
                <p className="text-xs text-muted-foreground">{label}</p>
                <p className="text-2xl font-bold">{value}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Tabs defaultValue="agent">
        <TabsList>
          <TabsTrigger value="agent">Agent AI</TabsTrigger>
          <TabsTrigger value="logs">Loguri</TabsTrigger>
          <TabsTrigger value="limits">Rate Limits</TabsTrigger>
        </TabsList>

        {/* Agent config */}
        <TabsContent value="agent">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-base">Configurare Agent AI</CardTitle>
              {draft && (
                <Button size="sm" onClick={handleSaveConfig} disabled={updateConfig.isPending}>
                  {updateConfig.isPending ? <Loader2 className="animate-spin" /> : <Save />}
                  Salvează
                </Button>
              )}
            </CardHeader>
            <CardContent className="space-y-3">
              {configPending ? (
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              ) : effective ? (
                <>
                  {(["vision", "translation", "validation"] as const).map((type) => {
                    const fieldMap = { vision: effective.vision, translation: effective.translation, validation: effective.validation };
                    const labelMap = { vision: "Vision", translation: "Translation", validation: "Validation" };
                    const defaultMap = { vision: { provider: "anthropic" }, translation: { provider: "anthropic" }, validation: { provider: "rules_engine" } };
                    return (
                      <div key={type}>
                        <ProviderRow
                          label={labelMap[type]}
                          value={fieldMap[type] ?? defaultMap[type]}
                          onChange={(v) => setDraft((d) => ({ ...(d ?? effective!), [type]: v }))}
                        />
                        <div className="mt-1 flex items-center gap-2 px-1">
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 text-xs"
                            onClick={() => handleTest(type)}
                            disabled={testLoading === type}
                          >
                            {testLoading === type ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
                            Test conexiune
                          </Button>
                          {testResult[type] && (
                            <span className={`text-xs ${testResult[type].startsWith("✓") ? "text-green-600" : "text-destructive"}`}>
                              {testResult[type]}
                            </span>
                          )}
                        </div>
                      </div>
                    );
                  })}
                  {effective.fallback && (
                    <ProviderRow
                      label="Fallback"
                      value={effective.fallback}
                      onChange={(v) => setDraft((d) => ({ ...(d ?? effective!), fallback: v }))}
                    />
                  )}
                </>
              ) : (
                <p className="text-sm text-muted-foreground">Nicio configurare găsită.</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Agent logs */}
        <TabsContent value="logs">
          <Card>
            <CardContent className="p-0">
              {logsPending ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : !logsData?.logs?.length ? (
                <p className="py-8 text-center text-sm text-muted-foreground">Niciun log.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-3 py-2 text-left font-medium">Tip</th>
                        <th className="px-3 py-2 text-left font-medium">Provider</th>
                        <th className="px-3 py-2 text-left font-medium">Model</th>
                        <th className="px-3 py-2 text-left font-medium">Status</th>
                        <th className="px-3 py-2 text-left font-medium">Latență</th>
                        <th className="px-3 py-2 text-left font-medium">Tokens</th>
                        <th className="px-3 py-2 text-left font-medium">Data</th>
                      </tr>
                    </thead>
                    <tbody>
                      {logsData.logs.map((log) => (
                        <tr key={log.id} className="border-b last:border-0">
                          <td className="px-3 py-2">{log.agent_type}</td>
                          <td className="px-3 py-2">{log.provider}</td>
                          <td className="px-3 py-2 text-muted-foreground">{log.model}</td>
                          <td className="px-3 py-2">
                            <span className={`rounded px-1.5 py-0.5 ${log.success ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>
                              {log.success ? "OK" : "ERR"}
                            </span>
                          </td>
                          <td className="px-3 py-2">{log.latency_ms}ms</td>
                          <td className="px-3 py-2">{log.tokens_used}</td>
                          <td className="px-3 py-2 text-muted-foreground">{formatDate(log.created_at)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Rate limits */}
        <TabsContent value="limits">
          <Card>
            <CardContent className="pt-6">
              <form onSubmit={handleSaveRateLimits} className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label>Încărcări/minut</Label>
                    <Input
                      type="number"
                      placeholder={String(rateLimits?.uploads_per_minute ?? "")}
                      value={rlUpm}
                      onChange={(e) => setRlUpm(e.target.value)}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label>Tipăriri/zi</Label>
                    <Input
                      type="number"
                      placeholder={String(rateLimits?.prints_per_day ?? "")}
                      value={rlPpd}
                      onChange={(e) => setRlPpd(e.target.value)}
                    />
                  </div>
                </div>
                <Button type="submit" disabled={setLimits.isPending}>
                  {setLimits.isPending && <Loader2 className="animate-spin" />}
                  Actualizează
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
