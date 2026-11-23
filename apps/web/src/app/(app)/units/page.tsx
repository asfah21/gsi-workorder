import type { Metadata } from "next";
import { Suspense } from "react";

import { UnitsView } from "@/components/units-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Unit | PAMS",
  description: "Master unit dan HM/KM",
};

export default function UnitsPage() {
  return (
    <Suspense fallback={<UnitsPageSkeleton />}>
      <UnitsView />
    </Suspense>
  );
}

function UnitsPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Memuat halaman unit">
      <Skeleton className="h-9 w-40" />
      <Skeleton className="h-4 w-80 max-w-full" />
      <Skeleton className="h-28 w-full rounded-xl" />
      <Skeleton className="h-96 w-full rounded-xl" />
    </div>
  );
}
