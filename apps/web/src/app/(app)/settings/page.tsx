import type { Metadata } from "next";
import { Suspense } from "react";

import { SettingsView } from "@/components/settings-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Pengaturan | PAMS",
  description: "Pengaturan global P2H, kandidat work order, dan PM otomatis",
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
    <div className="flex flex-col gap-4" role="status" aria-label="Memuat pengaturan">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-9 w-48" />
        <Skeleton className="h-4 w-96 max-w-full" />
      </div>
      <div className="grid gap-4 lg:grid-cols-12">
        <Skeleton className="h-80 rounded-xl lg:col-span-7" />
        <Skeleton className="h-80 rounded-xl lg:col-span-5" />
      </div>
      <Skeleton className="h-64 rounded-xl" />
    </div>
  );
}
