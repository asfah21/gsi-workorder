import type { Metadata } from "next";
import { Suspense } from "react";

import { CandidatesView } from "@/components/candidates-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Usulan WO | PAMS",
  description: "Usulan work order dari temuan P2H",
};

export default function CandidatesPage() {
  return (
    <Suspense fallback={<CandidatesPageSkeleton />}>
      <CandidatesView />
    </Suspense>
  );
}

function CandidatesPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Memuat usulan WO">
      <Skeleton className="h-9 w-48" />
      <Skeleton className="h-4 w-96 max-w-full" />
      <Skeleton className="h-28 w-full rounded-xl" />
      <Skeleton className="h-96 w-full rounded-xl" />
    </div>
  );
}
