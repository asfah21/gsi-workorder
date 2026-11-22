"use client";

import {
  CheckCircle2,
  CircleAlert,
  Eye,
  Filter,
  Loader2,
  RotateCcw,
  ShieldCheck,
  XCircle,
} from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { PageHeader } from "@/components/layout/page-header";
import { FilterBar } from "@/components/list/list-filter-bar";
import { SearchInput } from "@/components/list/list-search";
import { CandidateStatusBadge, PriorityBadge } from "@/components/p2h-badges";
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
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
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
import { Textarea } from "@/components/ui/textarea";
import { useListParams } from "@/hooks/use-list-params";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type {
  CandidateApproveResponse,
  CandidateBulkResponse,
  CandidateDetail,
  DataResponse,
  ListResponse,
  WOCandidate,
} from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const sortableColumns = [
  "created_at",
  "report_date",
  "unit_code",
  "findings_count",
  "suggested_priority",
  "status",
] as const;

const statusLabels: Record<string, string> = {
  pending: "Pending",
  approved: "Disetujui",
  skipped: "Ditolak",
  auto_created: "WO otomatis",
  archived: "Arsip",
  all: "Semua status",
};

const priorityLabels: Record<string, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

type BulkAction = "approve" | "skip";

export function CandidatesView() {
  const { user } = useSession();
  const router = useRouter();
  const searchParams = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const status = searchParams.get("status") || "pending";
  const unitType = searchParams.get("unit_type") ?? "";
  const candidateParam = searchParams.get("candidate") ?? "";
  const list = useListParams({
    defaultSort: "created_at",
    allowedSorts: sortableColumns,
    defaultOrder: "desc",
  });

  const [items, setItems] = useState<WOCandidate[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(q);
  const [queryError, setQueryError] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [selectionError, setSelectionError] = useState("");
  const [success, setSuccess] = useState("");

  const [detail, setDetail] = useState<CandidateDetail | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [actionError, setActionError] = useState("");
  const [acting, setActing] = useState(false);
  const [approvePriority, setApprovePriority] = useState("medium");
  const [approveNote, setApproveNote] = useState("");
  const [skipReason, setSkipReason] = useState("");

  const [bulkAction, setBulkAction] = useState<BulkAction | null>(null);
  const [bulkPriority, setBulkPriority] = useState("");
  const [bulkNote, setBulkNote] = useState("");
  const [bulkReason, setBulkReason] = useState("");
  const [bulkError, setBulkError] = useState("");
  const [bulkActing, setBulkActing] = useState(false);

  const canWrite = user.role === "admin" || user.role === "planner";
  const hasFilters = q.trim() !== "" || status !== "pending" || unitType !== "";
  const totalPages = Math.max(1, Math.ceil(total / list.pageSize));
  const pendingOnPage = items.filter((item) => item.status === "pending");
  const allOnPageSelected =
    pendingOnPage.length > 0 && pendingOnPage.every((item) => selected.has(item.id));

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
    if (q.trim()) {
      params.set("q", q.trim());
    }
    params.set("status", status);
    if (unitType) {
      params.set("unit_type", unitType);
    }

    try {
      const result = await proxyGet<ListResponse<WOCandidate>>(
        `/wo-candidates?${params.toString()}`,
      );
      setItems(result.data);
      setTotal(result.meta.total);
      setSelected((current) => {
        const available = new Set(
          result.data.filter((item) => item.status === "pending").map((item) => item.id),
        );
        return new Set(Array.from(current).filter((id) => available.has(id)));
      });
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Daftar usulan tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [list.order, list.page, list.pageSize, list.sort, q, status, unitType]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const loadDetail = useCallback(async (id: number) => {
    setDetailLoading(true);
    setDetailError("");
    setActionError("");
    setApproveNote("");
    setSkipReason("");
    try {
      const result = await proxyGet<DataResponse<CandidateDetail>>(`/wo-candidates/${id}`);
      setDetail(result.data);
      setApprovePriority(result.data.candidate.suggested_priority);
      setDetailOpen(true);
    } catch (loadError) {
      setDetailError(
        loadError instanceof ApiError
          ? loadError.message
          : "Detail usulan tidak dapat dimuat.",
      );
      setDetailOpen(true);
    } finally {
      setDetailLoading(false);
    }
  }, []);

  useEffect(() => {
    if (candidateParam) {
      const id = Number(candidateParam);
      if (Number.isInteger(id) && id > 0) {
        void loadDetail(id);
      }
    }
  }, [candidateParam, loadDetail]);

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
    list.setFilters({ q: null, status: null, unit_type: null });
  };

  const toggleSelected = (id: number) => {
    setSelectionError("");
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        if (next.size >= 50) {
          setSelectionError("Maksimal 50 usulan per aksi.");
          return current;
        }
        next.add(id);
      }
      return next;
    });
  };

  const togglePage = () => {
    setSelectionError("");
    if (allOnPageSelected) {
      setSelected((current) => {
        const next = new Set(current);
        for (const item of pendingOnPage) {
          next.delete(item.id);
        }
        return next;
      });
      return;
    }
    setSelected((current) => {
      const next = new Set(current);
      for (const item of pendingOnPage) {
        if (next.size >= 50) {
          break;
        }
        next.add(item.id);
      }
      return next;
    });
  };

  const approve = async () => {
    if (!detail) return;
    setActing(true);
    setActionError("");
    try {
      const result = await proxyPost<DataResponse<CandidateApproveResponse>>(
        `/wo-candidates/${detail.candidate.id}/approve`,
        {
          priority: approvePriority,
          note: approveNote,
        },
      );
      setSuccess(`Usulan disetujui. WO ${result.data.work_order.code} dibuat.`);
      await loadDetail(detail.candidate.id);
      await loadList();
    } catch (actionFailure) {
      setActionError(
        actionFailure instanceof ApiError
          ? actionFailure.message
          : "Usulan tidak dapat disetujui.",
      );
    } finally {
      setActing(false);
    }
  };

  const skip = async () => {
    if (!detail) return;
    setActing(true);
    setActionError("");
    try {
      await proxyPost(`/wo-candidates/${detail.candidate.id}/skip`, {
        reason: skipReason,
      });
      setSuccess("Usulan ditolak.");
      await loadDetail(detail.candidate.id);
      await loadList();
    } catch (skipFailure) {
      setActionError(
        skipFailure instanceof ApiError
          ? skipFailure.message
          : "Usulan tidak dapat ditolak.",
      );
    } finally {
      setActing(false);
    }
  };

  const runBulk = async () => {
    if (!bulkAction) return;
    setBulkActing(true);
    setBulkError("");
    try {
      const result = await proxyPost<DataResponse<CandidateBulkResponse>>(
        "/wo-candidates/bulk",
        {
          ids: Array.from(selected),
          action: bulkAction,
          priority: bulkAction === "approve" ? bulkPriority || undefined : undefined,
          note: bulkAction === "approve" ? bulkNote : undefined,
          reason: bulkAction === "skip" ? bulkReason : undefined,
        },
      );
      const count =
        bulkAction === "approve"
          ? result.data.approved.length
          : result.data.skipped.length;
      setSuccess(
        bulkAction === "approve"
          ? `${formatNumber(count)} usulan disetujui dan WO dibuat.`
          : `${formatNumber(count)} usulan ditolak.`,
      );
      setSelected(new Set());
      setBulkAction(null);
      setBulkPriority("");
      setBulkNote("");
      setBulkReason("");
      await loadList();
    } catch (bulkFailure) {
      setBulkError(
        bulkFailure instanceof ApiError
          ? bulkFailure.message
          : "Aksi bulk tidak dapat diproses.",
      );
    } finally {
      setBulkActing(false);
    }
  };

  const selectedCount = selected.size;
  const detailCandidate = detail?.candidate;
  const pendingDetail = detailCandidate?.status === "pending";

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Usulan WO"
        description="Temuan P2H yang menunggu keputusan planner. Approve akan membuat work order."
        breadcrumb={
          <nav aria-label="Breadcrumb" className="flex items-center gap-2">
            <Link href="/dashboard" className="hover:text-foreground">
              Dashboard
            </Link>
            <span aria-hidden="true">/</span>
            <span aria-current="page">Usulan WO</span>
          </nav>
        }
        actions={
          <Badge variant="secondary" className="h-6 px-2.5">
            {formatNumber(total)} {status === "pending" ? "pending" : "usulan"}
          </Badge>
        }
      />

      <Card>
        <CardContent>
          <FilterBar onSubmit={submitSearch} aria-label="Cari dan filter usulan WO">
            <Field className="min-w-0 flex-1" data-invalid={Boolean(queryError)}>
              <FieldLabel htmlFor="candidate-search">Cari usulan</FieldLabel>
              <SearchInput
                id="candidate-search"
                name="q"
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setQueryError("");
                }}
                placeholder="Nama P2H, unit, foreman, atau lokasi"
                aria-invalid={Boolean(queryError)}
              />
              {queryError ? <FieldError>{queryError}</FieldError> : null}
            </Field>
            <Field className="md:w-44">
              <FieldLabel htmlFor="candidate-status">Status</FieldLabel>
              <NativeSelect
                id="candidate-status"
                value={status}
                onChange={(event) =>
                  list.setFilters({
                    status: event.target.value === "pending" ? null : event.target.value,
                  })
                }
                className="w-full"
              >
                {Object.entries(statusLabels).map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>
                    {label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field className="md:w-44">
              <FieldLabel htmlFor="candidate-unit-type">Tipe unit</FieldLabel>
              <NativeSelect
                id="candidate-unit-type"
                value={unitType}
                onChange={(event) =>
                  list.setFilter("unit_type", event.target.value || null)
                }
                className="w-full"
              >
                <NativeSelectOption value="">Semua tipe</NativeSelectOption>
                <NativeSelectOption value="light_vehicle">Light vehicle</NativeSelectOption>
                <NativeSelectOption value="heavy_vehicle">Heavy vehicle</NativeSelectOption>
              </NativeSelect>
            </Field>
            <div className="flex gap-2 md:ml-auto">
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
          </FilterBar>
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal memuat usulan</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      {success ? (
        <Alert className="border-emerald-500/30 bg-emerald-500/5">
          <CheckCircle2 aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardHeader className="gap-2">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <CardTitle>Daftar usulan</CardTitle>
              <CardDescription>
                Pilih maksimal 50 usulan pending untuk aksi bulk.
              </CardDescription>
            </div>
            {selectedCount > 0 ? (
              <div className="flex flex-wrap items-center gap-2 rounded-lg border border-border bg-muted/40 p-2">
                <Badge variant="secondary" className="h-6 px-2.5">
                  {selectedCount} dipilih
                </Badge>
                <Button type="button" size="sm" onClick={() => setBulkAction("approve")}>
                  <CheckCircle2 aria-hidden="true" />
                  Setujui
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => setBulkAction("skip")}
                >
                  <XCircle aria-hidden="true" />
                  Tolak
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setSelected(new Set());
                    setSelectionError("");
                  }}
                >
                  Bersihkan
                </Button>
              </div>
            ) : null}
          </div>
          {selectionError ? <FieldError>{selectionError}</FieldError> : null}
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
                <EmptyTitle>Tidak ada usulan</EmptyTitle>
                <EmptyDescription>
                  Belum ada temuan P2H yang cocok dengan filter ini.
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
                    <TableHead className="w-10">
                      <input
                        type="checkbox"
                        aria-label="Pilih semua usulan pending di halaman ini"
                        checked={allOnPageSelected}
                        onChange={togglePage}
                        disabled={pendingOnPage.length === 0}
                        className="size-4 rounded border-input accent-primary"
                      />
                    </TableHead>
                    <TableSortHeader
                      label="Unit"
                      sortKey="unit_code"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableSortHeader
                      label="Laporan"
                      sortKey="report_date"
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
                    <TableSortHeader
                      label="Prioritas"
                      sortKey="suggested_priority"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableSortHeader
                      label="Status"
                      sortKey="status"
                      currentSort={list.sort}
                      currentOrder={list.order}
                      onSort={list.toggleSort}
                    />
                    <TableHead className="hidden md:table-cell">Dibuat</TableHead>
                    <TableHead className="text-right">Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => {
                    const pending = item.status === "pending";
                    return (
                      <TableRow key={item.id} data-state={selected.has(item.id) ? "selected" : undefined}>
                        <TableCell>
                          <input
                            type="checkbox"
                            aria-label={`Pilih usulan ${item.unit_code || item.id}`}
                            checked={selected.has(item.id)}
                            onChange={() => toggleSelected(item.id)}
                            disabled={!pending}
                            className="size-4 rounded border-input accent-primary"
                          />
                        </TableCell>
                        <TableCell>
                          <div className="font-medium">{item.unit_code || "-"}</div>
                          <div className="text-xs capitalize text-muted-foreground">
                            {item.unit_type.replaceAll("_", " ") || "-"}
                          </div>
                        </TableCell>
                        <TableCell>
                          <button
                            type="button"
                            onClick={() => void loadDetail(item.id)}
                            className="max-w-56 truncate text-left font-medium underline-offset-4 hover:underline"
                          >
                            {item.report_name || `P2H-${item.p2h_report_id}`}
                          </button>
                          <div className="text-xs text-muted-foreground">
                            {item.report_date || "-"}
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <span className="font-medium">{formatNumber(item.findings_count)}</span>
                          <span className="ml-1 text-xs text-muted-foreground">
                            ({formatNumber(item.rusak_count)} rusak)
                          </span>
                        </TableCell>
                        <TableCell>
                          <PriorityBadge priority={item.suggested_priority} />
                        </TableCell>
                        <TableCell>
                          <CandidateStatusBadge status={item.status} />
                        </TableCell>
                        <TableCell className="hidden whitespace-nowrap text-muted-foreground md:table-cell">
                          {formatDateTimeWita(item.created_at)}
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
                    );
                  })}
                </TableBody>
              </Table>
            </div>
            <div className="grid gap-2 md:hidden">
              {items.map((item) => {
                const pending = item.status === "pending";
                return (
                  <div
                    key={item.id}
                    className="rounded-lg border border-border p-3"
                  >
                    <div className="flex items-start gap-3">
                      <input
                        type="checkbox"
                        aria-label={`Pilih usulan ${item.unit_code || item.id}`}
                        checked={selected.has(item.id)}
                        onChange={() => toggleSelected(item.id)}
                        disabled={!pending}
                        className="mt-1 size-4 rounded border-input accent-primary"
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-medium">{item.unit_code || "-"}</span>
                          <CandidateStatusBadge status={item.status} />
                          <PriorityBadge priority={item.suggested_priority} />
                        </div>
                        <button
                          type="button"
                          onClick={() => void loadDetail(item.id)}
                          className="mt-2 block max-w-full truncate text-left text-sm font-medium underline-offset-4 hover:underline"
                        >
                          {item.report_name || `P2H-${item.p2h_report_id}`}
                        </button>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {item.report_date || "-"} · {formatNumber(item.findings_count)} temuan
                          ({formatNumber(item.rusak_count)} rusak)
                        </p>
                        <div className="mt-3 flex items-center justify-between gap-2">
                          <span className="text-xs text-muted-foreground">
                            {formatDateTimeWita(item.created_at)}
                          </span>
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
                    </div>
                  </div>
                );
              })}
            </div>
            </>
          )}

          <TablePagination
            page={list.page}
            pageSize={list.pageSize}
            total={total}
            totalPages={totalPages}
            loading={loading}
            itemLabel="usulan"
            onPageChange={list.setPage}
            onPageSizeChange={list.setPageSize}
          />
        </CardContent>
      </Card>

      <Sheet
        open={detailOpen}
        onOpenChange={(next) => {
          setDetailOpen(next);
          if (!next) {
            setDetailError("");
            setActionError("");
            if (candidateParam) {
              const params = new URLSearchParams(searchParams.toString());
              params.delete("candidate");
              const query = params.toString();
              router.replace(query ? `/wo-candidates?${query}` : "/wo-candidates", {
                scroll: false,
              });
            }
          }
        }}
      >
        <SheetContent
          side="right"
          className="w-full gap-0 border-l-0 bg-background p-0 data-[side=right]:sm:max-w-2xl"
        >
          <SheetHeader className="border-b border-border p-6 pr-16">
            <div className="flex flex-wrap items-center gap-2">
              {detailCandidate ? (
                <>
                  <SheetTitle className="text-lg font-semibold">
                    {detailCandidate.report_name || `P2H-${detailCandidate.p2h_report_id}`}
                  </SheetTitle>
                  <CandidateStatusBadge status={detailCandidate.status} />
                </>
              ) : (
                <SheetTitle className="text-lg font-semibold">Detail usulan</SheetTitle>
              )}
            </div>
            <SheetDescription>
              {detailCandidate
                ? `${detailCandidate.unit_code || "Unit"} · ${detailCandidate.report_date || "-"}`
                : "Muat detail usulan P2H."}
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
                  <Metric label="Temuan" value={formatNumber(detail.report.findings_count)} />
                  <Metric label="Rusak" value={formatNumber(detail.report.rusak_count)} />
                  <Metric label="Item diperiksa" value={formatNumber(detail.report.checked_items)} />
                </section>

                <section className="grid gap-3 rounded-lg border border-border p-4 text-sm sm:grid-cols-2">
                  <Info label="P2H" value={detail.report.name || "-"} />
                  <Info label="State P2H" value={detail.report.state || "-"} />
                  <Info label="Foreman" value={detail.report.foreman_name || "-"} />
                  <Info label="Karyawan" value={detail.report.employee_name || "-"} />
                  <Info label="Shift" value={detail.report.shift || "-"} />
                  <Info label="Lokasi" value={detail.report.location || "-"} />
                  <Info
                    label="KM akhir"
                    value={
                      detail.report.km_akhir == null
                        ? "-"
                        : formatNumber(detail.report.km_akhir)
                    }
                  />
                  <Info
                    label="HM awal"
                    value={
                      detail.report.hm_awal == null
                        ? "-"
                        : formatNumber(detail.report.hm_awal)
                    }
                  />
                </section>

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
                        {detail.report.findings.map((finding) => (
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
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </section>

                {detail.work_order ? (
                  <section className="rounded-lg border border-border p-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                          Work order
                        </p>
                        <p className="mt-1 font-semibold">{detail.work_order.code}</p>
                      </div>
                      <Badge variant="secondary" className="capitalize">
                        {detail.work_order.status.replaceAll("_", " ")}
                      </Badge>
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                      {detail.work_order.title}
                    </p>
                  </section>
                ) : null}

                {detailCandidate && !pendingDetail ? (
                  <section className="rounded-lg border border-border p-4 text-sm">
                    <h2 className="font-semibold">Keputusan</h2>
                    <div className="mt-3 grid gap-3 sm:grid-cols-2">
                      <Info label="Diputuskan oleh" value={detailCandidate.decided_by_name || "-"} />
                      <Info
                        label="Waktu"
                        value={
                          detailCandidate.decided_at
                            ? formatDateTimeWita(detailCandidate.decided_at)
                            : "-"
                        }
                      />
                      {detailCandidate.skip_reason ? (
                        <Info label="Alasan ditolak" value={detailCandidate.skip_reason} />
                      ) : null}
                      {detailCandidate.decision_note ? (
                        <Info label="Catatan" value={detailCandidate.decision_note} />
                      ) : null}
                    </div>
                  </section>
                ) : null}

                {actionError ? (
                  <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                    <CircleAlert aria-hidden="true" />
                    <AlertTitle>Aksi gagal</AlertTitle>
                    <AlertDescription>{actionError}</AlertDescription>
                  </Alert>
                ) : null}

                {canWrite && pendingDetail ? (
                  <section className="grid gap-6 border-t border-border pt-6 xl:grid-cols-2">
                    <div className="space-y-3 rounded-lg border border-border p-4">
                      <div>
                        <h2 className="text-sm font-semibold">Setujui &amp; buat WO</h2>
                        <p className="mt-1 text-sm text-muted-foreground">
                          WO akan dibuat dengan temuan P2H terlampir.
                        </p>
                      </div>
                      <Field>
                        <FieldLabel htmlFor="approve-priority">Prioritas WO</FieldLabel>
                        <NativeSelect
                          id="approve-priority"
                          value={approvePriority}
                          onChange={(event) => setApprovePriority(event.target.value)}
                          className="w-full"
                        >
                          {Object.entries(priorityLabels).map(([value, label]) => (
                            <NativeSelectOption key={value} value={value}>
                              {label}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                      </Field>
                      <Field>
                        <FieldLabel htmlFor="approve-note">Catatan (opsional)</FieldLabel>
                        <Textarea
                          id="approve-note"
                          value={approveNote}
                          onChange={(event) => setApproveNote(event.target.value)}
                          maxLength={1000}
                          placeholder="Instruksi untuk mekanik..."
                        />
                      </Field>
                      <Button
                        type="button"
                        onClick={() => void approve()}
                        disabled={acting}
                        className="w-full"
                      >
                        {acting ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                        Setujui &amp; buat WO
                      </Button>
                    </div>

                    <div className="space-y-3 rounded-lg border border-border p-4">
                      <div>
                        <h2 className="text-sm font-semibold">Tolak usulan</h2>
                        <p className="mt-1 text-sm text-muted-foreground">
                          Alasan wajib, minimal 5 karakter.
                        </p>
                      </div>
                      <Field>
                        <FieldLabel htmlFor="skip-reason">Alasan</FieldLabel>
                        <Textarea
                          id="skip-reason"
                          value={skipReason}
                          onChange={(event) => setSkipReason(event.target.value)}
                          maxLength={1000}
                          placeholder="Contoh: sudah ditangani di WO lain"
                        />
                      </Field>
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => void skip()}
                        disabled={acting || skipReason.trim().length < 5}
                        className="w-full"
                      >
                        {acting ? <Loader2 className="animate-spin" aria-hidden="true" /> : <XCircle aria-hidden="true" />}
                        Tolak usulan
                      </Button>
                    </div>
                  </section>
                ) : null}
              </div>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

      <Sheet
        open={bulkAction !== null}
        onOpenChange={(next) => {
          if (!next) {
            setBulkAction(null);
            setBulkError("");
          }
        }}
      >
        <SheetContent side="right" className="w-full gap-0 border-l-0 bg-background p-0 data-[side=right]:sm:max-w-md">
          <SheetHeader className="border-b border-border p-6 pr-16">
            <SheetTitle className="text-lg font-semibold">
              {bulkAction === "approve" ? "Setujui usulan terpilih" : "Tolak usulan terpilih"}
            </SheetTitle>
            <SheetDescription>
              {selectedCount} usulan akan diproses dalam satu transaksi.
            </SheetDescription>
          </SheetHeader>
          <div className="space-y-4 p-6">
            {bulkAction === "approve" ? (
              <>
                <Field>
                  <FieldLabel htmlFor="bulk-priority">Prioritas</FieldLabel>
                  <NativeSelect
                    id="bulk-priority"
                    value={bulkPriority}
                    onChange={(event) => setBulkPriority(event.target.value)}
                    className="w-full"
                  >
                    <NativeSelectOption value="">Sesuai usulan</NativeSelectOption>
                    {Object.entries(priorityLabels).map(([value, label]) => (
                      <NativeSelectOption key={value} value={value}>
                        {label}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="bulk-note">Catatan (opsional)</FieldLabel>
                  <Textarea
                    id="bulk-note"
                    value={bulkNote}
                    onChange={(event) => setBulkNote(event.target.value)}
                    maxLength={1000}
                  />
                </Field>
              </>
            ) : (
              <Field>
                <FieldLabel htmlFor="bulk-reason">Alasan penolakan</FieldLabel>
                <Textarea
                  id="bulk-reason"
                  value={bulkReason}
                  onChange={(event) => setBulkReason(event.target.value)}
                  maxLength={1000}
                  placeholder="Alasan minimal 5 karakter"
                />
              </Field>
            )}

            {bulkError ? (
              <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                <CircleAlert aria-hidden="true" />
                <AlertTitle>Aksi bulk gagal</AlertTitle>
                <AlertDescription>{bulkError}</AlertDescription>
              </Alert>
            ) : null}
          </div>
          <SheetFooter className="border-t border-border p-6">
            <Button
              type="button"
              onClick={() => void runBulk()}
              disabled={
                bulkActing ||
                selectedCount === 0 ||
                (bulkAction === "skip" && bulkReason.trim().length < 5)
              }
              className="w-full"
            >
              {bulkActing ? (
                <Loader2 className="animate-spin" aria-hidden="true" />
              ) : bulkAction === "approve" ? (
                <CheckCircle2 aria-hidden="true" />
              ) : (
                <XCircle aria-hidden="true" />
              )}
              {bulkAction === "approve" ? "Setujui semua" : "Tolak semua"}
            </Button>
          </SheetFooter>
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
