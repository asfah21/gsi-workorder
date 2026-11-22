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
    <div className="flex flex-col gap-4" role="status" aria-label="Memuat master unit">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-9 w-48" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-28 w-full rounded-xl" />
        ))}
      </div>
      <div className="flex gap-2">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-8 w-36" />
      </div>
      <Skeleton className="h-[28rem] w-full rounded-xl" />
    </div>
  );
}
