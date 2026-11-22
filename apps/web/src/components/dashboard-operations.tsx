"use client";

import {
  CalendarClock,
  CircleAlert,
  Clock,
  PackageSearch,
  RefreshCcw,
  TriangleAlert,
  Wrench,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { FilterBar } from "@/components/list/list-filter-bar";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
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
  formatDateTimeWita,
  formatDurationMinutes,
  formatNumber,
  todayWitaDate,
} from "@/lib/time";
import type {
  DashboardInventory,
  DashboardMaintenance,
  DashboardMaterial,
  DashboardPM,
  DataResponse,
  ReportTable,
} from "@/lib/types";

type OperationsData = {
  pm?: DashboardPM;
  material?: DashboardMaterial;
  inventory?: DashboardInventory;
  maintenance?: DashboardMaintenance;
};

export function DashboardOperations() {
  const [days, setDays] = useState("30");
  const [data, setData] = useState<OperationsData>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({
      from: daysAgoWitaDate(Number(days)),
      to: todayWitaDate(),
      bucket: "day",
      limit: "6",
    });
    try {
      const [pm, material, inventory, maintenance] = await Promise.all([
        proxyGet<DataResponse<DashboardPM>>(`/dashboard/pm?${params}`),
        proxyGet<DataResponse<DashboardMaterial>>(`/dashboard/material?${params}`),
        proxyGet<DataResponse<DashboardInventory>>(`/dashboard/inventory?${params}`),
        proxyGet<DataResponse<DashboardMaintenance>>(`/dashboard/maintenance?${params}`),
      ]);
      setData({ pm: pm.data, material: material.data, inventory: inventory.data, maintenance: maintenance.data });
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
  }, [load]);

  const pm = data.pm;
  const material = data.material;
  const inventory = data.inventory;
  const maintenance = data.maintenance;
  const lowStock = numberValue(inventory?.summary.low) + numberValue(inventory?.summary.critical);

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="pt-6">
          <FilterBar aria-label="Filter periode dashboard operasional">
            <Field className="md:w-52">
              <FieldLabel htmlFor="operations-period">Periode</FieldLabel>
              <NativeSelect
                id="operations-period"
                value={days}
                onChange={(event) => setDays(event.target.value)}
              >
                <NativeSelectOption value="7">7 hari terakhir</NativeSelectOption>
                <NativeSelectOption value="30">30 hari terakhir</NativeSelectOption>
                <NativeSelectOption value="90">90 hari terakhir</NativeSelectOption>
              </NativeSelect>
            </Field>
            <div className="flex gap-2 md:ml-auto">
              <Button type="button" variant="outline" onClick={() => void load()} disabled={loading}>
                <RefreshCcw className={loading ? "animate-spin" : undefined} aria-hidden="true" />
                Muat ulang
              </Button>
            </div>
          </FilterBar>
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Data operasional tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="PM terlambat"
          value={pm?.summary.overdue}
          loading={loading && !pm}
          icon={CalendarClock}
          hint="Jadwal overdue aktif"
        />
        <MetricCard
          label="Material shortage"
          value={material?.summary.shortage}
          loading={loading && !material}
          icon={TriangleAlert}
          hint="WO dengan item kurang"
        />
        <MetricCard
          label="Stok rendah/kritis"
          value={inventory ? lowStock : undefined}
          loading={loading && !inventory}
          icon={PackageSearch}
          hint="Sparepart di bawah policy"
        />
        <MetricCard
          label="Downtime"
          value={maintenance?.summary.total_minutes}
          loading={loading && !maintenance}
          icon={Clock}
          hint="Menit, selesai dalam periode"
          duration
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
            <div>
              <CardTitle>Kesiapan PM &amp; material</CardTitle>
              <CardDescription>Bucket jatuh tempo dan umur shortage.</CardDescription>
            </div>
            <Link href="/reports" className={buttonVariants({ variant: "outline", size: "sm" })}>
              Laporan
            </Link>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <SectionLabel icon={CalendarClock} label="Jadwal PM" />
            <BucketList table={pm?.due} loading={loading && !pm} labelKey="bucket" valueKey="unit_code" />
            <SectionLabel icon={TriangleAlert} label="Umur shortage" />
            <BucketList table={material?.aging} loading={loading && !material} labelKey="age_bucket" valueKey="items" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Aktivitas &amp; downtime</CardTitle>
            <CardDescription>Ringkasan WO dan unit yang menunggu sparepart.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <MiniValue label="WO selesai" value={maintenance?.summary.done} loading={loading && !maintenance} />
              <MiniValue label="Backlog" value={maintenance?.summary.backlog} loading={loading && !maintenance} />
              <MiniValue label="Breakdown" value={maintenance?.summary.breakdown} loading={loading && !maintenance} />
              <MiniValue label="MTTR" value={maintenance?.summary.mttr_minutes} loading={loading && !maintenance} duration />
            </div>
            <SectionLabel icon={Wrench} label="Downtime per bucket" />
            <BucketList table={maintenance?.downtime} loading={loading && !maintenance} labelKey="bucket" valueKey="downtime_minutes" />
            <SectionLabel icon={Clock} label="Menunggu sparepart" />
            <CompactTable table={maintenance?.waiting} loading={loading && !maintenance} columns={["wo_code", "unit_code", "waiting_minutes"]} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function MetricCard({
  label,
  value,
  loading,
  icon: Icon,
  hint,
  duration,
}: {
  label: string;
  value?: number;
  loading: boolean;
  icon: LucideIcon;
  hint: string;
  duration?: boolean;
}) {
  return (
    <Card className="min-h-[132px]">
      <CardContent className="flex flex-1 flex-col gap-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-sm text-muted-foreground">{label}</p>
            {loading ? (
              <Skeleton className="mt-2 h-8 w-16" />
            ) : (
              <p className="mt-2 text-2xl font-semibold">
                {duration ? formatDurationMinutes(numberValue(value)) : formatNumber(numberValue(value))}
              </p>
            )}
          </div>
          <span className="grid size-9 shrink-0 place-items-center rounded-md border border-border bg-muted/40 text-muted-foreground">
            <Icon className="size-4" aria-hidden="true" />
          </span>
        </div>
        <p className="mt-auto text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}

function SectionLabel({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <p className="flex items-center gap-2 text-sm font-medium">
      <Icon className="size-4 text-muted-foreground" aria-hidden="true" />
      {label}
    </p>
  );
}

function MiniValue({ label, value, loading, duration }: { label: string; value?: number; loading: boolean; duration?: boolean }) {
  return (
    <div className="rounded-lg border border-border bg-muted/20 p-2">
      <p className="type-overline-xs text-muted-foreground">{label}</p>
      {loading ? (
        <Skeleton className="mt-1 h-5 w-12" />
      ) : (
        <p className="mt-1 text-lg font-semibold">
          {duration ? formatDurationMinutes(numberValue(value)) : formatNumber(numberValue(value))}
        </p>
      )}
    </div>
  );
}

function BucketList({
  table,
  loading,
  labelKey,
  valueKey = "count",
}: {
  table?: ReportTable;
  loading: boolean;
  labelKey: string;
  valueKey?: string;
}) {
  if (loading || !table) {
    return (
      <div className="space-y-2" role="status" aria-label="Memuat ringkasan bucket">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-7 w-full" />
        ))}
      </div>
    );
  }
  if (table.rows.length === 0) {
    return <p className="text-sm text-muted-foreground">Tidak ada data pada periode ini.</p>;
  }
  return (
    <div className="flex flex-wrap gap-2">
      {table.rows.slice(0, 6).map((row, index) => (
        <span
          key={`${String(row[labelKey] ?? "")}-${index}`}
          className="inline-flex items-center gap-2 rounded-md border border-border bg-muted/30 px-2 py-1 text-xs"
        >
          <span className="font-medium">{cell(row[labelKey])}</span>
          <span className="text-muted-foreground">{cell(row[valueKey])}</span>
        </span>
      ))}
    </div>
  );
}

function CompactTable({
  table,
  loading,
  columns,
}: {
  table?: ReportTable;
  loading: boolean;
  columns: string[];
}) {
  if (loading || !table) {
    return <Skeleton className="h-24 w-full" />;
  }
  if (table.rows.length === 0) {
    return <p className="text-sm text-muted-foreground">Tidak ada unit yang menunggu sparepart.</p>;
  }
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((key) => (
              <TableHead key={key}>{table.columns.find((column) => column.key === key)?.label ?? key}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {table.rows.map((row, index) => (
            <TableRow key={`${String(row.wo_code ?? "")}-${index}`}>
              {columns.map((key) => (
                <TableCell key={key}>{cell(row[key])}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function numberValue(value?: number): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function cell(value: unknown): string {
  if (value === null || value === undefined) {
    return "-";
  }
  if (typeof value === "number") {
    return formatNumber(value);
  }
  if (typeof value === "string") {
    return /^\d{4}-\d{2}-\d{2}T/.test(value) ? formatDateTimeWita(value) : value;
  }
  return String(value);
}
