"use client";

import {
  ArrowRight,
  CircleAlert,
  Lightbulb,
  Moon,
  RefreshCcw,
  Sun,
} from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { DashboardOperations } from "@/components/dashboard-operations";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet } from "@/lib/api-client";
import type {
  DashboardSummary,
  DataResponse,
  ListResponse,
  WorkOrderRecord,
} from "@/lib/types";
import { formatNumber, formatWita } from "@/lib/time";
import { cn } from "@/lib/utils";

const RECENT_SIZE = 5;

export function DashboardView() {
  const { user } = useSession();
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [recent, setRecent] = useState<WorkOrderRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [recentLoading, setRecentLoading] = useState(true);
  const [error, setError] = useState("");
  const [recentError, setRecentError] = useState("");
  const [today, setToday] = useState("");
  const [greeting, setGreeting] = useState("");
  const [days, setDays] = useState("30");
  const [refreshKey, setRefreshKey] = useState(0);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    const now = new Date();
    const hour = Number(formatWita(now, { hour: "numeric", hour12: false }));
    setGreeting(
      hour < 11
        ? "Selamat pagi"
        : hour < 15
          ? "Selamat siang"
          : hour < 19
            ? "Selamat sore"
            : "Selamat malam",
    );
    setToday(
      formatWita(now, {
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

  const loadRecent = useCallback(async () => {
    setRecentLoading(true);
    try {
      const result = await proxyGet<ListResponse<WorkOrderRecord>>(
        `/workorders?page=1&limit=${RECENT_SIZE}&sort=created_at&order=desc`,
      );
      setRecent(result.data);
      setRecentError("");
    } catch (loadError) {
      setRecentError(
        loadError instanceof ApiError
          ? loadError.message
          : "Work order terbaru tidak dapat dimuat.",
      );
    } finally {
      setRecentLoading(false);
    }
  }, []);

  const refresh = useCallback(async () => {
    setRefreshing(true);
    setRefreshKey((current) => current + 1);
    await Promise.all([loadSummary(), loadRecent()]);
    setRefreshing(false);
  }, [loadRecent, loadSummary]);

  useEffect(() => {
    void loadSummary();
    void loadRecent();
  }, [loadRecent, loadSummary]);

  return (
    <div className="flex flex-col gap-4">
      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Ringkasan tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid grid-cols-12 gap-4">
        <DashboardHero
          className="col-span-12 xl:col-span-8"
          user={user}
          greeting={greeting}
          today={today}
          summary={summary}
          loading={loading}
          refreshing={refreshing}
          onRefresh={() => void refresh()}
        />
        <DashboardInsights
          className="col-span-12 xl:col-span-4"
          summary={summary}
          loading={loading}
        />
        <DashboardOperations
          days={days}
          onDaysChange={setDays}
          refreshKey={refreshKey}
          summary={{
            total: summary?.units_total ?? 0,
            active: summary?.units_active ?? 0,
            breakdown: Math.max(0, summary?.units_breakdown ?? 0),
            loading: loading && !summary,
            failed: Boolean(error) && !summary,
            stockLow: summary?.stock_low ?? 0,
            stockCritical: summary?.stock_critical ?? 0,
            waitingPart: summary?.waiting_part_units ?? 0,
            pmOverdue: summary?.pm_overdue ?? 0,
            pmDue: summary?.pm_due ?? 0,
            pmDueSoon: summary?.pm_due_soon ?? 0,
          }}
          recent={recent}
          recentLoading={recentLoading}
          recentError={recentError}
        />
      </div>

      <p className="text-xs text-muted-foreground">
        Data operasional dihitung dari transaksi aktual; seluruh tanggal dan bucket
        memakai batas hari WITA.
      </p>
    </div>
  );
}

function DashboardHero({
  className,
  user,
  greeting,
  today,
  summary,
  loading,
  refreshing,
  onRefresh,
}: {
  className?: string;
  user: { name?: string; nik?: string };
  greeting: string;
  today: string;
  summary: DashboardSummary | null;
  loading: boolean;
  refreshing: boolean;
  onRefresh: () => void;
}) {
  const name = user.name || user.nik || "Pengguna";
  const displayGreeting = greeting || "Selamat datang";
  const isNight = displayGreeting === "Selamat malam";
  const summaryLine = summary
    ? `${formatNumber(summary.units_active)} unit aktif · ${formatNumber(
        summary.wo_open ?? 0,
      )} WO open · ${formatNumber(summary.pm_overdue ?? 0)} PM terlambat`
    : loading
      ? "Memuat ringkasan operasional..."
      : "Ringkasan operasional belum tersedia.";

  return (
    <Card className={cn("relative overflow-hidden border-border bg-card", className)}>
      <CardContent className="relative z-10 flex min-h-[232px] flex-col items-start justify-center gap-5 p-6 md:p-8">
        <div className="max-w-xl space-y-3 lg:max-w-[52%]">
          <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            {displayGreeting}, {name}
            {isNight ? (
              <Moon className="size-6 text-sky-300" aria-hidden="true" />
            ) : (
              <Sun className="size-6 text-amber-500" aria-hidden="true" />
            )}
          </h1>
          <p className="text-sm text-muted-foreground">
            {today ? `${today}. ` : ""}
            Ringkasan armada, maintenance, dan operasional gudang.
          </p>
          {summary ? (
            <div className="flex flex-wrap gap-2 pt-1">
              <Badge variant="outline" className="h-6 px-2.5">
                {summaryLine}
              </Badge>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{summaryLine}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Link href="/reports" className={buttonVariants({ variant: "default" })}>
            Lihat laporan
            <ArrowRight aria-hidden="true" />
          </Link>
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={onRefresh}
            disabled={refreshing}
            aria-label="Muat ulang dashboard"
            title="Muat ulang dashboard"
          >
            <RefreshCcw
              className={refreshing ? "animate-spin" : undefined}
              aria-hidden="true"
            />
          </Button>
        </div>
      </CardContent>

      <Image
        src="/images/backgrounds/analytic-user-dark.svg"
        alt=""
        width={420}
        height={240}
        priority
        className="pointer-events-none absolute -bottom-2 right-4 hidden w-[300px] dark:block lg:w-[380px]"
      />
      <Image
        src="/images/backgrounds/analytic-user.svg"
        alt=""
        width={420}
        height={240}
        priority
        className="pointer-events-none absolute -bottom-2 right-4 hidden w-[300px] dark:hidden lg:block lg:w-[380px]"
      />
    </Card>
  );
}

function DashboardInsights({
  className,
  summary,
  loading,
}: {
  className?: string;
  summary: DashboardSummary | null;
  loading: boolean;
}) {
  const total = summary?.units_total ?? 0;
  const active = summary?.units_active ?? 0;
  const breakdown = Math.max(0, summary?.units_breakdown ?? 0);
  const inactive = Math.max(0, total - active);
  const readyPct = total > 0 ? Math.round((active / total) * 100) : 0;
  const segments = 40;
  const activeSegments = total > 0 ? Math.round((active / total) * segments) : 0;
  const breakdownSegments =
    total > 0
      ? Math.min(segments - activeSegments, Math.round((breakdown / total) * segments))
      : 0;

  return (
    <Card className={cn("flex flex-col", className)}>
      <CardHeader className="flex flex-row items-center gap-2 border-b">
        <Lightbulb className="size-4 text-muted-foreground" aria-hidden="true" />
        <CardTitle>Ringkasan armada</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-5 pt-4">
        {loading && !summary ? (
          <>
            <Skeleton className="h-8 w-24" />
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-20 w-full" />
          </>
        ) : !summary ? (
          <p className="text-sm text-muted-foreground">
            Ringkasan unit tidak tersedia. Coba muat ulang dashboard.
          </p>
        ) : (
          <>
            <div className="space-y-1">
              <p className="text-2xl font-semibold tracking-tight">
                {formatNumber(active)}
              </p>
              <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
                  {readyPct}% siap
                </p>
                <p className="text-sm text-muted-foreground">
                  dari {formatNumber(total)} unit
                </p>
              </div>
            </div>

            <div
              className="flex h-5 w-full items-stretch gap-[3px]"
              aria-hidden="true"
            >
              {Array.from({ length: segments }).map((_, index) => {
                const segmentClass =
                  index < activeSegments
                    ? "bg-foreground"
                    : index < activeSegments + breakdownSegments
                      ? "bg-chart-1"
                      : "bg-muted-foreground/30";
                return (
                  <span
                    key={index}
                    className={cn("h-full flex-1 rounded-[1px]", segmentClass)}
                  />
                );
              })}
            </div>

            <div className="mt-auto grid gap-2.5">
              <InsightRow label="Aktif" value={active} color="var(--foreground)" />
              <InsightRow
                label="Breakdown"
                value={breakdown}
                color="var(--chart-1)"
              />
              <InsightRow
                label="Nonaktif"
                value={inactive}
                color="var(--muted-foreground)"
              />
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function InsightRow({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: string;
}) {
  return (
    <div className="flex items-center justify-between gap-3 text-sm">
      <span className="flex items-center gap-2 text-muted-foreground">
        <span
          className="size-2.5 rounded-full"
          style={{ backgroundColor: color }}
          aria-hidden="true"
        />
        {label}
      </span>
      <span className="font-medium">{formatNumber(value)}</span>
    </div>
  );
}
