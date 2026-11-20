"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DashboardSummary, DataResponse } from "@/lib/types";
import { formatNumber } from "@/lib/time";

export function DashboardView() {
  const { user } = useSession();
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    proxyGet<DataResponse<DashboardSummary>>("/dashboard/summary")
      .then((result) => {
        if (!cancelled) {
          setSummary(result.data);
        }
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(
            loadError instanceof ApiError
              ? loadError.message
              : "Ringkasan dashboard tidak dapat dimuat.",
          );
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <p className="text-sm text-muted-foreground">Selamat datang</p>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">
          {user.name || user.nik}
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
          Ringkasan master unit dan pembacaan HM/KM. Modul P2H, work order, PM, dan inventory
          belum termasuk Milestone 1.
        </p>
      </div>

      {error ? (
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <section aria-labelledby="ringkasan-unit" className="space-y-3">
        <h2 id="ringkasan-unit" className="text-lg font-semibold tracking-tight">
          Ringkasan unit
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <SummaryCard label="Total unit" value={summary?.units_total} />
          <SummaryCard label="Unit aktif" value={summary?.units_active} />
          <SummaryCard label="Unit breakdown" value={summary?.units_breakdown} />
          <SummaryCard label="Model terdaftar" value={summary?.unit_models} />
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        <section aria-labelledby="aktivitas-meter" className="rounded-xl border border-border bg-background p-5">
          <h2 id="aktivitas-meter" className="text-base font-semibold">
            Pembacaan HM/KM
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Reading manual 30 hari terakhir.
          </p>
          <p className="mt-5 text-3xl font-semibold tabular-nums">
            {summary ? formatNumber(summary.readings_last_30_days) : "-"}
          </p>
        </section>
        <section aria-labelledby="akses-cepat" className="rounded-xl border border-border bg-background p-5">
          <h2 id="akses-cepat" className="text-base font-semibold">
            Akses cepat
          </h2>
          <div className="mt-4 flex flex-wrap gap-2">
            <Link
              href="/units"
              className="inline-flex min-h-10 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              Daftar unit
            </Link>
            <Link
              href="/master"
              className="inline-flex min-h-10 items-center rounded-md border border-border px-4 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              Master model
            </Link>
          </div>
        </section>
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value?: number }) {
  return (
    <div className="rounded-xl border border-border bg-background p-5">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-3 text-3xl font-semibold tabular-nums">
        {typeof value === "number" ? formatNumber(value) : "-"}
      </p>
    </div>
  );
}
