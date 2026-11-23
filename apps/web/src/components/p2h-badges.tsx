import { Badge } from "@/components/ui/badge";
import type { WOCandidate } from "@/lib/types";
import { cn } from "@/lib/utils";

const statusLabels: Record<string, string> = {
  pending: "Pending",
  validated: "Tervalidasi",
  converted: "Dikonversi",
  approved: "Dikonversi",
  skipped: "Ditolak",
  auto_created: "WO otomatis",
  archived: "Arsip",
};

export function CandidateStatusBadge({ status }: { status: string }) {
  return (
    <Badge
      variant="secondary"
      className={cn(
        "h-5 border-transparent px-2",
        status === "pending" && "bg-amber-500/10 text-amber-700 dark:text-amber-300",
        status === "validated" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
        (status === "converted" || status === "approved") &&
          "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        status === "auto_created" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
        status === "skipped" && "bg-muted text-muted-foreground",
        status === "archived" && "bg-transparent text-muted-foreground",
      )}
    >
      {statusLabels[status] || status || "-"}
    </Badge>
  );
}

const priorityLabels: Record<WOCandidate["suggested_priority"], string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

export function PriorityBadge({
  priority,
}: {
  priority: WOCandidate["suggested_priority"];
}) {
  return (
    <Badge
      variant="outline"
      className={cn(
        "h-5 px-2",
        priority === "critical" && "border-destructive/30 text-destructive",
        priority === "high" &&
          "border-amber-500/30 text-amber-700 dark:text-amber-300",
        priority === "medium" && "border-sky-500/30 text-sky-700 dark:text-sky-300",
        priority === "low" && "text-muted-foreground",
      )}
    >
      {priorityLabels[priority]}
    </Badge>
  );
}

export function P2HStateBadge({ state }: { state: string }) {
  const label = state ? state.charAt(0).toUpperCase() + state.slice(1) : "-";
  return (
    <Badge
      variant="secondary"
      className={cn(
        "h-5 border-transparent px-2 capitalize",
        state === "approved" && "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        state === "submitted" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
        state === "rejected" && "bg-destructive/10 text-destructive",
        state === "draft" && "bg-muted text-muted-foreground",
      )}
    >
      {label}
    </Badge>
  );
}
