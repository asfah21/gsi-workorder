"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { UnitForm } from "@/components/unit-form";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DataResponse, Unit } from "@/lib/types";

export function UnitEditView({ unitId }: { unitId: number }) {
  const { user } = useSession();
  const [unit, setUnit] = useState<Unit | null>(null);
  const [error, setError] = useState("");
  const canWrite = user.role === "admin" || user.role === "planner";

  useEffect(() => {
    if (!canWrite) {
      return;
    }
    proxyGet<DataResponse<Unit>>(`/units/${unitId}`)
      .then((result) => setUnit(result.data))
      .catch((loadError) => {
        setError(loadError instanceof ApiError ? loadError.message : "Unit tidak dapat dimuat.");
      });
  }, [canWrite, unitId]);

  if (!canWrite) {
    return (
      <div className="space-y-3">
        <h1 className="text-2xl font-semibold">Edit unit</h1>
        <p role="alert" className="text-sm text-destructive">Role Anda tidak dapat mengubah unit.</p>
        <Link href={`/units/${unitId}`} className="text-sm underline underline-offset-4">Kembali</Link>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-3">
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>
        <Link href={`/units/${unitId}`} className="text-sm underline underline-offset-4">Kembali</Link>
      </div>
    );
  }

  if (!unit) {
    return <p role="status" className="text-sm text-muted-foreground">Memuat unit...</p>;
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <Link href={`/units/${unit.id}`} className="text-sm text-muted-foreground underline-offset-4 hover:underline">
          {unit.code}
        </Link>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">Edit unit</h1>
      </div>
      <div className="rounded-xl border border-border bg-background p-5 sm:p-6">
        <UnitForm unit={unit} />
      </div>
    </div>
  );
}
