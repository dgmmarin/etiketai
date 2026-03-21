import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { labelsApi } from "@/lib/api/labels";

export const labelKeys = {
  all: ["labels"] as const,
  list: (params?: object) => [...labelKeys.all, "list", params] as const,
  detail: (id: string) => [...labelKeys.all, "detail", id] as const,
  status: (id: string) => [...labelKeys.all, "status", id] as const,
  compliance: (id: string) => [...labelKeys.all, "compliance", id] as const,
};

export function useLabels(params?: { status?: string; page?: number; per_page?: number }) {
  return useQuery({
    queryKey: labelKeys.list(params),
    queryFn: () => labelsApi.list(params),
    staleTime: 30_000,
  });
}

export function useLabelStatus(id: string, enabled = true) {
  return useQuery({
    queryKey: labelKeys.status(id),
    queryFn: () => labelsApi.getStatus(id),
    enabled,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === "confirmed" || status === "failed") return false;
      return 3_000;
    },
  });
}

export function useLabelCompliance(id: string, enabled = true) {
  return useQuery({
    queryKey: labelKeys.compliance(id),
    queryFn: () => labelsApi.getCompliance(id),
    enabled,
    staleTime: 60_000,
  });
}

export function useUploadLabel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => labelsApi.upload(file),
    onSuccess: () => qc.invalidateQueries({ queryKey: labelKeys.all }),
  });
}

export function useUpdateFields() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, fields, draft }: { id: string; fields: Record<string, string>; draft?: boolean }) =>
      labelsApi.updateFields(id, fields, draft),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: labelKeys.status(id) });
      qc.invalidateQueries({ queryKey: labelKeys.compliance(id) });
    },
  });
}

export function useConfirmLabel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => labelsApi.confirm(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: labelKeys.status(id) });
      qc.invalidateQueries({ queryKey: labelKeys.list() });
    },
  });
}

export function useDeleteLabel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => labelsApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: labelKeys.all }),
  });
}

export function useCreatePrintJob() {
  return useMutation({
    mutationFn: ({ id, opts }: { id: string; opts: Parameters<typeof labelsApi.createPrintJob>[1] }) =>
      labelsApi.createPrintJob(id, opts),
  });
}
