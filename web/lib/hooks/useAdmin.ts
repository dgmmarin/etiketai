import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api/admin";
import { useAuthStore } from "@/lib/stores/authStore";

export function useAgentConfig() {
  const workspaceId = useAuthStore((s) => s.user?.workspaceId ?? "");
  return useQuery({
    queryKey: ["admin", "agent-config", workspaceId],
    queryFn: () => adminApi.getAgentConfig(workspaceId),
    enabled: !!workspaceId,
    staleTime: 5 * 60_000,
  });
}

export function useUpdateAgentConfig() {
  const qc = useQueryClient();
  const workspaceId = useAuthStore((s) => s.user?.workspaceId ?? "");
  return useMutation({
    mutationFn: (config: Parameters<typeof adminApi.updateAgentConfig>[1]) =>
      adminApi.updateAgentConfig(workspaceId, config),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "agent-config", workspaceId] }),
  });
}

export function useAgentLogs(page = 1) {
  const workspaceId = useAuthStore((s) => s.user?.workspaceId ?? "");
  return useQuery({
    queryKey: ["admin", "agent-logs", workspaceId, page],
    queryFn: () => adminApi.getAgentLogs(workspaceId, { page, per_page: 20 }),
    enabled: !!workspaceId,
  });
}

export function useMetrics() {
  const workspaceId = useAuthStore((s) => s.user?.workspaceId ?? "");
  return useQuery({
    queryKey: ["admin", "metrics", workspaceId],
    queryFn: () => adminApi.getMetrics(workspaceId),
    enabled: !!workspaceId,
    staleTime: 60_000,
  });
}

export function useRateLimits() {
  const workspaceId = useAuthStore((s) => s.user?.workspaceId ?? "");
  const qc = useQueryClient();

  const query = useQuery({
    queryKey: ["admin", "rate-limits", workspaceId],
    queryFn: () => adminApi.getRateLimits(workspaceId),
    enabled: !!workspaceId,
  });

  const mutation = useMutation({
    mutationFn: (limits: Parameters<typeof adminApi.setRateLimits>[1]) =>
      adminApi.setRateLimits(workspaceId, limits),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "rate-limits", workspaceId] }),
  });

  return { ...query, setLimits: mutation };
}
