"use client";

import {
  Activity,
  ArrowRight,
  Boxes,
  CheckCircle2,
  CircleAlert,
  Plus,
  RefreshCcw,
  TriangleAlert,
  Truck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { roleLabels, useSession } from "@/components/app-shell";
import { FleetStatusChart } from "@/components/fleet-status-chart";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DashboardSummary, DataResponse } from "@/lib/types";
import { formatNumber, formatWita } from "@/lib/time";
import { cn } from "@/lib/utils";

export function DashboardView() {
  const { user } = useSession();
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [today, setToday] = useState("");

  const canWrite = user.role === "admin" || user.role === "planner";

  useEffect(() => {
    setToday(
      formatWita(new Date(), {
        weekday: "long",
        day: "numeric",
        month: "long",
        year: "numeric",
      }),
    );
  }, []);

  const loadSummary = useCallback(async () => {
    setLoading(true);
    try {
      const result = await proxyGet<DataResponse<DashboardSummary>>("/dashboard/summary");
      setSummary(result.data);
      setError("");
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Ringkasan dashboard tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSummary();
  }, [loadSummary]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm text-muted-foreground">{today || "Ringkasan hari ini"}</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight sm:text-3xl">
            Selamat datang, {user.name || user.nik || "Pengguna"}
          </h1>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <Badge variant="secondary" className="h-6 px-2.5">
              {user.role ? roleLabels[user.role] : "-"}
            </Badge>
            <span>Ringkasan master unit dan pembacaan HM/KM.</span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => void loadSummary()}
            disabled={loading}
            className="size-11 p-0"
            aria-label="Muat ulang ringkasan"
            title="Muat ulang ringkasan"
          >
            <RefreshCcw className={loading ? "animate-spin" : undefined} aria-hidden="true" />
          </Button>
          {canWrite ? (
            <Link
              href="/units/new"
              className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            >
              <Plus className="size-4" aria-hidden="true" />
              Tambah unit
            </Link>
          ) : null}
        </div>
      </div>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Ringkasan tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <section aria-labelledby="ringkasan-unit" className="space-y-3">
        <div className="flex items-end justify-between gap-3">
          <h2 id="ringkasan-unit" className="text-lg font-semibold tracking-tight">
            Ringkasan unit
          </h2>
          <p className="hidden text-xs text-muted-foreground sm:block">
            Data master unit saat ini
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <SummaryCard
            label="Total unit"
            value={summary?.units_total}
            loading={loading}
            icon={Truck}
            tone="sky"
          />
          <SummaryCard
            label="Unit aktif"
            value={summary?.units_active}
            loading={loading}
            icon={CheckCircle2}
            tone="emerald"
          />
          <SummaryCard
            label="Unit breakdown"
            value={summary?.units_breakdown}
            loading={loading}
            icon={TriangleAlert}
            tone="amber"
          />
          <SummaryCard
            label="Unit model"
            value={summary?.unit_models}
            loading={loading}
            icon={Boxes}
            tone="violet"
          />
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader className="border-b">
            <CardTitle>Kesehatan armada</CardTitle>
            <CardDescription>Perbandingan unit aktif dan breakdown.</CardDescription>
          </CardHeader>
          <CardContent className="pt-4">
            <FleetStatusChart
              unitsTotal={summary?.units_total ?? 0}
              unitsActive={summary?.units_active ?? 0}
              unitsBreakdown={summary?.units_breakdown ?? 0}
              loading={loading && !summary}
              failed={Boolean(error) && !summary}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="border-b">
            <CardTitle>Pembacaan HM/KM</CardTitle>
            <CardDescription>Reading manual 30 hari terakhir.</CardDescription>
          </CardHeader>
          <CardContent className="flex h-full flex-col justify-between gap-5 pt-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-4xl font-semibold tabular-nums">
                  {loading && !summary ? (
                    <Skeleton className="h-10 w-20" />
                  ) : (
                    formatOptionalNumber(summary?.readings_last_30_days)
                  )}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">reading tercatat</p>
              </div>
              <span className="grid size-12 shrink-0 place-items-center rounded-2xl bg-sky-500/10 text-sky-700 dark:text-sky-300">
                <Activity className="size-6" aria-hidden="true" />
              </span>
            </div>
            <p className="text-xs leading-5 text-muted-foreground">
              Reading lebih rendah dari nilai saat ini hanya dapat dikoreksi admin dengan alasan.
            </p>
            <Link
              href="/units"
              className="inline-flex items-center gap-1.5 text-sm font-medium text-primary underline-offset-4 hover:underline"
            >
              Lihat daftar unit
              <ArrowRight className="size-4" aria-hidden="true" />
            </Link>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="border-b">
            <CardTitle>Akses cepat</CardTitle>
            <CardDescription>Mulai dari master unit atau unit baru.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-2 pt-4">
            <QuickLink
              href="/units"
              icon={Truck}
              title="Daftar unit"
              description="HM/KM terakhir dan status unit."
            />
            <QuickLink
              href="/master"
              icon={Boxes}
              title="Master model"
              description="Unit model dan engine model."
            />
            {canWrite ? (
              <QuickLink
                href="/units/new"
                icon={Plus}
                title="Tambah unit"
                description="Daftarkan unit baru ke master."
              />
            ) : null}
          </CardContent>
        </Card>
      </div>

      <p className="text-xs leading-5 text-muted-foreground">
        Modul P2H, work order, PM, dan inventory belum termasuk Milestone 1.
      </p>
    </div>
  );
}

const tones = {
  sky: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
  emerald: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  amber: "bg-amber-500/10 text-amber-700 dark:text-amber-300",
  violet: "bg-violet-500/10 text-violet-700 dark:text-violet-300",
} as const;

type Tone = keyof typeof tones;

function SummaryCard({
  label,
  value,
  loading,
  icon: Icon,
  tone,
}: {
  label: string;
  value?: number;
  loading: boolean;
  icon: LucideIcon;
  tone: Tone;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">{label}</p>
          {loading && value === undefined ? (
            <Skeleton className="mt-3 h-8 w-20" />
          ) : (
            <p className="mt-2 font-mono text-3xl font-semibold tabular-nums">
              {formatOptionalNumber(value)}
            </p>
          )}
        </div>
        <span className={cn("grid size-10 shrink-0 place-items-center rounded-xl", tones[tone])}>
          <Icon className="size-5" aria-hidden="true" />
        </span>
      </div>
    </div>
  );
}

function QuickLink({
  href,
  icon: Icon,
  title,
  description,
}: {
  href: string;
  icon: LucideIcon;
  title: string;
  description: string;
}) {
  return (
    <Link
      href={href}
      className="group flex items-center gap-3 rounded-lg border border-border/70 p-3 transition-colors hover:border-border hover:bg-muted/50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-muted text-muted-foreground transition-colors group-hover:text-foreground">
        <Icon className="size-4" aria-hidden="true" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-medium">{title}</span>
        <span className="block truncate text-xs text-muted-foreground">{description}</span>
      </span>
      <ArrowRight
        className="size-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5"
        aria-hidden="true"
      />
    </Link>
  );
}

function formatOptionalNumber(value?: number): string {
  return typeof value === "number" ? formatNumber(value) : "-";
}
