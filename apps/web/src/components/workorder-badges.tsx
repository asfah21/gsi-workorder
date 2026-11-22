import { Badge } from "@/components/ui/badge";
import type { WorkOrderMaintenanceType, WorkOrderPriority } from "@/lib/types";
import { cn } from "@/lib/utils";

export const workOrderStatusLabels: Record<string, string> = {
  open: "Open",
  in_progress: "In progress",
  waiting_part: "Waiting part",
  done: "Done",
  cancelled: "Cancelled",
};

export const workOrderMaintenanceLabels: Record<WorkOrderMaintenanceType, string> = {
  preventive: "Preventive",
  corrective: "Corrective",
  breakdown: "Breakdown",
  inspection: "Inspection",
};

export const workOrderPriorityLabels: Record<WorkOrderPriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

export function WorkOrderStatusBadge({ status }: { status: string }) {
  return (
    <Badge
      variant="secondary"
      className={cn(
        "h-5 border-transparent px-2",
        status === "open" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
        status === "in_progress" && "bg-amber-500/10 text-amber-700 dark:text-amber-300",
        status === "waiting_part" && "bg-orange-500/10 text-orange-700 dark:text-orange-300",
        status === "done" && "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        status === "cancelled" && "bg-muted text-muted-foreground",
      )}
    >
      {workOrderStatusLabels[status] || status || "-"}
    </Badge>
  );
}

export function WorkOrderPriorityBadge({ priority }: { priority: WorkOrderPriority | string }) {
  if (!priority) {
    return null;
  }
  return (
    <Badge
      variant="outline"
      className={cn(
        "h-5 px-2",
        priority === "critical" && "border-destructive/30 text-destructive",
        priority === "high" && "border-amber-500/30 text-amber-700 dark:text-amber-300",
        priority === "medium" && "border-sky-500/30 text-sky-700 dark:text-sky-300",
      )}
    >
      {workOrderPriorityLabels[priority as WorkOrderPriority] || priority}
    </Badge>
  );
}

export function WorkOrderMaintenanceBadge({ type }: { type: WorkOrderMaintenanceType | string }) {
  return (
    <Badge variant="outline" className="h-5 px-2">
      {workOrderMaintenanceLabels[type as WorkOrderMaintenanceType] || type || "-"}
    </Badge>
  );
}
