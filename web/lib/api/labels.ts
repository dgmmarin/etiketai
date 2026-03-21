import { api } from "./client";

export interface FieldValue {
  value: string;
  confidence: number;
  source: string;
}

export interface LabelListItem {
  id: string;
  workspace_id: string;
  original_filename: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface LabelStatus {
  id: string;
  status: string;
  fields: Record<string, FieldValue>;
  created_at: string;
  updated_at: string;
}

export interface ComplianceResult {
  label_id: string;
  score: number;
  missing_fields: string[];
  warnings: string[];
  regulation: string;
  checked_at: string;
}

export interface PrintJobResult {
  job_id: string;
  status: string;
}

export interface PrintJob {
  job_id: string;
  label_id: string;
  workspace_id: string;
  status: string;
  pdf_url?: string;
  error?: string;
  created_at: string;
  completed_at?: string;
}

export interface LabelsListResponse {
  labels: LabelListItem[];
  total: number;
}

export const labelsApi = {
  list: (params?: { status?: string; page?: number; per_page?: number }) =>
    api.get<LabelsListResponse>("/v1/labels?" + new URLSearchParams(
      Object.fromEntries(Object.entries(params ?? {}).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)]))
    )),

  getStatus: (id: string) =>
    api.get<LabelStatus>(`/v1/labels/${id}/status`),

  getCompliance: (id: string) =>
    api.get<ComplianceResult>(`/v1/labels/${id}/compliance`),

  updateFields: (id: string, fields: Record<string, string>, draft = false) =>
    api.patch<LabelStatus>(`/v1/labels/${id}/fields${draft ? "?draft=true" : ""}`, { fields }),

  confirm: (id: string) =>
    api.post<{ success: boolean }>(`/v1/labels/${id}/confirm`),

  delete: (id: string) =>
    api.delete<{ success: boolean }>(`/v1/labels/${id}`),

  upload: (file: File) => {
    const fd = new FormData();
    fd.append("image", file);
    return api.upload<LabelListItem>("/v1/labels/upload", fd);
  },

  createPrintJob: (id: string, opts: { format?: string; size?: string; copies?: number; printer_id?: string }) =>
    api.post<PrintJobResult>(`/v1/labels/${id}/print/pdf`, opts),

  getPrintJob: (labelId: string, jobId: string) =>
    api.get<PrintJob>(`/v1/labels/${labelId}/print/pdf/${jobId}`),

  getReprintURL: (labelId: string, jobId: string) =>
    api.get<{ job_id: string; pdf_url: string }>(`/v1/labels/${labelId}/print/pdf/${jobId}/url`),

  export: () =>
    api.get<Blob>("/v1/labels/export"),
};
