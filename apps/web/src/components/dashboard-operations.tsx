"use client";

import { CircleAlert } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { BucketBarChart, type BucketDatum } from "@/components/dashboard-charts";
import { FleetStatusChart } from "@/components/fleet-status-chart";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ApiError, proxyGet } from "@/lib/api-client";
import {
  daysAgoWitaDate,
  formatDurationMinutes,
  formatNumber,
  todayWitaDate,
} from "@/lib/time";
import type {
  DashboardMaintenance,
  DashboardMaterial,
  DashboardPM,
  DataResponse,
} from "@/lib/types";

const periods = [
  { value: "7", label: "7H" },
  { value: "30", label: "30H" },
  { value: "90", label: "90H" },
] as const;

const monthLabels = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "Mei",
  "Jun",
  "Jul",
  "Agu",
  "Sep",
  "Okt",
  "Nov",
  "Des",
];

type SummarySnapshot = {
  total: number;
  active: number;
  breakdown: number;
  loading: boolean;
  failed: boolean;
  stockLow: number;
  stockCritical: number;
  waitingPart: number;
  downtime30d: number;
  mttr30d: number;
  pmOverdue: number;
  pmDue: number;
  pmDueSoon: number;
  woOpen: number;
  woInProgress: number;
};

type OperationsData = {
  pm?: DashboardPM;
  material?: DashboardMaterial;
  maintenance?: DashboardMaintenance;
};

export function DashboardOperations({
  days,
  onDaysChange,
  refreshKey,
  summary,
}: {
  days: string;
  onDaysChange: (days: string) => void;
  refreshKey: number;
  summary: SummarySnapshot;
}) {
  const [data, setData] = useState<OperationsData>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({
      from: daysAgoWitaDate(Number(days)),
      to: todayWitaDate(),
      bucket: days === "7" ? "day" : "week",
      limit: "100",
    });
    try {
      const [pm, material, maintenance] = await Promise.all([
        proxyGet<DataResponse<DashboardPM>>(`/dashboard/pm?${params}`),
        proxyGet<DataResponse<DashboardMaterial>>(`/dashboard/material?${params}`),
        proxyGet<DataResponse<DashboardMaintenance>>(`/dashboard/maintenance?${params}`),
      ]);
      setData({ pm: pm.data, material: material.data, maintenance: maintenance.data });
      setError("");
    } catch (loadError) {
      setError(
        loadError instanceof ApiError && loadError.status === 403
          ? "Anda tidak memiliki akses ke data dashboard operasional."
          : "Data operasional tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  const pm = data.pm;
  const material = data.material;
  const maintenance = data.maintenance;
  const maintenanceSummary = maintenance?.summary;
  const periodLabel =
    days === "7" ? "7 hari terakhir" : days === "90" ? "90 hari terakhir" : "30 hari terakhir";

  const downtimeTrendData = buildDowntimeTrendData(maintenance?.downtime.rows ?? []);
  const downtimeClassData = buildDowntimeData(maintenanceSummary);
  const workOrderData = buildWorkOrderData(maintenanceSummary, summary);
  const pmData = buildPmData(pm?.summary, summary);
  const agingData = buildAgingData(material?.aging.rows ?? []);

  const totalDowntime = maintenanceSummary
    ? numberValue(maintenanceSummary.total_minutes)
    : summary.downtime30d;
  const mttr = maintenanceSummary
    ? numberValue(maintenanceSummary.mttr_minutes)
    : summary.mttr30d;
  const waitingUnits = maintenanceSummary
    ? numberValue(maintenanceSummary.units_waiting)
    : summary.waitingPart;
  const waitingMinutes = numberValue(maintenanceSummary?.waiting_minutes_total);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-sm font-medium">Operasional</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Downtime, status work order, dan kesiapan armada {periodLabel}.
          </p>
        </div>
        <div
          role="group"
          aria-label="Periode dashboard operasional"
          className="flex items-center gap-1 rounded-lg border border-border p-1"
        >
          {periods.map((period) => (
            <Button
              key={period.value}
              type="button"
              size="sm"
              variant={days === period.value ? "secondary" : "ghost"}
              aria-pressed={days === period.value}
              onClick={() => onDaysChange(period.value)}
            >
              {period.label}
            </Button>
          ))}
        </div>
      </div>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Data operasional tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-8">
          <CardHeader className="flex flex-row items-start justify-between gap-4 border-b">
            <div>
              <CardTitle>Downtime &amp; MTTR</CardTitle>
              <CardDescription>
                Durasi downtime dan rata-rata pemulihan per bucket pada {periodLabel}.
              </CardDescription>
            </div>
            <div className="flex shrink-0 gap-6">
              <MiniStat label="Total downtime" value={formatDurationMinutes(totalDowntime)} />
              <MiniStat label="MTTR" value={formatDurationMinutes(mttr)} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="mb-3 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
              {downtimeClassData.map((item) => (
                <span key={item.label} className="flex items-center gap-1.5">
                  <span
                    className="size-2.5 rounded-full"
                    style={{ backgroundColor: item.fill }}
                    aria-hidden="true"
                  />
                  {item.label}: {formatDurationMinutes(item.value)}
                </span>
              ))}
            </div>
            <BucketBarChart
              data={downtimeTrendData}
              loading={loading && !maintenance}
              failed={Boolean(error) && !maintenance}
              valueLabel="Downtime"
              formatValue={formatDurationMinutes}
              emptyTitle="Tidak ada downtime"
              emptyDescription="Downtime pada periode ini akan muncul di sini."
            />
          </CardContent>
        </Card>

        <Card className="lg:col-span-4">
          <CardHeader className="flex flex-row items-start justify-between gap-3 border-b">
            <div>
              <CardTitle>Status work order</CardTitle>
              <CardDescription>
                Komposisi work order aktif dan selesai pada {periodLabel}.
              </CardDescription>
            </div>
            <Link
              href="/workorders"
              className={buttonVariants({ variant: "outline", size: "sm" })}
            >
              Lihat
            </Link>
          </CardHeader>
          <CardContent>
            <BucketBarChart
              data={workOrderData}
              loading={loading && !maintenance}
              failed={Boolean(error) && !maintenance}
              valueLabel="Work order"
              emptyTitle="Belum ada work order"
              emptyDescription="Work order pada periode ini akan muncul di sini."
            />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader className="pb-0">
            <CardTitle>Kesiapan stok</CardTitle>
            <CardDescription>Stok menipis dan umur shortage material.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <MiniStat label="Menipis" value={formatNumber(summary.stockLow)} />
              <MiniStat label="Kritis" value={formatNumber(summary.stockCritical)} />
            </div>
            <BucketBarChart
              data={agingData}
              loading={loading && !material}
              failed={Boolean(error) && !material}
              valueLabel="Item"
              emptyTitle="Tidak ada shortage"
              emptyDescription="Tidak ada item shortage pada periode ini."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-3 pb-0">
            <div>
              <CardTitle>Armada aktif</CardTitle>
              <CardDescription>Komposisi unit dari master unit.</CardDescription>
            </div>
            <Link
              href="/units"
              className={buttonVariants({ variant: "outline", size: "sm" })}
            >
              Lihat
            </Link>
          </CardHeader>
          <CardContent>
            <FleetStatusChart
              unitsTotal={summary.total}
              unitsActive={summary.active}
              unitsBreakdown={summary.breakdown}
              loading={summary.loading}
              failed={summary.failed}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-0">
            <CardTitle>Kondisi armada</CardTitle>
            <CardDescription>Unit breakdown dan jadwal PM mendesak.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <MiniStat label="Breakdown" value={formatNumber(summary.breakdown)} />
              <MiniStat label="PM terlambat" value={formatNumber(summary.pmOverdue)} />
            </div>
            <BucketBarChart
              data={pmData}
              loading={loading && !pm}
              failed={Boolean(error) && !pm}
              valueLabel="Jadwal"
              emptyTitle="Tidak ada jadwal PM"
              emptyDescription="Jadwal PM pada periode ini akan muncul di sini."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-0">
            <CardTitle>Penunggu part</CardTitle>
            <CardDescription>Work order yang menunggu ketersediaan part.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <MiniStat label="Unit" value={formatNumber(waitingUnits)} />
              <MiniStat label="Durasi" value={formatDurationMinutes(waitingMinutes)} />
            </div>
            <p className="text-sm text-muted-foreground">
              Data dihitung dari work order berstatus waiting_part pada periode ini.
            </p>
            <Link
              href="/workorders"
              className={buttonVariants({ variant: "outline", size: "sm" })}
            >
              Lihat work order
            </Link>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="text-2xl font-semibold">{value}</p>
    </div>
  );
}

function buildDowntimeTrendData(rows: Array<Record<string, unknown>>): BucketDatum[] {
  const buckets = new Map<string, number>();
  for (const row of rows) {
    const bucket = typeof row.bucket === "string" ? row.bucket : "";
    if (!bucket) {
      continue;
    }
    buckets.set(bucket, (buckets.get(bucket) ?? 0) + numberValue(row.downtime_minutes));
  }
  return [...buckets.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .slice(-12)
    .map(([bucket, value]) => ({
      label: formatBucket(bucket),
      value,
      fill: "var(--chart-1)",
    }));
}

function buildWorkOrderData(
  data: Record<string, number> | undefined,
  fallback: SummarySnapshot,
): BucketDatum[] {
  const value = (key: string, fallbackValue = 0) =>
    data ? numberValue(data[key]) : fallbackValue;
  return [
    { label: "Open", value: value("open", fallback.woOpen), fill: "var(--chart-1)" },
    {
      label: "Dalam proses",
      value: value("in_progress", fallback.woInProgress),
      fill: "var(--chart-2)",
    },
    {
      label: "Tunggu part",
      value: value("waiting_part", fallback.waitingPart),
      fill: "var(--chart-3)",
    },
    { label: "Backlog", value: value("backlog"), fill: "var(--chart-4)" },
    { label: "Selesai", value: value("done"), fill: "var(--chart-5)" },
  ];
}

function buildPmData(
  data: Record<string, number> | undefined,
  fallback: SummarySnapshot,
): BucketDatum[] {
  const value = (key: string, fallbackValue = 0) =>
    data ? numberValue(data[key]) : fallbackValue;
  const dueSoon = data
    ? numberValue(data["due soon"] ?? data.due_soon)
    : fallback.pmDueSoon;
  return [
    {
      label: "Overdue",
      value: value("overdue", fallback.pmOverdue),
      fill: "var(--chart-1)",
    },
    { label: "Due", value: value("due", fallback.pmDue), fill: "var(--chart-3)" },
    { label: "Due soon", value: dueSoon, fill: "var(--chart-4)" },
    { label: "Upcoming", value: value("upcoming"), fill: "var(--chart-2)" },
    { label: "Scheduled", value: value("scheduled"), fill: "var(--chart-5)" },
  ];
}

function buildAgingData(rows: Array<Record<string, unknown>>): BucketDatum[] {
  return rows.map((row) => ({
    label: String(row.age_bucket ?? "-"),
    value: numberValue(row.items),
    fill: "var(--chart-3)",
  }));
}

function buildDowntimeData(summary?: Record<string, number>): BucketDatum[] {
  return [
    {
      label: "Planned",
      value: numberValue(summary?.planned_minutes),
      fill: "var(--chart-2)",
    },
    {
      label: "Unplanned",
      value: numberValue(summary?.unplanned_minutes),
      fill: "var(--chart-1)",
    },
  ];
}

function formatBucket(value: string): string {
  const [year, month, day] = value.split("-");
  const monthLabel = monthLabels[Number(month) - 1] ?? month;
  if (!day) {
    return `${monthLabel} ${year}`;
  }
  return `${Number(day)} ${monthLabel}`;
}

function numberValue(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}
