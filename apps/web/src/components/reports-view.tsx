"use client";

import { CircleAlert, Download, FileText, RotateCcw, Search } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { FilterBar } from "@/components/list/list-filter-bar";
import { PageHeader } from "@/components/layout/page-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
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
import { TablePagination } from "@/components/table-pagination";
import { ApiError, proxyGet } from "@/lib/api-client";
import {
  daysAgoWitaDate,
  formatDateTimeWita,
  formatNumber,
  todayWitaDate,
} from "@/lib/time";
import type { DataResponse, ReportTable } from "@/lib/types";

const REPORTS = [
  { value: "pm-compliance", label: "Kepatuhan PM", hasGroup: true },
  { value: "shortage-aging", label: "Umur shortage", hasGroup: false },
  { value: "sparepart-consumption", label: "Pemakaian sparepart", hasGroup: false },
  { value: "inventory-movement", label: "Pergerakan inventory", hasGroup: false },
  { value: "component-replacement", label: "Penggantian komponen", hasGroup: false },
] as const;

type ReportName = (typeof REPORTS)[number]["value"];

export function ReportsView() {
  const [report, setReport] = useState<ReportName>("pm-compliance");
  const [from, setFrom] = useState(daysAgoWitaDate(30));
  const [to, setTo] = useState(todayWitaDate());
  const [bucket, setBucket] = useState("day");
  const [groupBy, setGroupBy] = useState("plan");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [table, setTable] = useState<ReportTable | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [unauthorized, setUnauthorized] = useState(false);

  const selected = REPORTS.find((item) => item.value === report) ?? REPORTS[0];

  const query = useCallback(
    (format?: string) => {
      const params = new URLSearchParams({
        from,
        to,
        bucket,
        page: String(page),
        limit: String(pageSize),
      });
      if (selected.hasGroup) {
        params.set("group_by", groupBy);
      }
      if (format) {
        params.set("format", format);
      }
      return params.toString();
    },
    [bucket, from, groupBy, page, pageSize, selected.hasGroup, to],
  );

  const load = useCallback(async () => {
    setLoading(true);
    setUnauthorized(false);
    try {
      const result = await proxyGet<DataResponse<ReportTable>>(`/reports/${report}?${query()}`);
      setTable(result.data);
      setError("");
    } catch (loadError) {
      if (loadError instanceof ApiError && (loadError.status === 401 || loadError.status === 403)) {
        setUnauthorized(true);
      } else {
        setError(loadError instanceof ApiError ? loadError.message : "Laporan tidak dapat dimuat.");
      }
    } finally {
      setLoading(false);
    }
  }, [query, report]);

  useEffect(() => {
    void load();
  }, [load]);

  const downloadCSV = async () => {
    try {
      const response = await fetch(`/api/proxy/reports/${report}?${query("csv")}`, {
        credentials: "same-origin",
        cache: "no-store",
        headers: { Accept: "text/csv" },
      });
      if (!response.ok) {
        throw new Error("CSV tidak dapat dibuat.");
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${report}.csv`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch (downloadError) {
      setError(downloadError instanceof Error ? downloadError.message : "CSV tidak dapat dibuat.");
    }
  };

  const reset = () => {
    setReport("pm-compliance");
    setFrom(daysAgoWitaDate(30));
    setTo(todayWitaDate());
    setBucket("day");
    setGroupBy("plan");
    setPage(1);
    setPageSize(25);
  };

  const totalPages = Math.max(1, Math.ceil((table?.meta.total ?? 0) / pageSize));

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Laporan"
        description="Laporan kepatuhan PM, shortage, pemakaian sparepart, pergerakan stok, dan penggantian komponen."
        actions={<Badge variant="secondary">WITA</Badge>}
      />

      {unauthorized ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Akses ditolak</AlertTitle>
          <AlertDescription>Anda tidak memiliki izin untuk melihat laporan ini.</AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardContent className="pt-6">
          <FilterBar aria-label="Filter laporan" onSubmit={(event) => { event.preventDefault(); setPage(1); void load(); }}>
            <Field className="md:w-56">
              <FieldLabel htmlFor="report-kind">Jenis laporan</FieldLabel>
              <NativeSelect
                id="report-kind"
                value={report}
                onChange={(event) => {
                  setReport(event.target.value as ReportName);
                  setPage(1);
                }}
              >
                {REPORTS.map((item) => (
                  <NativeSelectOption key={item.value} value={item.value}>
                    {item.label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field className="md:w-44">
              <FieldLabel htmlFor="report-from">Dari</FieldLabel>
              <Input
                id="report-from"
                type="date"
                value={from}
                max={to}
                onChange={(event) => setFrom(event.target.value)}
              />
            </Field>
            <Field className="md:w-44">
              <FieldLabel htmlFor="report-to">Sampai</FieldLabel>
              <Input
                id="report-to"
                type="date"
                value={to}
                min={from}
                onChange={(event) => setTo(event.target.value)}
              />
            </Field>
            <Field className="md:w-36">
              <FieldLabel htmlFor="report-bucket">Bucket</FieldLabel>
              <NativeSelect id="report-bucket" value={bucket} onChange={(event) => setBucket(event.target.value)}>
                <NativeSelectOption value="day">Harian</NativeSelectOption>
                <NativeSelectOption value="week">Mingguan</NativeSelectOption>
                <NativeSelectOption value="month">Bulanan</NativeSelectOption>
              </NativeSelect>
            </Field>
            {selected.hasGroup ? (
              <Field className="md:w-36">
                <FieldLabel htmlFor="report-group">Kelompok</FieldLabel>
                <NativeSelect id="report-group" value={groupBy} onChange={(event) => setGroupBy(event.target.value)}>
                  <NativeSelectOption value="plan">Rencana PM</NativeSelectOption>
                  <NativeSelectOption value="unit">Unit</NativeSelectOption>
                  <NativeSelectOption value="model">Model</NativeSelectOption>
                </NativeSelect>
              </Field>
            ) : null}
            <div className="flex gap-2 md:ml-auto">
              <Button type="submit" variant="outline">
                <Search aria-hidden="true" />
                Terapkan
              </Button>
              <Button type="button" variant="ghost" onClick={reset}>
                <RotateCcw aria-hidden="true" />
                Reset
              </Button>
              <Button type="button" onClick={() => void downloadCSV()} disabled={unauthorized}>
                <Download aria-hidden="true" />
                Ekspor CSV
              </Button>
            </div>
          </FilterBar>
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Laporan tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
          <div>
            <CardTitle className="flex items-center gap-2">
              <FileText className="size-4 text-muted-foreground" aria-hidden="true" />
              {selected.label}
            </CardTitle>
            <CardDescription>
              Batas hari WITA, dari {from} sampai {to}.
            </CardDescription>
          </div>
          <Badge variant="outline">{table?.meta.total ?? 0} baris</Badge>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {loading ? (
            <div className="space-y-3" role="status" aria-label="Memuat laporan">
              {Array.from({ length: 8 }).map((_, index) => (
                <Skeleton key={index} className="h-9 w-full" />
              ))}
            </div>
          ) : !table || table.rows.length === 0 ? (
            <EmptyState />
          ) : (
            <>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      {table.columns.map((column) => (
                        <TableHead key={column.key}>{column.label}</TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {table.rows.map((row, index) => (
                      <TableRow key={`${String(row[table.columns[0]?.key ?? ""] ?? "")}-${index}`}>
                        {table.columns.map((column) => (
                          <TableCell key={column.key}>{cell(row[column.key])}</TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              <TablePagination
                page={page}
                pageSize={pageSize}
                total={table.meta.total}
                totalPages={totalPages}
                loading={loading}
                itemLabel="baris"
                onPageChange={setPage}
                onPageSizeChange={(size) => { setPageSize(size); setPage(1); }}
              />
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function EmptyState() {
  return (
    <Empty className="py-12">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <FileText aria-hidden="true" />
        </EmptyMedia>
        <EmptyTitle>Tidak ada data</EmptyTitle>
        <EmptyDescription>
          Ubah periode atau jenis laporan untuk melihat data.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
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
