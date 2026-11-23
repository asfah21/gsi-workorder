"use client";

import {
  Activity,
  ArrowRight,
  Box,
  CalendarClock,
  ChevronRight,
  CircleAlert,
  ClipboardList,
  PackageSearch,
  Truck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import {
  MaintenanceActivityChart,
  type ActivityDatum,
} from "@/components/dashboard-charts";
import { UnitStatusChart } from "@/components/unit-status-chart";
import { WorkOrderStatusBadge } from "@/components/workorder-badges";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardContent,
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
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
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
import {
  daysAgoWitaDate,
  formatNumber,
  formatWita,
  todayWitaDate,
} from "@/lib/time";
import type {
  DashboardMaintenance,
  DashboardPM,
  DataResponse,
  WorkOrderRecord,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const periods = [
  { value: "7", label: "7 hari" },
  { value: "30", label: "30 hari" },
  { value: "90", label: "90 hari" },
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

const pmStatusLabels: Record<string, string> = {
  overdue: "Overdue",
  due: "Due",
  "due soon": "Due soon",
  upcoming: "Upcoming",
  scheduled: "Scheduled",
  wo_created: "WO dibuat",
  completed: "Selesai",
  cancelled: "Batal",
  skipped: "Skip",
};

type SummarySnapshot = {
  total: number;
  active: number;
  breakdown: number;
  loading: boolean;
  failed: boolean;
  stockLow: number;
  stockCritical: number;
  waitingPart: number;
  pmOverdue: number;
  pmDue: number;
  pmDueSoon: number;
};

type OperationsData = {
  pm?: DashboardPM;
  maintenance?: DashboardMaintenance;
};

export function DashboardOperations({
  days,
  onDaysChange,
  refreshKey,
  summary,
  recent,
  recentLoading,
  recentError,
}: {
  days: string;
  onDaysChange: (days: string) => void;
  refreshKey: number;
  summary: SummarySnapshot;
  recent: WorkOrderRecord[];
  recentLoading: boolean;
  recentError: string;
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
      const [pm, maintenance] = await Promise.all([
        proxyGet<DataResponse<DashboardPM>>(`/dashboard/pm?${params}`),
        proxyGet<DataResponse<DashboardMaintenance>>(
          `/dashboard/maintenance?${params}`,
        ),
      ]);
      setData({ pm: pm.data, maintenance: maintenance.data });
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
  const maintenance = data.maintenance;
  const periodLabel =
    days === "7"
      ? "7 hari terakhir"
      : days === "90"
        ? "90 hari terakhir"
        : "30 hari terakhir";

  const activityData = buildActivityTrendData(maintenance?.activity.rows ?? []);
  const totalCreated = activityData.reduce((total, item) => total + item.created, 0);
  const totalCompleted = activityData.reduce(
    (total, item) => total + item.completed,
    0,
  );
  const completedPct =
    totalCreated > 0 ? Math.round((totalCompleted / totalCreated) * 100) : 0;
  const readyPct =
    summary.total > 0 ? Math.round((summary.active / summary.total) * 100) : 0;
  const pmRows = (pm?.due.rows ?? [])
    .slice()
    .sort((left, right) => pmRank(left) - pmRank(right))
    .slice(0, 4);

  return (
    <>
      {error ? (
        <Alert
          variant="destructive"
          className="col-span-12 border-destructive/30 bg-destructive/5"
        >
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Data operasional tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <Card className="col-span-12 xl:col-span-8">
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3 border-b">
          <CardTitle className="flex items-center gap-2">
            <Activity className="size-4 text-muted-foreground" aria-hidden="true" />
            Tren maintenance
          </CardTitle>
          <div className="flex items-center gap-2">
            <Badge
              variant="secondary"
              className="h-5 border-transparent bg-emerald-500/10 px-2 text-emerald-700 dark:text-emerald-300"
            >
              {completedPct}% selesai
            </Badge>
            <NativeSelect
              aria-label="Periode tren maintenance"
              value={days}
              onChange={(event) => onDaysChange(event.target.value)}
              className="h-8 w-fit text-sm"
            >
              {periods.map((period) => (
                <NativeSelectOption key={period.value} value={period.value}>
                  {period.label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-5 pt-5">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Work order dibuat</p>
              <p className="text-2xl font-semibold tracking-tight">
                {formatNumber(totalCreated)}
              </p>
              <p className="text-sm text-muted-foreground">{periodLabel}</p>
            </div>
            <div className="flex flex-wrap items-center gap-4 text-sm">
              <span className="flex items-center gap-1.5 text-muted-foreground">
                <span className="size-2.5 rounded-full bg-foreground" aria-hidden="true" />
                Dibuat
              </span>
              <span className="flex items-center gap-1.5 text-muted-foreground">
                <span
                  className="size-2.5 rounded-full bg-foreground/50"
                  aria-hidden="true"
                />
                Selesai
              </span>
            </div>
          </div>
          <MaintenanceActivityChart
            data={activityData}
            loading={loading && !maintenance}
            failed={Boolean(error) && !maintenance}
          />
        </CardContent>
      </Card>

      <Card className="col-span-12 xl:col-span-4">
        <CardHeader className="flex flex-row items-center gap-2 border-b">
          <Box className="size-4 text-muted-foreground" aria-hidden="true" />
          <CardTitle>Status unit</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-1 flex-col justify-between gap-5 pt-5">
          <UnitStatusChart
            unitsTotal={summary.total}
            unitsActive={summary.active}
            unitsBreakdown={summary.breakdown}
            loading={summary.loading}
            failed={summary.failed}
          />
        </CardContent>
      </Card>

      <StatCard
        className="col-span-12 md:col-span-4"
        label="Unit aktif"
        value={summary.active}
        hint={`${formatNumber(summary.total)} unit total`}
        badge={`${readyPct}%`}
        icon={Truck}
        href="/units"
      />
      <StatCard
        className="col-span-12 md:col-span-4"
        label="PM terlambat"
        value={summary.pmOverdue}
        hint={`${formatNumber(summary.pmDue)} due · ${formatNumber(
          summary.pmDueSoon,
        )} due soon`}
        badge={summary.pmOverdue > 0 ? "Perlu tindakan" : "Aman"}
        tone={summary.pmOverdue > 0 ? "danger" : "positive"}
        icon={CalendarClock}
        href="/pm"
      />
      <StatCard
        className="col-span-12 md:col-span-4"
        label="Stok kritis"
        value={summary.stockCritical}
        hint={`${formatNumber(summary.stockLow)} menipis · ${formatNumber(
          summary.waitingPart,
        )} unit tunggu part`}
        badge={summary.stockCritical > 0 ? "Kritis" : "Aman"}
        tone={summary.stockCritical > 0 ? "danger" : "positive"}
        icon={PackageSearch}
        href="/inventory"
      />

      <Card className="col-span-12 xl:col-span-6">
        <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
          <CardTitle className="flex items-center gap-2">
            <CalendarClock
              className="size-4 text-muted-foreground"
              aria-hidden="true"
            />
            PM mendesak
          </CardTitle>
          <Link
            href="/pm"
            className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            Lihat semua
          </Link>
        </CardHeader>
        <CardContent className="px-0">
          {loading && !pm ? (
            <div className="space-y-3 px-6 py-4" role="status" aria-label="Memuat PM">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-14 w-full" />
              ))}
            </div>
          ) : pmRows.length === 0 ? (
            <Empty className="border-0 py-8">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <CalendarClock aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Tidak ada PM mendesak</EmptyTitle>
                <EmptyDescription>
                  Jadwal PM yang mendekati jatuh tempo akan muncul di sini.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="flex flex-col">
              {pmRows.map((row, index) => {
                const status = pmBucket(row);
                return (
                  <div
                    key={`${String(row.schedule_id ?? index)}-${index}`}
                    className="flex items-center gap-3 border-b border-border px-6 py-4 last:border-0"
                  >
                    <div className="grid size-10 shrink-0 place-items-center rounded-md border border-border">
                      <CalendarClock className="size-4" aria-hidden="true" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {stringValue(row.unit_code) || "Unit"}
                      </p>
                      <p className="truncate text-sm text-muted-foreground">
                        {stringValue(row.plan_name) || "Rencana PM"}
                      </p>
                    </div>
                    <div className="flex shrink-0 flex-col items-end gap-1">
                      <Badge
                        variant="outline"
                        className={cn("h-5 px-2", pmStatusClass(status))}
                      >
                        {pmStatusLabels[status] || status || "-"}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {formatDue(row)}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="col-span-12 xl:col-span-6">
        <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
          <CardTitle className="flex items-center gap-2">
            <ClipboardList
              className="size-4 text-muted-foreground"
              aria-hidden="true"
            />
            Work order terbaru
          </CardTitle>
          <Link
            href="/workorders"
            className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            Lihat semua
          </Link>
        </CardHeader>
        <CardContent className="px-0">
          {recentLoading ? (
            <div className="space-y-3 px-6 py-4" role="status" aria-label="Memuat work order">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-12 w-full" />
              ))}
            </div>
          ) : recentError ? (
            <div className="px-6 py-6 text-sm text-muted-foreground">{recentError}</div>
          ) : recent.length === 0 ? (
            <Empty className="border-0 py-8">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ClipboardList aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada work order</EmptyTitle>
                <EmptyDescription>
                  Work order yang baru dibuat akan muncul di sini.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="pl-4">WO</TableHead>
                  <TableHead>Assignee</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Dibuat</TableHead>
                  <TableHead className="pr-4 text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recent.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="pl-4">
                      <Link
                        href={`/workorders?q=${encodeURIComponent(item.code)}`}
                        className="font-medium underline-offset-4 hover:underline"
                      >
                        {item.code}
                      </Link>
                      <span className="block max-w-[12rem] truncate text-sm text-muted-foreground">
                        {item.title} · {item.unit_code || "Tanpa unit"}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Avatar className="size-7">
                          <AvatarFallback className="bg-muted text-[10px] font-semibold text-muted-foreground">
                            {initials(item.assignee_name)}
                          </AvatarFallback>
                        </Avatar>
                        <span className="max-w-[7rem] truncate text-sm">
                          {item.assignee_name || "Belum di-assign"}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <WorkOrderStatusBadge status={item.status} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      {formatWita(item.created_at, { day: "numeric", month: "short" })}
                    </TableCell>
                    <TableCell className="pr-4 text-right">
                      <Link
                        href={`/workorders?q=${encodeURIComponent(item.code)}`}
                        aria-label={`Buka work order ${item.code}`}
                        className={buttonVariants({ variant: "ghost", size: "icon-xs" })}
                      >
                        <ChevronRight aria-hidden="true" />
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </>
  );
}

function StatCard({
  className,
  label,
  value,
  hint,
  badge,
  tone = "positive",
  icon: Icon,
  href,
}: {
  className?: string;
  label: string;
  value: number;
  hint: string;
  badge: string;
  tone?: "positive" | "warning" | "danger";
  icon: LucideIcon;
  href: string;
}) {
  const badgeClass =
    tone === "danger"
      ? "bg-destructive/10 text-destructive"
      : tone === "warning"
        ? "bg-amber-500/10 text-amber-700 dark:text-amber-300"
        : "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";

  return (
    <Card className={className}>
      <CardContent className="flex flex-col gap-4 p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">{label}</p>
            <div className="flex flex-wrap items-center gap-2">
              <p className="text-2xl font-semibold tracking-tight">
                {formatNumber(value)}
              </p>
              <Badge
                variant="secondary"
                className={cn("h-5 border-transparent px-2", badgeClass)}
              >
                {badge}
              </Badge>
            </div>
          </div>
          <div className="rounded-md border border-border p-2.5">
            <Icon className="size-4" aria-hidden="true" />
          </div>
        </div>
        <p className="text-sm text-muted-foreground">{hint}</p>
        <Link
          href={href}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "w-fit")}
        >
          Lihat detail
          <ArrowRight aria-hidden="true" />
        </Link>
      </CardContent>
    </Card>
  );
}

function buildActivityTrendData(
  rows: Array<Record<string, unknown>>,
): ActivityDatum[] {
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
      ...value,
    }));
}

function formatDue(row: Record<string, unknown>): string {
  const dueDate = typeof row.due_date === "string" ? row.due_date : "";
  const days = numberValue(row.days_to_due);
  if (dueDate) {
    return formatWita(dueDate, { day: "numeric", month: "short" });
  }
  if (days < 0) {
    return `${Math.abs(days)} hari lewat`;
  }
  if (days === 0) {
    return "Hari ini";
  }
  return `${days} hari lagi`;
}

function pmBucket(row: Record<string, unknown>): string {
  return stringValue(row.bucket) || stringValue(row.status);
}

function pmRank(row: Record<string, unknown>): number {
  const bucket = pmBucket(row);
  if (bucket === "overdue") return 0;
  if (bucket === "due") return 1;
  if (bucket === "due soon") return 2;
  if (bucket === "upcoming") return 3;
  if (bucket === "scheduled") return 4;
  return 5;
}

function pmStatusClass(status: string): string {
  if (status === "overdue") {
    return "border-destructive/30 text-destructive";
  }
  if (status === "due") {
    return "border-orange-500/30 text-orange-700 dark:text-orange-300";
  }
  if (status === "due soon") {
    return "border-amber-500/30 text-amber-700 dark:text-amber-300";
  }
  return "text-muted-foreground";
}

function initials(value?: string | null): string {
  const words = (value ?? "").trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return "WO";
  }
  return words
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? "")
    .join("");
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function numberValue(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function formatBucket(value: string): string {
  const [year, month, day] = value.split("-");
  const monthLabel = monthLabels[Number(month) - 1] ?? month;
  if (!day) {
    return `${monthLabel} ${year}`;
  }
  return `${Number(day)} ${monthLabel}`;
}
