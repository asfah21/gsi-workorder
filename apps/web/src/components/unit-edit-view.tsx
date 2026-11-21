"use client";

import { ArrowLeft, CircleAlert, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { UnitForm } from "@/components/unit-form";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DataResponse, Unit } from "@/lib/types";

export function UnitEditView({ unitId }: { unitId: number }) {
  const { user } = useSession();
  const [unit, setUnit] = useState<Unit | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const canWrite = user.role === "admin" || user.role === "planner";

  useEffect(() => {
    if (!canWrite) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    proxyGet<DataResponse<Unit>>(`/units/${unitId}`)
      .then((result) => {
        if (!cancelled) {
          setUnit(result.data);
        }
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(
            loadError instanceof ApiError
              ? loadError.message
              : "Unit tidak dapat dimuat.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [canWrite, unitId]);

  if (!canWrite) {
    return (
      <div className="space-y-4">
        <h1 className="text-xl font-semibold">Edit unit</h1>
        <Alert className="border-sky-500/30 bg-sky-500/5">
          <ShieldCheck aria-hidden="true" />
          <AlertTitle>Akses baca saja</AlertTitle>
          <AlertDescription>Role Anda tidak dapat mengubah unit.</AlertDescription>
        </Alert>
        <Link
          href={`/units/${unitId}`}
          className="inline-flex h-9 items-center gap-2 text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Kembali ke detail unit
        </Link>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-4">
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Unit tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
        <Link
          href={`/units/${unitId}`}
          className="inline-flex h-9 items-center gap-2 text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Kembali ke detail unit
        </Link>
      </div>
    );
  }

  if (loading || !unit) {
    return (
      <div className="space-y-6">
        <div>
          <Skeleton className="h-4 w-16" />
          <Skeleton className="mt-3 h-8 w-40" />
        </div>
        <Skeleton className="h-96 rounded-xl" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardContent>
          <Link
            href={`/units/${unit.id}`}
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            {unit.code}
          </Link>
          <h1 className="mt-2 text-xl font-semibold">Edit unit</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Perbarui identitas, model, atau status unit.
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent>
          <UnitForm unit={unit} />
        </CardContent>
      </Card>
    </div>
  );
}
