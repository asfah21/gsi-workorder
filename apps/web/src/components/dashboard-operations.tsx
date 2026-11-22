"use client";

import { CircleAlert } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { FleetStatusChart } from "@/components/fleet-status-chart";
import {
  BucketBarChart,
  type BucketDatum,
  MaintenanceActivityChart,
  type ActivityDatum,
} from "@/components/dashboard-charts";
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

type FleetSnapshot = {
  total: number;
  active: number;
  breakdown: number;
  loading: boolean;
  failed: boolean;
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
  fleet,
}: {
  days: string;
  onDaysChange: (days: string) => void;
  refreshKey: number;
  fleet: FleetSnapshot;
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
  const periodLabel =
    days === "7" ? "7 hari terakhir" : days === "90" ? "90 hari terakhir" : "30 hari terakhir";

  const activityData = buildActivityData(maintenance?.activity.rows ?? []);
  const pmData = buildPmData(pm?.summary);
  const agingData = buildAgingData(material?.aging.rows ?? []);
  const downtimeData = buildDowntimeData(maintenance?.summary);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-base font-medium">Operasional</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Aktivitas maintenance, PM, material, dan downtime {periodLabel}.
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
          <CardHeader className="flex flex-row items-start justify-between gap-3">
            <div>
              <CardTitle>Aktivitas maintenance</CardTitle>
              <CardDescription>
                Work order dibuat dan selesai per bucket pada {periodLabel}.
              </CardDescription>
            </div>
            <div className="hidden items-center gap-4 text-xs text-muted-foreground sm:flex">
              <span className="flex items-center gap-1.5">
                <span
                  className="size-2.5 rounded-full"
                  style={{ backgroundColor: "var(--chart-1)" }}
                  aria-hidden="true"
                />
                WO dibuat
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="size-2.5 rounded-full"
                  style={{ backgroundColor: "var(--chart-2)" }}
                  aria-hidden="true"
                />
                WO selesai
              </span>
            </div>
          </CardHeader>
          <CardContent>
            <MaintenanceActivityChart
              data={activityData}
              loading={loading && !maintenance}
              failed={Boolean(error) && !maintenance}
            />
          </CardContent>
        </Card>

        <Card className="lg:col-span-4">
          <CardHeader className="flex flex-row items-start justify-between gap-3">
            <div>
              <CardTitle>Status unit</CardTitle>
              <CardDescription>Komposisi status unit saat ini.</CardDescription>
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
              unitsTotal={fleet.total}
              unitsActive={fleet.active}
              unitsBreakdown={fleet.breakdown}
              loading={fleet.loading}
              failed={fleet.failed}
            />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <ChartCard
          title="Jadwal PM"
          description="Bucket jatuh tempo dalam periode."
        >
          <BucketBarChart
            data={pmData}
            loading={loading && !pm}
            failed={Boolean(error) && !pm}
            valueLabel="Jadwal"
            emptyTitle="Tidak ada jadwal PM"
            emptyDescription="Jadwal PM pada periode ini akan muncul di sini."
          />
        </ChartCard>

        <ChartCard
          title="Umur shortage"
          description="Distribusi item shortage berdasarkan umur."
        >
          <BucketBarChart
            data={agingData}
            loading={loading && !material}
            failed={Boolean(error) && !material}
            valueLabel="Item"
            emptyTitle="Tidak ada shortage"
            emptyDescription="Tidak ada item shortage pada periode ini."
          />
        </ChartCard>

        <ChartCard
          title="Downtime"
          description="Perbandingan downtime planned dan unplanned."
        >
          <BucketBarChart
            data={downtimeData}
            loading={loading && !maintenance}
            failed={Boolean(error) && !maintenance}
            valueLabel="Durasi"
            formatValue={formatDurationMinutes}
            emptyTitle="Tidak ada downtime"
            emptyDescription="Downtime pada periode ini akan muncul di sini."
          />
        </ChartCard>
      </div>
    </div>
  );
}

function ChartCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

function buildActivityData(rows: Array<Record<string, unknown>>): ActivityDatum[] {
  const buckets = new Map<string, { created: number; completed: number }>();
  for (const row of rows) {
    const bucket = typeof row.bucket === "string" ? row.bucket : "";
    if (!bucket) {
      continue;
    }
    const current = buckets.get(bucket) ?? { created: 0, completed: 0 };
    current.created += numberValue(row.wo_count);
    current.completed += numberValue(row.completed);
    buckets.set(bucket, current);
  }
  return [...buckets.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .slice(-12)
    .map(([bucket, value]) => ({
      label: formatBucket(bucket),
      created: value.created,
      completed: value.completed,
    }));
}

function buildPmData(summary?: Record<string, number>): BucketDatum[] {
  const value = (key: string) => numberValue(summary?.[key]);
  const dueSoon = summary?.["due soon"] ?? summary?.due_soon ?? 0;
  return [
    { label: "Overdue", value: value("overdue"), fill: "var(--chart-1)" },
    { label: "Due", value: value("due"), fill: "var(--chart-3)" },
    { label: "Due soon", value: numberValue(dueSoon), fill: "var(--chart-4)" },
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
