import type { Metadata } from "next";
import { Suspense } from "react";

import { P2HReportsView } from "@/components/p2h-reports-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "P2H | PAMS",
  description: "Snapshot laporan P2H dan temuan",
};

export default function P2HPage() {
  return (
    <Suspense fallback={<P2HPageSkeleton />}>
      <P2HReportsView />
    </Suspense>
  );
}

function P2HPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Memuat laporan P2H">
      <Skeleton className="h-9 w-48" />
      <Skeleton className="h-4 w-96 max-w-full" />
      <Skeleton className="h-28 w-full rounded-xl" />
      <Skeleton className="h-96 w-full rounded-xl" />
    </div>
  );
}
