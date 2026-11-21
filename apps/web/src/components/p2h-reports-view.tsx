"use client";

import {
  CircleAlert,
  Eye,
  Filter,
  RotateCcw,
  Search,
  ShieldCheck,
} from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { CandidateStatusBadge, P2HStateBadge } from "@/components/p2h-badges";
import { TablePagination } from "@/components/table-pagination";
import { TableSortHeader } from "@/components/table-sort-header";
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
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useListParams } from "@/hooks/use-list-params";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DataResponse, ListResponse, P2HReport } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const sortableColumns = [
  "report_date",
  "name",
  "unit_code",
  "state",
  "findings_count",
  "checked_items",
] as const;

const stateLabels: Record<string, string> = {
  "": "Semua state",
  draft: "Draft",
  submitted: "Submitted",
  approved: "Approved",
  rejected: "Rejected",
};

export function P2HReportsView() {
  const searchParams = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const state = searchParams.get("state") ?? "";
  const list = useListParams({
    defaultSort: "report_date",
    allowedSorts: sortableColumns,
    defaultOrder: "desc",
  });

  const [items, setItems] = useState<P2HReport[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(q);
  const [queryError, setQueryError] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<P2HReport | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");

  const hasFilters = q.trim() !== "" || state !== "";
  const totalPages = Math.max(1, Math.ceil(total / list.pageSize));

  useEffect(() => {
    setQuery(q);
    setQueryError("");
  }, [q]);

  const loadList = useCallback(async () => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams({
      page: String(list.page),
      limit: String(list.pageSize),
      sort: list.sort,
      order: list.order,
    });
    if (q.trim()) params.set("q", q.trim());
    if (state) params.set("state", state);

    try {
      const result = await proxyGet<ListResponse<P2HReport>>(
        `/p2h/reports?${params.toString()}`,
      );
      setItems(result.data);
      setTotal(result.meta.total);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Laporan P2H tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [list.order, list.page, list.pageSize, list.sort, q, state]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const loadDetail = async (id: number) => {
    setDetailLoading(true);
    setDetailError("");
    setDetailOpen(true);
    try {
      const report = await proxyGet<DataResponse<P2HReport>>(`/p2h/reports/${id}`);
      setDetail(report.data);
    } catch (loadError) {
      setDetail(null);
      setDetailError(
        loadError instanceof ApiError
          ? loadError.message
          : "Detail P2H tidak dapat dimuat.",
      );
    } finally {
      setDetailLoading(false);
    }
  };

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const term = query.trim();
    if (term.length > 100) {
      setQueryError("Kata kunci maksimal 100 karakter.");
      return;
    }
    setQueryError("");
    list.setFilters({ q: term || null });
  };

  const resetFilters = () => {
    setQuery("");
    list.setFilters({ q: null, state: null });
  };

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="flex flex-row flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold">Laporan P2H</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Snapshot laporan dan temuan P2H yang tersinkron dari Odoo.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <nav aria-label="Breadcrumb" className="hidden items-center gap-2 text-sm sm:flex">
              <Link href="/dashboard" className="text-muted-foreground hover:text-foreground">
                Dashboard
              </Link>
              <span className="text-muted-foreground" aria-hidden="true">/</span>
              <span aria-current="page">P2H</span>
            </nav>
            <Badge variant="secondary" className="h-6 px-2.5">
              {formatNumber(total)} laporan
            </Badge>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <form
            onSubmit={submitSearch}
            aria-label="Cari dan filter laporan P2H"
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
          >
            <Field className="min-w-0 flex-1" data-invalid={Boolean(queryError)}>
              <FieldLabel htmlFor="p2h-search">Cari laporan</FieldLabel>
              <div className="relative">
                <Search
                  className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                  aria-hidden="true"
                />
                <Input
                  id="p2h-search"
                  name="q"
                  type="search"
                  enterKeyHint="search"
                  value={query}
                  onChange={(event) => {
                    setQuery(event.target.value);
                    setQueryError("");
                  }}
                  placeholder="Nama P2H, unit, foreman, atau lokasi"
                  className="pl-9 text-sm leading-5"
                />
              </div>
              {queryError ? <FieldError>{queryError}</FieldError> : null}
            </Field>
            <Field className="sm:w-44">
              <FieldLabel htmlFor="p2h-state">State</FieldLabel>
              <NativeSelect
                id="p2h-state"
                value={state}
                onChange={(event) => list.setFilter("state", event.target.value || null)}
                className="w-full sm:w-44"
              >
                {Object.entries(stateLabels).map(([value, label]) => (
                  <NativeSelectOption key={value || "all"} value={value}>
                    {label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <div className="flex gap-2">
              <Button type="submit" variant="outline">
                <Filter aria-hidden="true" />
                Terapkan
              </Button>
              {hasFilters ? (
                <Button type="button" variant="ghost" onClick={resetFilters}>
                  <RotateCcw aria-hidden="true" />
                  Reset
                </Button>
              ) : null}
            </div>
          </form>
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal memuat laporan</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle>Daftar laporan</CardTitle>
          <CardDescription>Data read-only dari PostgreSQL.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {loading ? (
            <div className="space-y-2">
              {Array.from({ length: 6 }).map((_, index) => (
                <Skeleton key={index} className="h-14 w-full rounded-lg" />
              ))}
            </div>
          ) : items.length === 0 ? (
            <Empty className="border border-dashed">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ShieldCheck aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Tidak ada laporan</EmptyTitle>
                <EmptyDescription>
                  Belum ada laporan P2H yang cocok dengan filter ini.
                </EmptyDescription>
              </EmptyHeader>
              {hasFilters ? (
                <EmptyContent>
                  <Button type="button" variant="outline" onClick={resetFilters}>
                    Reset filter
                  </Button>
                </EmptyContent>
              ) : null}
            </Empty>
          ) : (
            <>
            <div className="hidden overflow-x-auto md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableSortHeader
                      label="Unit"
                      sortKey="unit_code"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableSortHeader
                      label="Laporan"
                      sortKey="name"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableSortHeader
                      label="Tanggal"
                      sortKey="report_date"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableSortHeader
                      label="State"
                      sortKey="state"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableSortHeader
                      label="Temuan"
                      sortKey="findings_count"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                      align="end"
                    />
                    <TableHead>Usulan</TableHead>
                    <TableHead className="text-right">Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>
                        <div className="font-medium">{item.unit_code || "-"}</div>
                        <div className="text-xs capitalize text-muted-foreground">
                          {item.unit_type.replaceAll("_", " ") || "-"}
                        </div>
                      </TableCell>
                      <TableCell className="max-w-56 truncate font-medium">
                        {item.name || `P2H-${item.odoo_report_id}`}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        {item.report_date || "-"}
                      </TableCell>
                      <TableCell>
                        <P2HStateBadge state={item.state} />
                      </TableCell>
                      <TableCell className="text-right">
                        <span className="font-medium">{formatNumber(item.findings_count)}</span>
                        <span className="ml-1 text-xs text-muted-foreground">
                          ({formatNumber(item.rusak_count)} rusak)
                        </span>
                      </TableCell>
                      <TableCell>
                        {item.candidate_id ? (
                          <Link
                            href={`/wo-candidates?candidate=${item.candidate_id}`}
                            className="inline-flex items-center gap-1.5 text-sm underline-offset-4 hover:underline"
                          >
                            <CandidateStatusBadge status={item.candidate_status} />
                          </Link>
                        ) : (
                          <span className="text-sm text-muted-foreground">Tidak ada</span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => void loadDetail(item.id)}
                        >
                          <Eye aria-hidden="true" />
                          Detail
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <div className="grid gap-2 md:hidden">
              {items.map((item) => (
                <div key={item.id} className="rounded-lg border border-border p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="font-medium">{item.unit_code || "-"}</span>
                    <P2HStateBadge state={item.state} />
                  </div>
                  <p className="mt-2 truncate text-sm font-medium">
                    {item.name || `P2H-${item.odoo_report_id}`}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {item.report_date || "-"} · {formatNumber(item.findings_count)} temuan
                    ({formatNumber(item.rusak_count)} rusak)
                  </p>
                  <div className="mt-3 flex items-center justify-between gap-2">
                    {item.candidate_id ? (
                      <Link
                        href={`/wo-candidates?candidate=${item.candidate_id}`}
                        className="text-sm underline-offset-4 hover:underline"
                      >
                        <CandidateStatusBadge status={item.candidate_status} />
                      </Link>
                    ) : (
                      <span className="text-xs text-muted-foreground">Tanpa usulan</span>
                    )}
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => void loadDetail(item.id)}
                    >
                      <Eye aria-hidden="true" />
                      Detail
                    </Button>
                  </div>
                </div>
              ))}
            </div>
            </>
          )}

          <TablePagination
            page={list.page}
            pageSize={list.pageSize}
            total={total}
            totalPages={totalPages}
            loading={loading}
            itemLabel="laporan"
            onPageChange={list.setPage}
            onPageSizeChange={list.setPageSize}
          />
        </CardContent>
      </Card>

      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent side="right" className="w-full gap-0 border-l-0 bg-background p-0 data-[side=right]:sm:max-w-2xl">
          <SheetHeader className="border-b border-border p-6 pr-16">
            <SheetTitle className="text-lg font-semibold">
              {detail?.name || "Detail P2H"}
            </SheetTitle>
            <SheetDescription>
              {detail
                ? `${detail.unit_code || "Unit"} · ${detail.report_date || "-"}`
                : "Snapshot laporan P2H."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto">
            {detailLoading ? (
              <div className="space-y-3 p-6">
                <Skeleton className="h-24 w-full rounded-lg" />
                <Skeleton className="h-40 w-full rounded-lg" />
              </div>
            ) : detailError ? (
              <div className="p-6">
                <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                  <CircleAlert aria-hidden="true" />
                  <AlertTitle>Gagal memuat detail</AlertTitle>
                  <AlertDescription>{detailError}</AlertDescription>
                </Alert>
              </div>
            ) : detail ? (
              <div className="space-y-6 p-6">
                <section className="grid gap-3 sm:grid-cols-3">
                  <Metric label="Temuan" value={formatNumber(detail.findings_count)} />
                  <Metric label="Rusak" value={formatNumber(detail.rusak_count)} />
                  <Metric label="Diperiksa" value={formatNumber(detail.checked_items)} />
                </section>

                <section className="grid gap-3 rounded-lg border border-border p-4 text-sm sm:grid-cols-2">
                  <Info label="State" value={detail.state || "-"} />
                  <Info label="Approval" value={detail.approval_status || "-"} />
                  <Info label="Foreman" value={detail.foreman_name || "-"} />
                  <Info label="Karyawan" value={detail.employee_name || "-"} />
                  <Info label="Shift" value={detail.shift || "-"} />
                  <Info label="Lokasi" value={detail.location || "-"} />
                  <Info
                    label="KM akhir"
                    value={detail.km_akhir == null ? "-" : formatNumber(detail.km_akhir)}
                  />
                  <Info
                    label="HM awal"
                    value={detail.hm_awal == null ? "-" : formatNumber(detail.hm_awal)}
                  />
                </section>

                {detail.candidate_id ? (
                  <Button
                    render={<Link href={`/wo-candidates?candidate=${detail.candidate_id}`} />}
                    variant="outline"
                  >
                    Buka usulan WO
                  </Button>
                ) : null}

                <section>
                  <h2 className="text-sm font-semibold">Temuan</h2>
                  <div className="mt-2 overflow-hidden rounded-lg border border-border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Komponen</TableHead>
                          <TableHead>Kategori</TableHead>
                          <TableHead>Kondisi</TableHead>
                          <TableHead>Catatan</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {detail.findings.length === 0 ? (
                          <TableRow>
                            <TableCell colSpan={4} className="py-8 text-center text-muted-foreground">
                              Tidak ada temuan rusak/tidak ada.
                            </TableCell>
                          </TableRow>
                        ) : (
                          detail.findings.map((finding) => (
                            <TableRow key={finding.id}>
                              <TableCell className="font-medium">{finding.component || "-"}</TableCell>
                              <TableCell className="max-w-44 whitespace-normal break-words text-muted-foreground">
                                {finding.category || "-"}
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={finding.condition === "rusak" ? "destructive" : "outline"}
                                  className="h-5 px-2"
                                >
                                  {finding.condition === "rusak" ? "Rusak" : "Tidak ada"}
                                </Badge>
                              </TableCell>
                              <TableCell className="max-w-44 whitespace-normal break-words text-muted-foreground">
                                {finding.remark || "-"}
                              </TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </section>

                <p className="text-xs text-muted-foreground">
                  Sinkron terakhir: {formatDateTimeWita(detail.synced_at)}
                </p>
              </div>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-0.5 break-words">{value || "-"}</p>
    </div>
  );
}
