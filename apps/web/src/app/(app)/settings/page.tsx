import type { Metadata } from "next";
import { Suspense } from "react";

import { SettingsView } from "@/components/settings-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Pengaturan | GSI Work Order",
  description: "Pengaturan global P2H dan usulan WO",
};

export default function SettingsPage() {
  return (
    <Suspense fallback={<SettingsPageSkeleton />}>
      <SettingsView />
    </Suspense>
  );
}

function SettingsPageSkeleton() {
  return (
    <div className="space-y-6" role="status" aria-label="Memuat pengaturan">
      <Skeleton className="h-9 w-48" />
      <Skeleton className="h-4 w-96 max-w-full" />
      <div className="grid gap-4 lg:grid-cols-12">
        <Skeleton className="h-80 rounded-xl lg:col-span-7" />
        <Skeleton className="h-80 rounded-xl lg:col-span-5" />
      </div>
    </div>
  );
}
