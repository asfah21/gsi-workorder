import type { Metadata } from "next";
import { Suspense } from "react";

import { Skeleton } from "@/components/ui/skeleton";
import { WorkOrdersView } from "@/components/workorders-view";

export const metadata: Metadata = {
  title: "Work Order | GSI Work Order",
  description: "Work order manual dan siklus hidup pekerjaan maintenance",
};

export default function WorkOrdersPage() {
  return (
    <Suspense fallback={<WorkOrdersPageSkeleton />}>
      <WorkOrdersView />
    </Suspense>
  );
}

function WorkOrdersPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Memuat work order">
      <Skeleton className="h-9 w-48" />
      <Skeleton className="h-4 w-96 max-w-full" />
      <Skeleton className="h-28 w-full rounded-xl" />
      <Skeleton className="h-96 w-full rounded-xl" />
    </div>
  );
}
