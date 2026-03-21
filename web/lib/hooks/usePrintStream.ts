"use client";

import { useEffect, useRef, useState } from "react";
import { useAuthStore } from "@/lib/stores/authStore";
import type { PrintJob } from "@/lib/api/labels";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface StreamState {
  job: PrintJob | null;
  error: string | null;
  isDone: boolean;
}

export function usePrintStream(labelId: string, jobId: string | null) {
  const token = useAuthStore((s) => s.accessToken);
  const [state, setState] = useState<StreamState>({ job: null, error: null, isDone: false });
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!jobId || !token) return;

    const ctrl = new AbortController();
    abortRef.current = ctrl;

    (async () => {
      try {
        const res = await fetch(
          `${API_URL}/v1/labels/${labelId}/print/pdf/${jobId}/stream`,
          {
            headers: { Authorization: `Bearer ${token}` },
            signal: ctrl.signal,
          }
        );

        if (!res.ok || !res.body) {
          setState({ job: null, error: "Stream failed to connect", isDone: true });
          return;
        }

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buf += decoder.decode(value, { stream: true });
          const blocks = buf.split("\n\n");
          buf = blocks.pop() ?? "";

          for (const block of blocks) {
            const eventLine = block.split("\n").find((l) => l.startsWith("event:"));
            const dataLine = block.split("\n").find((l) => l.startsWith("data:"));

            const event = eventLine?.replace("event:", "").trim();
            const rawData = dataLine?.replace("data:", "").trim();

            if (!rawData) continue;

            if (event === "error") {
              const parsed = JSON.parse(rawData);
              setState({ job: null, error: parsed.error ?? "Unknown error", isDone: true });
              return;
            }

            if (event === "status") {
              const job: PrintJob = JSON.parse(rawData);
              const isDone = job.status === "ready" || job.status === "failed" || job.status === "printed";
              setState({ job, error: null, isDone });
              if (isDone) return;
            }
          }
        }
      } catch (err) {
        if ((err as Error).name !== "AbortError") {
          setState((prev) => ({ ...prev, error: "Stream disconnected", isDone: true }));
        }
      }
    })();

    return () => ctrl.abort();
  }, [labelId, jobId, token]);

  return state;
}
