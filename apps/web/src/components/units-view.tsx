"use client";

import {
  ChevronRight,
  CircleAlert,
  Filter,
  Loader2,
  Plus,
  RotateCcw,
  Search,
  Truck,
} from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { UnitStatusBadge } from "@/components/status-badges";
import { TablePagination } from "@/components/table-pagination";
import { TableSortHeader } from "@/components/table-sort-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Skeleton } from "@/components/ui/skeleton";
import {
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useListParams } from "@/hooks/use-list-params";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { ListResponse, Unit } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const filterLabels: Record<string, string> = {
  active: "Aktif",
  inactive: "Nonaktif",
  all: "Semua",
};

const sortableColumns = [
  "code",
  "model",
  "site_location",
  "current_hm",
  "current_km",
  "is_active",
] as const;

export function UnitsView() {
  const { user } = useSession();
  const searchParams = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const rawActive = searchParams.get("is_active");
  const activeFilter =
    rawActive === "false" ? "inactive" : rawActive === "all" ? "all" : "active";
  const list = useListParams({
    defaultSort: "code",
    allowedSorts: sortableColumns,
  });

  const [units, setUnits] = useState<Unit[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(q);
  const [queryError, setQueryError] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const canWrite = user.role === "admin" || user.role === "planner";
  const hasFilters = q.trim() !== "" || activeFilter !== "active";
  const totalPages = Math.max(1, Math.ceil(total / list.pageSize));

  useEffect(() => {
    setQuery(q);
    setQueryError("");
  }, [q]);

  useEffect(() => {
    let cancelled = false;
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
    if (activeFilter === "active") {
      params.set("is_active", "true");
    } else if (activeFilter === "inactive") {
      params.set("is_active", "false");
    }

    proxyGet<ListResponse<Unit>>(`/units?${params.toString()}`)
      .then((result) => {
        if (!cancelled) {
          setUnits(result.data);
          setTotal(result.meta.total);
        }
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(
            loadError instanceof ApiError
              ? loadError.message
              : "Daftar unit tidak dapat dimuat.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [activeFilter, list.order, list.page, list.pageSize, list.sort, q]);

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const term = query.trim();
    if (term.length > 100) {
      setQueryError("Kata kunci maksimal 100 karakter.");
      requestAnimationFrame(() => document.getElementById("unit-search")?.focus());
      return;
    }
    setQueryError("");
    list.setFilters({ q: term || null });
  };

  const resetFilters = () => {
    setQuery("");
    setQueryError("");
    list.setFilters({ q: null, is_active: null });
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">Unit</h1>
            <Badge variant="secondary" className="h-6 px-2.5">
              {formatNumber(total)} unit
            </Badge>
          </div>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Master unit, model, dan posisi HM/KM terakhir.
          </p>
        </div>
        {canWrite ? (
          <Link
            href="/units/new"
            className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            <Plus className="size-4" aria-hidden="true" />
            Tambah unit
          </Link>
        ) : null}
      </div>

      <search>
        <form
          onSubmit={submitSearch}
          className="rounded-xl border border-border bg-background p-4"
          aria-label="Cari dan filter unit"
        >
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <Field className="min-w-0 flex-1" data-invalid={Boolean(queryError)}>
              <FieldLabel htmlFor="unit-search">Cari unit</FieldLabel>
              <div className="relative">
                <Search
                  className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                  aria-hidden="true"
                />
                <Input
                  id="unit-search"
                  name="q"
                  type="search"
                  enterKeyHint="search"
                  autoCapitalize="none"
                  spellCheck={false}
                  value={query}
                  onChange={(event) => {
                    setQuery(event.target.value);
                    setQueryError("");
                  }}
                  aria-invalid={Boolean(queryError)}
                  aria-describedby={queryError ? "unit-search-error" : undefined}
                  placeholder="Kode, nopol, lokasi, atau merk"
                  className="pl-10"
                />
              </div>
              {queryError ? (
                <FieldError id="unit-search-error">{queryError}</FieldError>
              ) : null}
            </Field>
            <Field className="sm:w-40 sm:shrink-0">
              <FieldLabel htmlFor="unit-active-filter">Status</FieldLabel>
              <NativeSelect
                id="unit-active-filter"
                name="is_active"
                value={activeFilter}
                onChange={(event) => {
                  list.setFilters({
                    is_active:
                      event.target.value === "active" ? null : event.target.value,
                  });
                }}
                className="w-full sm:w-40"
              >
                <NativeSelectOption value="active">Aktif</NativeSelectOption>
                <NativeSelectOption value="inactive">Nonaktif</NativeSelectOption>
                <NativeSelectOption value="all">Semua</NativeSelectOption>
              </NativeSelect>
            </Field>
            <div className="flex gap-2">
              <Button type="submit" variant="outline" className="h-11 flex-1 px-4 sm:flex-none">
                <Filter aria-hidden="true" />
                Terapkan
              </Button>
              {hasFilters ? (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={resetFilters}
                  className="h-11 flex-1 px-3 sm:flex-none"
                >
                  <RotateCcw aria-hidden="true" />
                  Reset
                </Button>
              ) : null}
            </div>
          </div>

          {hasFilters ? (
            <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <span>Filter aktif:</span>
              {q.trim() ? (
                <Badge variant="outline" className="h-6 px-2.5">
                  Cari: {q.trim()}
                </Badge>
              ) : null}
              {activeFilter !== "active" ? (
                <Badge variant="outline" className="h-6 px-2.5">
                  Status: {filterLabels[activeFilter]}
                </Badge>
              ) : null}
            </div>
          ) : null}
        </form>
      </search>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Daftar unit tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {error && units.length === 0 ? null : (
        <div className="overflow-hidden rounded-xl border border-border bg-background">
          {loading && units.length === 0 ? (
          <div className="space-y-3 p-4" role="status" aria-label="Memuat unit">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-12 w-full" />
            ))}
          </div>
        ) : error && units.length === 0 ? null : units.length === 0 ? (
          <div className="p-4">
            <Empty className="border-0 py-10">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Truck aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada unit.</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "Tidak ada unit yang cocok dengan filter. Coba ubah kata kunci atau status."
                    : "Tambahkan unit pertama untuk mulai mencatat HM/KM."}
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                {hasFilters ? (
                  <Button type="button" variant="outline" onClick={resetFilters} className="h-11 px-4">
                    <RotateCcw aria-hidden="true" />
                    Reset filter
                  </Button>
                ) : null}
                {canWrite ? (
                  <Link
                    href="/units/new"
                    className="inline-flex min-h-11 items-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    <Plus className="size-4" aria-hidden="true" />
                    Tambah unit
                  </Link>
                ) : null}
              </EmptyContent>
            </Empty>
          </div>
        ) : (
          <div
            className="responsive-table-wrapper"
            tabIndex={0}
            role="region"
            aria-labelledby="unit-table-caption"
            aria-busy={loading}
          >
            <table className="responsive-table">
              <TableCaption
                id="unit-table-caption"
                className="border-b border-border px-4 py-3 text-left text-sm text-muted-foreground"
              >
                {loading ? (
                  <span className="inline-flex items-center gap-2">
                    <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                    Memuat unit...
                  </span>
                ) : (
                  `${formatNumber(total)} unit`
                )}
              </TableCaption>
              <TableHeader>
                <TableRow>
                  <TableSortHeader
                    label="Unit"
                    sortKey="code"
                    currentSort={list.sort}
                    currentOrder={list.order}
                    onSort={list.toggleSort}
                  />
                  <TableSortHeader
                    label="Model"
                    sortKey="model"
                    currentSort={list.sort}
                    currentOrder={list.order}
                    onSort={list.toggleSort}
                  />
                  <TableSortHeader
                    label="Lokasi"
                    sortKey="site_location"
                    currentSort={list.sort}
                    currentOrder={list.order}
                    onSort={list.toggleSort}
                  />
                  <TableSortHeader
                    label="HM"
                    sortKey="current_hm"
                    currentSort={list.sort}
                    currentOrder={list.order}
                    onSort={list.toggleSort}
                    align="end"
                  />
                  <TableSortHeader
                    label="KM"
                    sortKey="current_km"
                    currentSort={list.sort}
                    currentOrder={list.order}
                    onSort={list.toggleSort}
                    align="end"
                  />
                  <TableSortHeader
                    label="Status"
                    sortKey="is_active"
                    currentSort={list.sort}
                    currentOrder={list.order}
                    onSort={list.toggleSort}
                  />
                  <TableHead scope="col" className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {units.map((unit) => (
                  <TableRow key={unit.id}>
                    <TableHead scope="row" className="unit-row-heading whitespace-normal">
                      <Link
                        href={`/units/${unit.id}`}
                        className="font-semibold underline-offset-4 hover:underline"
                      >
                        {unit.code}
                      </Link>
                      <span className="block text-xs font-normal text-muted-foreground">
                        {unit.unit_type || "Tanpa tipe"}
                      </span>
                    </TableHead>
                    <TableCell data-label="Model">{modelLabel(unit)}</TableCell>
                    <TableCell data-label="Lokasi">{unit.site_location || "-"}</TableCell>
                    <TableCell data-label="HM" className="text-right font-mono tabular-nums">
                      {unit.hm_updated_at ? formatNumber(unit.current_hm) : "—"}
                    </TableCell>
                    <TableCell data-label="KM" className="text-right font-mono tabular-nums">
                      {unit.km_updated_at ? formatNumber(unit.current_km) : "—"}
                    </TableCell>
                    <TableCell data-label="Status">
                      <UnitStatusBadge active={unit.is_active} breakdown={unit.is_breakdown} />
                      <span className="mt-1 block text-xs text-muted-foreground">
                        {unit.hm_updated_at
                          ? `HM ${formatDateTimeWita(unit.hm_updated_at)}`
                          : "Belum ada reading"}
                      </span>
                    </TableCell>
                    <TableCell data-label="Aksi" className="text-right">
                      <Link
                        href={`/units/${unit.id}`}
                        className="inline-flex min-h-10 items-center gap-1.5 rounded-lg border border-border px-3 text-sm font-medium transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                      >
                        Detail
                        <ChevronRight className="size-4" aria-hidden="true" />
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </table>
          </div>
        )}
        </div>
      )}

      {total > 0 ? (
        <TablePagination
          page={list.page}
          pageSize={list.pageSize}
          total={total}
          totalPages={totalPages}
          loading={loading}
          itemLabel="unit"
          onPageChange={list.setPage}
          onPageSizeChange={list.setPageSize}
        />
      ) : null}
    </div>
  );
}

function modelLabel(unit: Unit): string {
  if (unit.unit_model) {
    return `${unit.unit_model.manufacturer} ${unit.unit_model.model}`;
  }
  return unit.merk || "-";
}
