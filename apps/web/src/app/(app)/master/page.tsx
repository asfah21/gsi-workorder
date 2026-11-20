import type { Metadata } from "next";
import { Suspense } from "react";

import { MasterView } from "@/components/master-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Master Unit | GSI Work Order",
  description: "Master unit model dan engine model",
};

export default function MasterPage() {
  return (
    <Suspense fallback={<MasterPageSkeleton />}>
      <MasterView />
    </Suspense>
  );
}

function MasterPageSkeleton() {
  return (
    <div className="space-y-8" role="status" aria-label="Memuat master unit">
      <Skeleton className="h-9 w-48" />
      <Skeleton className="h-4 w-80 max-w-full" />
      <Skeleton className="h-40 w-full rounded-xl" />
      <Skeleton className="h-80 w-full rounded-xl" />
      <Skeleton className="h-40 w-full rounded-xl" />
      <Skeleton className="h-80 w-full rounded-xl" />
    </div>
  );
}
