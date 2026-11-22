"use client";

import {
  ArrowRight,
  Boxes,
  CalendarClock,
  CircleAlert,
  FileText,
  PackageSearch,
  Plus,
  RefreshCcw,
  TriangleAlert,
  Truck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { DashboardOperations } from "@/components/dashboard-operations";
import { KpiCard } from "@/components/kpi-card";
import { PageHeader } from "@/components/layout/page-header";
import { UnitStatusBadge } from "@/components/status-badges";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  const [days, setDays] = useState("30");
  const [refreshKey, setRefreshKey] = useState(0);

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
    setRefreshKey((current) => current + 1);
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

  const unitHint = summary
    ? `${formatNumber(summary.units_total)} unit · ${formatNumber(breakdown)} breakdown`
    : error
      ? "Data tidak tersedia"
      : "Memuat data unit";
  const pmHint = summary
    ? `${formatNumber(summary.pm_due ?? 0)} due · ${formatNumber(summary.pm_due_soon ?? 0)} due soon`
    : error
      ? "Data tidak tersedia"
      : "Memuat jadwal PM";
  const materialHint = summary
    ? `${formatNumber(summary.waiting_part_units ?? 0)} unit menunggu part`
    : error
      ? "Data tidak tersedia"
      : "Memuat kesiapan material";
  const stockHint = summary
    ? `${formatNumber(summary.stock_low ?? 0)} menipis · ${formatNumber(summary.stock_critical ?? 0)} kritis`
    : error
      ? "Data tidak tersedia"
      : "Memuat status stok";

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Dashboard"
        description={
          <>
            {user.name || user.nik ? `Selamat datang, ${user.name || user.nik}. ` : ""}
            {today ? `${today}. ` : ""}
            Ringkasan armada, maintenance, dan operasional gudang.
          </>
        }
        actions={
          <>
            <Link href="/reports" className={buttonVariants({ variant: "outline" })}>
              <FileText aria-hidden="true" />
              Laporan
            </Link>
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={refresh}
              disabled={loading || previewLoading}
              aria-label="Muat ulang dashboard"
              title="Muat ulang dashboard"
            >
              <RefreshCcw
                className={loading || previewLoading ? "animate-spin" : undefined}
                aria-hidden="true"
              />
            </Button>
          </>
        }
      />

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Ringkasan tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Unit aktif"
          value={summary?.units_active}
          loading={loading && !summary}
          icon={Truck}
          href="/units"
          hint={unitHint}
        />
        <KpiCard
          label="PM terlambat"
          value={summary?.pm_overdue}
          loading={loading && !summary}
          icon={CalendarClock}
          href="/pm"
          hint={pmHint}
        />
        <KpiCard
          label="Material shortage"
          value={summary?.material_shortage}
          loading={loading && !summary}
          icon={PackageSearch}
          href="/workorders"
          hint={materialHint}
        />
        <KpiCard
          label="Stok kritis"
          value={summary?.stock_critical}
          loading={loading && !summary}
          icon={TriangleAlert}
          href="/inventory"
          hint={stockHint}
        />
      </div>

      <DashboardOperations
        days={days}
        onDaysChange={setDays}
        refreshKey={refreshKey}
        fleet={{
          total: unitsTotal,
          active: unitsActive,
          breakdown,
          loading: loading && !summary,
          failed: Boolean(error) && !summary,
        }}
      />

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
                        <TableCell className="text-right">
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

      <p className="text-xs text-muted-foreground">
        Data operasional dihitung dari transaksi aktual; seluruh tanggal dan bucket
        memakai batas hari WITA.
      </p>
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
