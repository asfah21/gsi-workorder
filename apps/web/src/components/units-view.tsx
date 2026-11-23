"use client";

import {
  ChevronRight,
  CircleAlert,
  CircleCheck,
  Download,
  Filter,
  Loader2,
  Plus,
  RotateCcw,
  Truck,
  Upload,
} from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { UnitCreateSheet } from "@/components/unit-create-sheet";
import { UnitImportSheet } from "@/components/unit-import-sheet";
import { PageHeader } from "@/components/layout/page-header";
import { FilterBar } from "@/components/list/list-filter-bar";
import { SearchInput } from "@/components/list/list-search";
import { UnitStatusBadge } from "@/components/status-badges";
import { TablePagination } from "@/components/table-pagination";
import { TableSortHeader } from "@/components/table-sort-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
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
import { downloadBlob } from "@/lib/download";
import type { ListResponse, Unit, UnitImportResult } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const filterLabels: Record<string, string> = {
  active: "Aktif",
  inactive: "Nonaktif",
  breakdown: "Breakdown",
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
  const router = useRouter();
  const searchParams = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const rawActive = searchParams.get("is_active");
  const rawBreakdown = searchParams.get("is_breakdown");
  const activeFilter =
    rawBreakdown === "true"
      ? "breakdown"
      : rawActive === "false"
        ? "inactive"
        : rawActive === "all"
          ? "all"
          : "active";
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
  const [refreshKey, setRefreshKey] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);
  const [createKey, setCreateKey] = useState(0);
  const [importOpen, setImportOpen] = useState(false);
  const [importKey, setImportKey] = useState(0);
  const [importResult, setImportResult] = useState<UnitImportResult | null>(null);
  const [exporting, setExporting] = useState(false);
  const [actionError, setActionError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  const canWrite = user.role === "admin" || user.role === "planner";
  const hasFilters = q.trim() !== "" || activeFilter !== "active";
  const totalPages = Math.max(1, Math.ceil(total / list.pageSize));

  useEffect(() => {
    setQuery(q);
    setQueryError("");
  }, [q]);

  useEffect(() => {
    if (searchParams.get("create") !== "1") {
      return;
    }
    setCreateKey((current) => current + 1);
    setCreateOpen(true);
    const nextParams = new URLSearchParams(searchParams.toString());
    nextParams.delete("create");
    const suffix = nextParams.toString();
    router.replace(`/units${suffix ? `?${suffix}` : ""}`, { scroll: false });
  }, [router, searchParams]);

  const buildListParams = useCallback((includePagination: boolean) => {
    const params = new URLSearchParams({
      sort: list.sort,
      order: list.order,
    });
    if (q.trim()) {
      params.set("q", q.trim());
    }
    if (activeFilter === "breakdown") {
      params.set("is_active", "true");
      params.set("is_breakdown", "true");
    } else if (activeFilter === "active") {
      params.set("is_active", "true");
    } else if (activeFilter === "inactive") {
      params.set("is_active", "false");
    }
    if (includePagination) {
      params.set("page", String(list.page));
      params.set("limit", String(list.pageSize));
    }
    return params;
  }, [activeFilter, list.order, list.page, list.pageSize, list.sort, q]);

  const refresh = () => {
    setRefreshKey((current) => current + 1);
  };

  const openCreate = () => {
    setCreateKey((current) => current + 1);
    setSuccessMessage("");
    setActionError("");
    setCreateOpen(true);
  };

  const openImport = () => {
    setImportKey((current) => current + 1);
    setSuccessMessage("");
    setActionError("");
    setImportOpen(true);
  };

  const exportXlsx = async () => {
    setExporting(true);
    setActionError("");
    setSuccessMessage("");
    try {
      const response = await fetch(
        `/api/proxy/units/export?${buildListParams(false).toString()}`,
        {
          credentials: "same-origin",
          cache: "no-store",
          headers: {
            Accept:
              "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
          },
        },
      );
      if (!response.ok) {
        throw new Error("XLSX tidak dapat dibuat.");
      }
      downloadBlob(await response.blob(), `unit-${new Date().toISOString().slice(0, 10)}.xlsx`);
      setSuccessMessage("Data unit berhasil diekspor.");
    } catch (exportError) {
      setActionError(
        exportError instanceof Error
          ? exportError.message
          : "XLSX tidak dapat dibuat.",
      );
    } finally {
      setExporting(false);
    }
  };

  const handleImported = (result: UnitImportResult) => {
    setImportResult(result);
    if (result.failed > 0) {
      setActionError(
        `Impor dibatalkan: ${result.failed} baris tidak valid. Tidak ada unit yang disimpan.`,
      );
      setSuccessMessage("");
    } else {
      setActionError("");
      setSuccessMessage(
        `Impor selesai: ${result.created} unit dibuat, ${result.skipped} baris dilewati.`,
      );
    }
    refresh();
  };

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");

    const params = buildListParams(true);

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
  }, [buildListParams, refreshKey]);

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
    list.setFilters({ q: null, is_active: null, is_breakdown: null });
  };

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Unit"
        description="Master unit, model, dan posisi HM/KM terakhir."
        actions={
          <>
            <Badge variant="secondary" className="h-6 px-2.5">
              {formatNumber(total)} unit
            </Badge>
            <Button
              type="button"
              variant="outline"
              onClick={() => void exportXlsx()}
              disabled={exporting}
            >
              {exporting ? (
                <Loader2 className="animate-spin" aria-hidden="true" />
              ) : (
                <Download aria-hidden="true" />
              )}
              Ekspor XLSX
            </Button>
            {canWrite ? (
              <>
                <Button type="button" variant="outline" onClick={openImport}>
                  <Upload aria-hidden="true" />
                  Impor XLSX
                </Button>
                <Button type="button" onClick={openCreate}>
                  <Plus aria-hidden="true" />
                  Tambah unit
                </Button>
              </>
            ) : null}
          </>
        }
      />

      <Card>
        <CardContent>
          <search>
            <FilterBar onSubmit={submitSearch} aria-label="Cari dan filter unit">
              <Field className="min-w-0 flex-1" data-invalid={Boolean(queryError)}>
                <FieldLabel htmlFor="unit-search">Cari unit</FieldLabel>
                <SearchInput
                  id="unit-search"
                  name="q"
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
                />
                {queryError ? (
                  <FieldError id="unit-search-error">{queryError}</FieldError>
                ) : null}
              </Field>
              <Field className="md:w-40 md:shrink-0">
                <FieldLabel htmlFor="unit-active-filter">Status</FieldLabel>
                <NativeSelect
                  id="unit-active-filter"
                  name="is_active"
                  value={activeFilter}
                  onChange={(event) => {
                    const value = event.target.value;
                    list.setFilters({
                      is_active:
                        value === "breakdown"
                          ? "true"
                          : value === "active"
                            ? null
                            : value,
                      is_breakdown: value === "breakdown" ? "true" : null,
                    });
                  }}
                  className="w-full"
                >
                  <NativeSelectOption value="active">Aktif</NativeSelectOption>
                  <NativeSelectOption value="inactive">Nonaktif</NativeSelectOption>
                  <NativeSelectOption value="breakdown">Breakdown</NativeSelectOption>
                  <NativeSelectOption value="all">Semua</NativeSelectOption>
                </NativeSelect>
              </Field>
              <div className="flex gap-2 md:ml-auto">
                <Button type="submit" variant="outline" className="flex-1 md:flex-none">
                  <Filter aria-hidden="true" />
                  Terapkan
                </Button>
                {hasFilters ? (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={resetFilters}
                    className="flex-1 md:flex-none"
                  >
                    <RotateCcw aria-hidden="true" />
                    Reset
                  </Button>
                ) : null}
              </div>
            </FilterBar>
          </search>

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
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Daftar unit tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {actionError ? (
        <Alert
          role="alert"
          variant="destructive"
          className="border-destructive/30 bg-destructive/5"
        >
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Aksi gagal</AlertTitle>
          <AlertDescription>{actionError}</AlertDescription>
        </Alert>
      ) : null}

      {successMessage ? (
        <Alert
          aria-live="polite"
          className="border-emerald-500/30 bg-emerald-500/5"
        >
          <CircleCheck aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription>{successMessage}</AlertDescription>
        </Alert>
      ) : null}

      {importResult && importResult.failed > 0 ? (
        <Card className="gap-0">
          <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
            <div>
              <CardTitle>Baris impor gagal</CardTitle>
              <CardDescription>
                Perbaiki file lalu unggah ulang. Tidak ada unit yang disimpan dari file tersebut.
              </CardDescription>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setImportResult(null)}
            >
              Tutup
            </Button>
          </CardHeader>
          <CardContent className="px-0">
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead scope="col">Baris</TableHead>
                    <TableHead scope="col">Kode unit</TableHead>
                    <TableHead scope="col">Alasan</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {importResult.rows
                    .filter((row) => row.status === "failed")
                    .map((row) => (
                      <TableRow key={row.row}>
                        <TableCell>{row.row}</TableCell>
                        <TableCell className="font-medium">{row.code || "-"}</TableCell>
                        <TableCell>{row.message || "Data tidak valid"}</TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      ) : null}

      <Card className="gap-0">
        <CardHeader className="flex flex-row items-center justify-between gap-3 border-b">
          <div>
            <CardTitle>Daftar unit</CardTitle>
            <CardDescription>
              {loading ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
                  Memuat unit...
                </span>
              ) : (
                `${formatNumber(total)} unit ditemukan`
              )}
            </CardDescription>
          </div>
          {hasFilters ? (
            <Badge variant="outline" className="h-6">
              Terfilter
            </Badge>
          ) : null}
        </CardHeader>
        <CardContent className="px-0">
          {loading && units.length === 0 ? (
            <div className="space-y-3 px-6 py-4" role="status" aria-label="Memuat unit">
              {Array.from({ length: 6 }).map((_, index) => (
                <Skeleton key={index} className="h-12 w-full" />
              ))}
            </div>
          ) : units.length === 0 ? (
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
                  <Button type="button" variant="outline" onClick={resetFilters}>
                    <RotateCcw aria-hidden="true" />
                    Reset filter
                  </Button>
                ) : null}
                {canWrite ? (
                  <Button type="button" onClick={openCreate}>
                    <Plus aria-hidden="true" />
                    Tambah unit
                  </Button>
                ) : null}
              </EmptyContent>
            </Empty>
          ) : (
            <div aria-busy={loading}>
              <Table>
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
                    <TableHead scope="col" className="text-right">
                      Aksi
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {units.map((unit) => (
                    <TableRow key={unit.id}>
                      <TableHead scope="row" className="whitespace-normal">
                        <Link
                          href={`/units/${unit.id}`}
                          className="font-medium underline-offset-4 hover:underline"
                        >
                          {unit.code}
                        </Link>
                        <span className="block text-xs font-normal text-muted-foreground">
                          {unit.unit_type || "Tanpa tipe"}
                        </span>
                      </TableHead>
                      <TableCell>{modelLabel(unit)}</TableCell>
                      <TableCell>{unit.site_location || "-"}</TableCell>
                      <TableCell className="text-right">
                        {unit.hm_updated_at ? formatNumber(unit.current_hm) : "-"}
                      </TableCell>
                      <TableCell className="text-right">
                        {unit.km_updated_at ? formatNumber(unit.current_km) : "-"}
                      </TableCell>
                      <TableCell>
                        <UnitStatusBadge
                          active={unit.is_active}
                          breakdown={unit.is_breakdown}
                        />
                        <span className="mt-1 block text-xs text-muted-foreground">
                          {unit.hm_updated_at
                            ? `HM ${formatDateTimeWita(unit.hm_updated_at)}`
                            : "Belum ada reading"}
                        </span>
                      </TableCell>
                      <TableCell className="text-right">
                        <Link
                          href={`/units/${unit.id}`}
                          aria-label={`Detail unit ${unit.code}`}
                          className={buttonVariants({ variant: "ghost", size: "sm" })}
                        >
                          Detail
                          <ChevronRight aria-hidden="true" />
                        </Link>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
        {total > 0 ? (
          <CardFooter className="border-t">
            <div className="w-full">
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
            </div>
          </CardFooter>
        ) : null}
      </Card>

      <UnitCreateSheet
        key={`unit-create-${createKey}`}
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSaved={() => {
          setSuccessMessage("Unit berhasil ditambahkan.");
          refresh();
        }}
      />
      <UnitImportSheet
        key={`unit-import-${importKey}`}
        open={importOpen}
        onOpenChange={setImportOpen}
        onImported={handleImported}
      />
    </div>
  );
}

function modelLabel(unit: Unit): string {
  if (unit.unit_model) {
    return `${unit.unit_model.manufacturer} ${unit.unit_model.model}`;
  }
  return unit.merk || "-";
}
