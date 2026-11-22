"use client";

import { CircleAlert, Filter, History, RotateCcw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { FilterBar } from "@/components/list/list-filter-bar";
import { TablePagination } from "@/components/table-pagination";
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
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
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
import {
  WorkOrderMaintenanceBadge,
  WorkOrderPriorityBadge,
  WorkOrderStatusBadge,
  workOrderMaintenanceLabels,
  workOrderStatusLabels,
} from "@/components/workorder-badges";
import { ApiError, proxyGet } from "@/lib/api-client";
import type {
  ListResponse,
  UnitMaintenanceHistory,
  WorkOrderMaintenanceType,
} from "@/lib/types";
import { formatDateTimeWita, formatDurationMinutes, formatNumber } from "@/lib/time";

type TimelineFilters = {
  status: string;
  maintenanceType: string;
  from: string;
  to: string;
};

const emptyFilters: TimelineFilters = {
  status: "",
  maintenanceType: "",
  from: "",
  to: "",
};

const maintenanceTypes = Object.entries(workOrderMaintenanceLabels) as [
  WorkOrderMaintenanceType,
  string,
][];

export function UnitTimelinePanel({ unitId }: { unitId: number }) {
  const [items, setItems] = useState<UnitMaintenanceHistory[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [filters, setFilters] = useState<TimelineFilters>(emptyFilters);
  const [query, setQuery] = useState<TimelineFilters>(emptyFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(limit) });
      if (query.status) params.set("status", query.status);
      if (query.maintenanceType) params.set("maintenance_type", query.maintenanceType);
      if (query.from) params.set("from", query.from);
      if (query.to) params.set("to", query.to);
      const result = await proxyGet<ListResponse<UnitMaintenanceHistory>>(
        `/units/${unitId}/history?${params.toString()}`,
      );
      setItems(result.data);
      setTotal(result.meta.total);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError ? loadError.message : "Timeline maintenance tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [unitId, page, limit, query]);

  useEffect(() => {
    void load();
  }, [load]);

  const applyFilters = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPage(1);
    setQuery({ ...filters });
  };

  const resetFilters = () => {
    setFilters(emptyFilters);
    setQuery(emptyFilters);
    setPage(1);
  };

  const totalPages = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="border-b">
          <CardTitle>Filter timeline</CardTitle>
        </CardHeader>
        <CardContent>
          <FilterBar onSubmit={applyFilters}>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="history-status">Status</FieldLabel>
              <NativeSelect
                id="history-status"
                value={filters.status}
                onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value }))}
              >
                <NativeSelectOption value="">Semua status</NativeSelectOption>
                {Object.entries(workOrderStatusLabels).map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>
                    {label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field className="w-full md:w-44">
              <FieldLabel htmlFor="history-type">Tipe maintenance</FieldLabel>
              <NativeSelect
                id="history-type"
                value={filters.maintenanceType}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, maintenanceType: event.target.value }))
                }
              >
                <NativeSelectOption value="">Semua tipe</NativeSelectOption>
                {maintenanceTypes.map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>
                    {label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="history-from">Dari tanggal</FieldLabel>
              <Input
                id="history-from"
                type="date"
                value={filters.from}
                onChange={(event) => setFilters((current) => ({ ...current, from: event.target.value }))}
              />
            </Field>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="history-to">Sampai tanggal</FieldLabel>
              <Input
                id="history-to"
                type="date"
                value={filters.to}
                onChange={(event) => setFilters((current) => ({ ...current, to: event.target.value }))}
              />
            </Field>
            <div className="flex gap-2">
              <Button type="submit" size="sm" disabled={loading}>
                <Filter aria-hidden="true" /> Terapkan
              </Button>
              <Button type="button" size="sm" variant="outline" onClick={resetFilters} disabled={loading}>
                <RotateCcw aria-hidden="true" /> Reset
              </Button>
            </div>
          </FilterBar>
        </CardContent>
      </Card>

      <Card className="gap-0 overflow-hidden">
        <CardHeader className="border-b">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <CardTitle>Timeline maintenance</CardTitle>
              <p className="text-sm text-muted-foreground">
                Semua work order P2H, manual, dan PM dalam urutan terbaru.
              </p>
            </div>
            <Badge variant="secondary" className="h-6 px-2.5">
              {total} work order
            </Badge>
          </div>
        </CardHeader>

        {error ? (
          <CardContent className="py-6">
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Timeline tidak dapat dimuat</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          </CardContent>
        ) : loading ? (
          <CardContent className="flex flex-col gap-2 py-6">
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className="h-12 w-full" />
            ))}
          </CardContent>
        ) : items.length === 0 ? (
          <CardContent className="py-10">
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <History aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada riwayat maintenance.</EmptyTitle>
                <EmptyDescription>
                  Work order untuk unit ini akan muncul setelah dibuat.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          </CardContent>
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableCaption className="sr-only">Timeline maintenance unit</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>Work order</TableHead>
                    <TableHead>Tipe / status</TableHead>
                    <TableHead>Tanggal</TableHead>
                    <TableHead>Mekanik</TableHead>
                    <TableHead>Temuan</TableHead>
                    <TableHead>Material</TableHead>
                    <TableHead>Durasi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.wo_id}>
                      <TableCell className="min-w-56">
                        <p className="font-mono text-xs text-muted-foreground">{item.wo_code}</p>
                        <p className="font-medium">{item.title}</p>
                        <div className="mt-1 flex flex-wrap items-center gap-1">
                          <Badge variant="outline" className="h-5 px-1.5 text-[10px] uppercase">
                            {item.source}
                          </Badge>
                          <WorkOrderPriorityBadge priority={item.priority} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col items-start gap-1">
                          <WorkOrderMaintenanceBadge type={item.maintenance_type} />
                          <WorkOrderStatusBadge status={item.status} />
                        </div>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        <p>{formatDateTimeWita(item.closed_at ?? item.started_at ?? item.created_at)}</p>
                        <p className="text-xs text-muted-foreground">
                          {item.closed_at ? "Selesai" : item.started_at ? "Mulai" : "Dibuat"}
                        </p>
                      </TableCell>
                      <TableCell className="min-w-36 text-sm">
                        <p>{item.assignee_name || "-"}</p>
                        <p className="text-xs text-muted-foreground">
                          Supervisor: {item.supervisor_name || "-"}
                        </p>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        <p>{item.findings_count} findings</p>
                        <p className="text-xs text-muted-foreground">
                          Checklist {item.checklist_done}/{item.checklist_total}
                          {item.checklist_not_ok > 0 ? ` · ${item.checklist_not_ok} not OK` : ""}
                        </p>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        <p>{item.usage_count} sparepart</p>
                        <p className="text-xs text-muted-foreground">{formatNumber(item.usage_qty)} qty</p>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        {formatDurationMinutes(item.elapsed_minutes)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <div className="border-t px-4 py-3">
              <TablePagination
                page={page}
                pageSize={limit}
                total={total}
                totalPages={totalPages}
                loading={loading}
                itemLabel="work order"
                onPageChange={setPage}
                onPageSizeChange={(size) => {
                  setLimit(size);
                  setPage(1);
                }}
              />
            </div>
          </>
        )}
      </Card>
    </div>
  );
}
