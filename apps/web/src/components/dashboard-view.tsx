"use client";

import {
  Activity,
  ArrowRight,
  Boxes,
  CheckCircle2,
  CircleAlert,
  Gauge,
  Plus,
  RefreshCcw,
  TriangleAlert,
  Truck,
  Wrench,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { roleLabels, useSession } from "@/components/app-shell";
import { FleetStatusChart } from "@/components/fleet-status-chart";
import { UnitStatusBadge } from "@/components/status-badges";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DashboardSummary, DataResponse, ListResponse, Unit } from "@/lib/types";
import { formatNumber, formatWita } from "@/lib/time";

const PREVIEW_SIZE = 5;

export function DashboardView() {
  const { user } = useSession();
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [unitsPreview, setUnitsPreview] = useState<Unit[]>([]);
  const [loading, setLoading] = useState(true);
  const [previewLoading, setPreviewLoading] = useState(true);
  const [error, setError] = useState("");
  const [previewError, setPreviewError] = useState("");
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

  const loadPreview = useCallback(async () => {
    setPreviewLoading(true);
    try {
      const result = await proxyGet<ListResponse<Unit>>(
        `/units?page=1&limit=${PREVIEW_SIZE}&sort=code&order=asc`,
      );
      setUnitsPreview(result.data);
      setPreviewError("");
    } catch (loadError) {
      setPreviewError(
        loadError instanceof ApiError
          ? loadError.message
          : "Cuplikan unit tidak dapat dimuat.",
      );
    } finally {
      setPreviewLoading(false);
    }
  }, []);

  const refresh = useCallback(() => {
    void loadSummary();
    void loadPreview();
  }, [loadPreview, loadSummary]);

  useEffect(() => {
    void loadSummary();
    void loadPreview();
  }, [loadPreview, loadSummary]);

  const unitsTotal = summary?.units_total ?? 0;
  const unitsActive = summary?.units_active ?? 0;
  const breakdown = Math.max(0, summary?.units_breakdown ?? 0);
  const healthy = Math.max(0, unitsActive - breakdown);
  const inactive = Math.max(0, unitsTotal - unitsActive);
  const readiness = percentage(unitsActive, unitsTotal);

  return (
    <div className="flex flex-col gap-4">
      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Ringkasan tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-8">
          <CardContent className="flex flex-1 flex-col justify-between gap-6">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm text-muted-foreground">
                  {today || "Ringkasan hari ini"}
                </p>
                <h1 className="mt-1 text-xl font-semibold">
                  Selamat datang, {user.name || user.nik || "Pengguna"}
                </h1>
                <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
                  Ringkasan master unit dan pembacaan HM/KM terbaru. Gunakan menu
                  Unit untuk melihat detail dan riwayat reading.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={refresh}
                disabled={loading || previewLoading}
                aria-label="Muat ulang ringkasan"
                title="Muat ulang ringkasan"
              >
                <RefreshCcw
                  className={loading || previewLoading ? "animate-spin" : undefined}
                  aria-hidden="true"
                />
              </Button>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="secondary">{user.role ? roleLabels[user.role] : "-"}</Badge>
              <Link
                href="/units"
                className={buttonVariants({ variant: "default" })}
              >
                Lihat daftar unit
                <ArrowRight aria-hidden="true" />
              </Link>
              {canWrite ? (
                <Link
                  href="/units/new"
                  className={buttonVariants({ variant: "outline" })}
                >
                  <Plus aria-hidden="true" />
                  Tambah unit
                </Link>
              ) : null}
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-4">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="size-4 text-muted-foreground" aria-hidden="true" />
              Ringkasan kunci
            </CardTitle>
            <CardDescription>Data master saat ini.</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-3">
            <Insight
              icon={Truck}
              label="Total unit"
              value={summary?.units_total}
              loading={loading && !summary}
            />
            <Insight
              icon={CheckCircle2}
              label="Unit aktif"
              value={summary?.units_active}
              loading={loading && !summary}
            />
            <Insight
              icon={TriangleAlert}
              label="Breakdown"
              value={summary?.units_breakdown}
              loading={loading && !summary}
            />
            <Insight
              icon={Activity}
              label="Reading 30 hari"
              value={summary?.readings_last_30_days}
              loading={loading && !summary}
            />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-8">
          <CardHeader className="flex flex-row items-start justify-between gap-3 border-b">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Gauge className="size-4 text-muted-foreground" aria-hidden="true" />
                Kesiapan unit
              </CardTitle>
              <CardDescription>Komposisi status unit dari master.</CardDescription>
            </div>
            <Badge variant="outline" className="tabular-nums">
              {loading && !summary ? <Skeleton className="h-4 w-8" /> : `${readiness}%`}
            </Badge>
          </CardHeader>
          <CardContent>
            <div className="grid gap-6 sm:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
              <div className="flex flex-col justify-center rounded-xl border border-border bg-muted/30 p-4">
                <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Kesiapan
                </p>
                <p className="mt-2 text-3xl font-semibold tabular-nums">
                  {loading && !summary ? (
                    <Skeleton className="h-9 w-20" />
                  ) : (
                    `${readiness}%`
                  )}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {formatOptionalNumber(summary?.units_active)} dari{" "}
                  {formatOptionalNumber(summary?.units_total)} unit aktif
                </p>
              </div>

              <div className="flex flex-col justify-center gap-4">
                <StatusBar
                  label="Aktif sehat"
                  value={healthy}
                  total={unitsTotal}
                  color="var(--chart-2)"
                />
                <StatusBar
                  label="Breakdown"
                  value={breakdown}
                  total={unitsTotal}
                  color="var(--chart-1)"
                />
                <StatusBar
                  label="Nonaktif"
                  value={inactive}
                  total={unitsTotal}
                  color="var(--chart-4)"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-4">
          <CardHeader>
            <CardTitle>Distribusi status</CardTitle>
            <CardDescription>Perbandingan unit aktif dan breakdown.</CardDescription>
          </CardHeader>
          <CardContent>
            <FleetStatusChart
              unitsTotal={unitsTotal}
              unitsActive={unitsActive}
              unitsBreakdown={breakdown}
              loading={loading && !summary}
              failed={Boolean(error) && !summary}
            />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-12">
        <StatCard
          className="lg:col-span-4"
          label="Unit model"
          value={summary?.unit_models}
          loading={loading && !summary}
          icon={Boxes}
          note="Referensi model unit"
          href="/master"
        />
        <StatCard
          className="lg:col-span-4"
          label="Engine model"
          value={summary?.engine_models}
          loading={loading && !summary}
          icon={Wrench}
          note="Referensi engine"
          href="/master"
        />
        <StatCard
          className="lg:col-span-4"
          label="Reading 30 hari"
          value={summary?.readings_last_30_days}
          loading={loading && !summary}
          icon={Activity}
          note="HM/KM tercatat"
          href="/units"
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-8">
          <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
            <div>
              <CardTitle>Cuplikan unit</CardTitle>
              <CardDescription>Unit pertama dari master unit.</CardDescription>
            </div>
            <Link href="/units" className={buttonVariants({ variant: "outline", size: "sm" })}>
              Lihat semua
              <ArrowRight aria-hidden="true" />
            </Link>
          </CardHeader>
          <CardContent className="px-0">
            {previewLoading ? (
              <div className="space-y-3 px-6 py-4" role="status" aria-label="Memuat unit">
                {Array.from({ length: PREVIEW_SIZE }).map((_, index) => (
                  <Skeleton key={index} className="h-12 w-full" />
                ))}
              </div>
            ) : previewError ? (
              <div className="px-6 py-4 text-sm text-muted-foreground">
                {previewError}
              </div>
            ) : unitsPreview.length === 0 ? (
              <Empty className="border-0 py-10">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Truck aria-hidden="true" />
                  </EmptyMedia>
                  <EmptyTitle>Belum ada unit</EmptyTitle>
                  <EmptyDescription>
                    Tambahkan unit pertama untuk mulai mencatat HM/KM.
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Unit</TableHead>
                      <TableHead>Model</TableHead>
                      <TableHead>Lokasi</TableHead>
                      <TableHead className="text-right">HM</TableHead>
                      <TableHead>Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {unitsPreview.map((unit) => (
                      <TableRow key={unit.id}>
                        <TableCell>
                          <Link
                            href={`/units/${unit.id}`}
                            className="font-medium underline-offset-4 hover:underline"
                          >
                            {unit.code}
                          </Link>
                          <span className="block text-xs text-muted-foreground">
                            {unit.unit_type || "Tanpa tipe"}
                          </span>
                        </TableCell>
                        <TableCell>{modelLabel(unit)}</TableCell>
                        <TableCell>{unit.site_location || "-"}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          {unit.hm_updated_at ? formatNumber(unit.current_hm) : "-"}
                        </TableCell>
                        <TableCell>
                          <UnitStatusBadge
                            active={unit.is_active}
                            breakdown={unit.is_breakdown}
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-4">
          <CardHeader>
            <CardTitle>Akses cepat</CardTitle>
            <CardDescription>Mulai dari master unit atau unit baru.</CardDescription>
          </CardHeader>
          <CardContent className="gap-2">
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

function Insight({
  icon: Icon,
  label,
  value,
  loading,
}: {
  icon: LucideIcon;
  label: string;
  value?: number;
  loading: boolean;
}) {
  return (
    <div className="rounded-lg border border-border bg-muted/20 p-3">
      <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        <Icon className="size-3.5" aria-hidden="true" />
        {label}
      </div>
      {loading ? (
        <Skeleton className="mt-2 h-6 w-16" />
      ) : (
        <p className="mt-2 text-2xl font-semibold tabular-nums">
          {formatOptionalNumber(value)}
        </p>
      )}
    </div>
  );
}

function StatusBar({
  label,
  value,
  total,
  color,
}: {
  label: string;
  value: number;
  total: number;
  color: string;
}) {
  const percent = Math.min(100, percentage(value, total));
  return (
    <div>
      <div className="flex items-center justify-between gap-3 text-sm">
        <span className="flex items-center gap-2 text-muted-foreground">
          <span
            className="size-2.5 rounded-full"
            style={{ backgroundColor: color }}
            aria-hidden="true"
          />
          {label}
        </span>
        <span className="font-medium tabular-nums">{formatNumber(value)}</span>
      </div>
      <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted" aria-hidden="true">
        <div
          className="h-full rounded-full transition-[width]"
          style={{ width: `${percent}%`, backgroundColor: color }}
        />
      </div>
    </div>
  );
}

function StatCard({
  className,
  label,
  value,
  loading,
  icon: Icon,
  note,
  href,
}: {
  className?: string;
  label: string;
  value?: number;
  loading: boolean;
  icon: LucideIcon;
  note: string;
  href: string;
}) {
  return (
    <Card className={className}>
      <CardContent className="flex flex-1 flex-col gap-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-sm text-muted-foreground">{label}</p>
            {loading ? (
              <Skeleton className="mt-2 h-8 w-16" />
            ) : (
              <p className="mt-2 text-2xl font-semibold tabular-nums">
                {formatOptionalNumber(value)}
              </p>
            )}
          </div>
          <span className="grid size-9 shrink-0 place-items-center rounded-md border border-border bg-muted/40 text-muted-foreground">
            <Icon className="size-4" aria-hidden="true" />
          </span>
        </div>
        <div className="mt-auto flex items-center justify-between gap-2">
          <span className="text-xs text-muted-foreground">{note}</span>
          <Link
            href={href}
            className={buttonVariants({ variant: "outline", size: "sm" })}
          >
            Lihat
            <ArrowRight aria-hidden="true" />
          </Link>
        </div>
      </CardContent>
    </Card>
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
      className="group flex items-center gap-3 rounded-md border border-border/70 p-3 transition-colors hover:border-border hover:bg-muted/50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      <span className="grid size-9 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground transition-colors group-hover:text-foreground">
        <Icon className="size-4" aria-hidden="true" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-medium">{title}</span>
        <span className="block truncate text-xs text-muted-foreground">
          {description}
        </span>
      </span>
      <ArrowRight
        className="size-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5"
        aria-hidden="true"
      />
    </Link>
  );
}

function modelLabel(unit: Unit): string {
  if (unit.unit_model) {
    return `${unit.unit_model.manufacturer} ${unit.unit_model.model}`;
  }
  return unit.merk || "-";
}

function percentage(value: number, total: number): number {
  if (total <= 0) {
    return 0;
  }
  return Math.round((value / total) * 100);
}

function formatOptionalNumber(value?: number): string {
  return typeof value === "number" ? formatNumber(value) : "-";
}
