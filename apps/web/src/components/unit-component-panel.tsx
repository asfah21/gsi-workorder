"use client";

import { CircleAlert, Filter, PackageSearch, RotateCcw } from "lucide-react";
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
import { WorkOrderMaintenanceBadge, WorkOrderStatusBadge } from "@/components/workorder-badges";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { ListResponse, UnitComponentHistory } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

type ComponentFilters = {
  from: string;
  to: string;
};

const emptyFilters: ComponentFilters = { from: "", to: "" };

export function UnitComponentPanel({ unitId }: { unitId: number }) {
  const [items, setItems] = useState<UnitComponentHistory[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [filters, setFilters] = useState<ComponentFilters>(emptyFilters);
  const [query, setQuery] = useState<ComponentFilters>(emptyFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({ page: String(page), limit: String(limit) });
      if (query.from) params.set("from", query.from);
      if (query.to) params.set("to", query.to);
      const result = await proxyGet<ListResponse<UnitComponentHistory>>(
        `/units/${unitId}/components?${params.toString()}`,
      );
      setItems(result.data);
      setTotal(result.meta.total);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Riwayat komponen tidak dapat dimuat.",
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
          <CardTitle>Filter pemakaian sparepart</CardTitle>
        </CardHeader>
        <CardContent>
          <FilterBar onSubmit={applyFilters}>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="component-from">Dari tanggal</FieldLabel>
              <Input
                id="component-from"
                type="date"
                value={filters.from}
                onChange={(event) => setFilters((current) => ({ ...current, from: event.target.value }))}
              />
            </Field>
            <Field className="w-full md:w-40">
              <FieldLabel htmlFor="component-to">Sampai tanggal</FieldLabel>
              <Input
                id="component-to"
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
              <CardTitle>Riwayat komponen</CardTitle>
              <p className="text-sm text-muted-foreground">
                Satu event adalah satu work order dan satu komponen. Jika komponen tidak dipilih,
                sparepart dipakai sebagai fallback.
              </p>
            </div>
            <Badge variant="secondary" className="h-6 px-2.5">
              {total} event
            </Badge>
          </div>
        </CardHeader>

        {error ? (
          <CardContent className="py-6">
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Riwayat komponen tidak dapat dimuat</AlertTitle>
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
                  <PackageSearch aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada pemakaian sparepart.</EmptyTitle>
                <EmptyDescription>
                  Issue material yang memiliki komponen atau sparepart akan muncul di sini.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          </CardContent>
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableCaption className="sr-only">Riwayat komponen unit</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>Komponen</TableHead>
                    <TableHead>Work order</TableHead>
                    <TableHead>Waktu pasang</TableHead>
                    <TableHead>Qty</TableHead>
                    <TableHead>HM / KM</TableHead>
                    <TableHead>Interval</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={`${item.wo_id}-${item.component_id ?? "s"}-${item.fallback_sparepart_id ?? ""}`}>
                      <TableCell className="min-w-56">
                        <div className="flex flex-wrap items-center gap-1">
                          <p className="font-medium">{item.component_name || "-"}</p>
                          <Badge variant="outline">#{item.replacement_no}</Badge>
                          {item.is_fallback ? <Badge variant="secondary">Sparepart</Badge> : null}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {item.component_code || item.sparepart_codes || "-"}
                        </p>
                        {!item.is_fallback && item.sparepart_codes ? (
                          <p className="mt-1 text-xs text-muted-foreground">
                            Part: {item.sparepart_codes}
                          </p>
                        ) : null}
                      </TableCell>
                      <TableCell className="min-w-48">
                        <p className="font-mono text-xs text-muted-foreground">{item.wo_code}</p>
                        <p className="font-medium">{item.wo_title}</p>
                        <div className="mt-1 flex flex-wrap items-center gap-1">
                          <WorkOrderMaintenanceBadge type={item.maintenance_type} />
                          <WorkOrderStatusBadge status={item.wo_status} />
                        </div>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        {formatDateTimeWita(item.used_at)}
                        {item.closed_at ? (
                          <p className="text-xs text-muted-foreground">
                            WO selesai {formatDateTimeWita(item.closed_at)}
                          </p>
                        ) : null}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        {formatNumber(item.qty)}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        <p>HM {item.hm !== undefined ? formatNumber(item.hm) : "-"}</p>
                        <p className="text-xs text-muted-foreground">
                          KM {item.km !== undefined ? formatNumber(item.km) : "-"}
                        </p>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-sm">
                        <p>
                          {item.days_since_previous !== undefined
                            ? `${formatNumber(item.days_since_previous)} hari`
                            : "-"}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {item.hm_since_previous !== undefined
                            ? `HM +${formatNumber(item.hm_since_previous)}`
                            : "HM -"}
                          {" · "}
                          {item.km_since_previous !== undefined
                            ? `KM +${formatNumber(item.km_since_previous)}`
                            : "KM -"}
                        </p>
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
                itemLabel="event"
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
