export function formatDate(iso: string) {
  return new Intl.DateTimeFormat("ro-RO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso));
}

export function formatDateShort(iso: string) {
  return new Intl.DateTimeFormat("ro-RO", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(new Date(iso));
}

export const STATUS_COLORS: Record<string, string> = {
  pending:      "bg-yellow-100 text-yellow-800",
  processing:   "bg-blue-100 text-blue-800",
  needs_review: "bg-orange-100 text-orange-800",
  confirmed:    "bg-green-100 text-green-800",
  failed:       "bg-red-100 text-red-800",
  queued:       "bg-gray-100 text-gray-700",
  ready:        "bg-green-100 text-green-800",
  printed:      "bg-purple-100 text-purple-800",
};

export const PLAN_LABELS: Record<string, string> = {
  starter:    "Starter — 49 lei/lună",
  business:   "Business — 149 lei/lună",
  enterprise: "Enterprise — 399 lei/lună",
};
