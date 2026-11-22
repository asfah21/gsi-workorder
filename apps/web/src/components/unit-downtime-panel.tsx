"use client";

import {
  CircleAlert,
  Clock3,
  Filter,
  Hourglass,
  RotateCcw,
  TimerOff,
  Wrench,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { FilterBar } from "@/components/list/list-filter-bar";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { WorkOrderMaintenanceBadge } from "@/components/workorder-badges";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DataResponse, UnitDowntimeResponse } from "@/lib/types";
import { formatDateTimeWita, formatDurationMinutes } from "@/lib/time";
import { cn } from "@/lib/utils";

type Period = {
  from: string;
  to: string;
};

const emptyPeriod: Period = { from: "", to: "" };

export function UnitDowntimePanel({ unitId }: { unitId: number }) {
  const [data, setData] = useState<UnitDowntimeResponse | null>(null);
  const [period, setPeriod] = useState<Period>(emptyPeriod);
  const [query, setQuery] = useState<Period>(emptyPeriod);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({ limit: "50" });
      if (query.from) params.set("from", query.from);
      if (query.to) params.set("to", query.to);
      const result = await proxyGet<DataResponse<UnitDowntimeResponse>>(
        `/units/${unitId}/downtime?${params.toString()}`,
      );
      setData(result.data);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError ? loadError.message : "Downtime tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [unitId, query]);

  useEffect(() => {
    void load();
  }, [load]);

  const applyPeriod = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setQuery({ ...period });
  };

  const resetPeriod = () => {
    setPeriod(emptyPeriod);
    setQuery(emptyPeriod);
  };

  const summary = data?.summary;

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="border-b">
          <CardTitle>Filter periode downtime</CardTitle>
        </CardHeader>
        <CardContent>
          <FilterBar onSubmit={applyPeriod}>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="downtime-from">Dari tanggal</FieldLabel>
              <Input
                id="downtime-from"
                type="date"
                value={period.from}
                onChange={(event) => setPeriod((current) => ({ ...current, from: event.target.value }))}
              />
            </Field>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="downtime-to">Sampai tanggal</FieldLabel>
              <Input
                id="downtime-to"
                type="date"
                value={period.to}
                onChange={(event) => setPeriod((current) => ({ ...current, to: event.target.value }))}
              />
            </Field>
            <div className="flex gap-2">
              <Button type="submit" size="sm" disabled={loading}>
                <Filter aria-hidden="true" /> Terapkan
              </Button>
              <Button type="button" size="sm" variant="outline" onClick={resetPeriod} disabled={loading}>
                <RotateCcw aria-hidden="true" /> Reset
              </Button>
            </div>
          </FilterBar>
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Downtime tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : loading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-28 rounded-xl" />
          ))}
        </div>
      ) : summary ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
            <DowntimeMetric
              label="Total downtime"
              value={formatDurationMinutes(summary.total_minutes)}
              hint={`${summary.events} work order selesai`}
              icon={TimerOff}
              tone="rose"
            />
            <DowntimeMetric
              label="Unplanned"
              value={formatDurationMinutes(summary.unplanned_minutes)}
              hint={`${summary.unplanned_events} breakdown/corrective`}
              icon={Wrench}
              tone="amber"
            />
            <DowntimeMetric
              label="Planned"
              value={formatDurationMinutes(summary.planned_minutes)}
              hint={`${summary.planned_events} preventive`}
              icon={Clock3}
              tone="sky"
            />
            <DowntimeMetric
              label="Waiting part"
              value={formatDurationMinutes(summary.waiting_minutes)}
              hint="Bagian dari downtime"
              icon={Hourglass}
              tone="violet"
            />
            <DowntimeMetric
              label="MTTR"
              value={formatDurationMinutes(summary.mttr_minutes)}
              hint="Rata-rata per unplanned WO"
              icon={Wrench}
              tone="emerald"
            />
          </div>

          <Card className="gap-0 overflow-hidden">
            <CardHeader className="border-b">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <CardTitle>Event downtime</CardTitle>
                  <p className="text-sm text-muted-foreground">
                    Total downtime memakai union interval, sehingga WO yang overlap tidak dihitung
                    dua kali.
                  </p>
                </div>
                <Badge variant="secondary" className="h-6 px-2.5">
                  {summary.events} event
                </Badge>
              </div>
            </CardHeader>

            {data.events.length === 0 ? (
              <CardContent className="py-10">
                <Empty className="border-0">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <TimerOff aria-hidden="true" />
                    </EmptyMedia>
                    <EmptyTitle>Belum ada downtime selesai.</EmptyTitle>
                    <EmptyDescription>
                      Hanya work order selesai dengan tipe preventive, corrective, atau breakdown
                      yang dihitung.
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              </CardContent>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <Table>
                    <TableCaption className="sr-only">Event downtime unit</TableCaption>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Work order</TableHead>
                        <TableHead>Klasifikasi</TableHead>
                        <TableHead>Mulai</TableHead>
                        <TableHead>Selesai</TableHead>
                        <TableHead>Durasi</TableHead>
                        <TableHead>Waiting part</TableHead>
                        <TableHead>Mekanik</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.events.map((event) => (
                        <TableRow key={event.wo_id}>
                          <TableCell className="min-w-48">
                            <p className="font-mono text-xs text-muted-foreground">{event.wo_code}</p>
                            <p className="font-medium">{event.title}</p>
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-col items-start gap-1">
                              <Badge
                                variant={event.downtime_class === "unplanned" ? "destructive" : "secondary"}
                                className={cn(
                                  event.downtime_class === "planned" &&
                                    "border-transparent bg-sky-500/10 text-sky-700 dark:text-sky-300",
                                )}
                              >
                                {event.downtime_class === "unplanned" ? "Unplanned" : "Planned"}
                              </Badge>
                              <WorkOrderMaintenanceBadge type={event.maintenance_type} />
                            </div>
                          </TableCell>
                          <TableCell className="whitespace-nowrap text-sm">
                            {formatDateTimeWita(event.downtime_start)}
                          </TableCell>
                          <TableCell className="whitespace-nowrap text-sm">
                            {formatDateTimeWita(event.downtime_end)}
                          </TableCell>
                          <TableCell className="whitespace-nowrap text-sm font-medium">
                            {formatDurationMinutes(event.downtime_minutes)}
                          </TableCell>
                          <TableCell className="whitespace-nowrap text-sm">
                            {formatDurationMinutes(event.waiting_minutes)}
                          </TableCell>
                          <TableCell className="text-sm">
                            <p>{event.assignee_name || "-"}</p>
                            <p className="text-xs text-muted-foreground">
                              Supervisor: {event.supervisor_name || "-"}
                            </p>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                {summary.events > data.events.length ? (
                  <div className="border-t px-4 py-3 text-sm text-muted-foreground">
                    Menampilkan {data.events.length} event terbaru dari {summary.events} event pada
                    periode ini.
                  </div>
                ) : null}
              </>
            )}
          </Card>
        </>
      ) : null}
    </div>
  );
}

const downtimeTones = {
  rose: "bg-destructive/10 text-destructive",
  amber: "bg-amber-500/10 text-amber-700 dark:text-amber-300",
  sky: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
  violet: "bg-violet-500/10 text-violet-700 dark:text-violet-300",
  emerald: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
} as const;

function DowntimeMetric({
  label,
  value,
  hint,
  icon: Icon,
  tone,
}: {
  label: string;
  value: string;
  hint: string;
  icon: LucideIcon;
  tone: keyof typeof downtimeTones;
}) {
  return (
    <div className="rounded-xl border border-border bg-background p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">{label}</p>
          <p className="mt-2 text-xl font-semibold">{value}</p>
          <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
        </div>
        <span className={cn("grid size-9 shrink-0 place-items-center rounded-md", downtimeTones[tone])}>
          <Icon className="size-5" aria-hidden="true" />
        </span>
      </div>
    </div>
  );
}
