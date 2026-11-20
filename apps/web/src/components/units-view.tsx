"use client";

import {
  ChevronLeft,
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
import { useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import { UnitStatusBadge } from "@/components/status-badges";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { ListResponse, Unit } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const PAGE_SIZE = 25;

const filterLabels: Record<string, string> = {
  active: "Aktif",
  inactive: "Nonaktif",
  all: "Semua",
};

export function UnitsView() {
  const { user } = useSession();
  const [units, setUnits] = useState<Unit[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState("");
  const [appliedQuery, setAppliedQuery] = useState("");
  const [activeFilter, setActiveFilter] = useState("active");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const canWrite = user.role === "admin" || user.role === "planner";
  const hasFilters = appliedQuery.trim() !== "" || activeFilter !== "active";

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");

    const params = new URLSearchParams({
      page: String(page),
      limit: String(PAGE_SIZE),
      sort: "code",
      order: "asc",
    });
    if (appliedQuery.trim()) {
      params.set("q", appliedQuery.trim());
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
  }, [page, appliedQuery, activeFilter]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const range = useMemo(() => {
    if (total === 0) {
      return "Tidak ada unit";
    }
    const start = (page - 1) * PAGE_SIZE + 1;
    const end = Math.min(page * PAGE_SIZE, total);
    return `${start}–${end} dari ${total} unit`;
  }, [page, total]);

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPage(1);
    setAppliedQuery(query);
  };

  const resetFilters = () => {
    setQuery("");
    setAppliedQuery("");
    setActiveFilter("active");
    setPage(1);
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
          className="rounded-xl border border-border bg-background p-4 shadow-sm"
          aria-label="Cari dan filter unit"
        >
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="min-w-0 flex-1">
              <Label htmlFor="unit-search">Cari unit</Label>
              <div className="relative mt-2">
                <Search
                  className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                  aria-hidden="true"
                />
                <Input
                  id="unit-search"
                  name="q"
                  type="search"
                  enterKeyHint="search"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Kode, nopol, lokasi, atau merk"
                  className="pl-10"
                />
              </div>
            </div>
            <div>
              <Label htmlFor="unit-active-filter">Status</Label>
              <select
                id="unit-active-filter"
                name="is_active"
                value={activeFilter}
                onChange={(event) => {
                  setActiveFilter(event.target.value);
                  setPage(1);
                }}
                className="mt-2 h-11 w-full rounded-lg border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 sm:w-40"
              >
                <option value="active">Aktif</option>
                <option value="inactive">Nonaktif</option>
                <option value="all">Semua</option>
              </select>
            </div>
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
              {appliedQuery.trim() ? (
                <Badge variant="outline" className="h-6 px-2.5">
                  Cari: {appliedQuery.trim()}
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

      <div className="overflow-hidden rounded-xl border border-border bg-background shadow-sm">
        {loading && units.length === 0 ? (
          <div className="space-y-3 p-4" role="status" aria-label="Memuat unit">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-12 w-full" />
            ))}
          </div>
        ) : units.length === 0 && !error ? (
          <div className="px-4 py-14 text-center">
            <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-muted text-muted-foreground">
              <Truck className="size-6" aria-hidden="true" />
            </span>
            <p className="mt-4 font-medium">Belum ada unit.</p>
            <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
              {hasFilters
                ? "Tidak ada unit yang cocok dengan filter. Coba ubah kata kunci atau status."
                : "Tambahkan unit pertama untuk mulai mencatat HM/KM."}
            </p>
            <div className="mt-5 flex flex-wrap justify-center gap-2">
              {hasFilters ? (
                <Button type="button" variant="outline" onClick={resetFilters} className="h-11 px-4">
                  <RotateCcw aria-hidden="true" />
                  Reset filter
                </Button>
              ) : null}
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
              <caption
                id="unit-table-caption"
                className="border-b border-border px-4 py-3 text-left text-sm text-muted-foreground"
                aria-live="polite"
              >
                {loading ? (
                  <span className="inline-flex items-center gap-2">
                    <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                    Memuat unit...
                  </span>
                ) : (
                  range
                )}
              </caption>
              <thead>
                <tr>
                  <th scope="col">Unit</th>
                  <th scope="col">Model</th>
                  <th scope="col">Lokasi</th>
                  <th scope="col" className="text-right">HM</th>
                  <th scope="col" className="text-right">KM</th>
                  <th scope="col">Status</th>
                  <th scope="col" className="text-right">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {units.map((unit) => (
                  <tr key={unit.id}>
                    <th scope="row" className="unit-row-heading">
                      <Link
                        href={`/units/${unit.id}`}
                        className="font-semibold underline-offset-4 hover:underline"
                      >
                        {unit.code}
                      </Link>
                      <span className="block text-xs font-normal text-muted-foreground">
                        {unit.unit_type || "Tanpa tipe"}
                      </span>
                    </th>
                    <td data-label="Model">{modelLabel(unit)}</td>
                    <td data-label="Lokasi">{unit.site_location || "-"}</td>
                    <td data-label="HM" className="text-right tabular-nums">
                      {unit.hm_updated_at ? formatNumber(unit.current_hm) : "—"}
                    </td>
                    <td data-label="KM" className="text-right tabular-nums">
                      {unit.km_updated_at ? formatNumber(unit.current_km) : "—"}
                    </td>
                    <td data-label="Status">
                      <UnitStatusBadge active={unit.is_active} breakdown={unit.is_breakdown} />
                      <span className="mt-1 block text-xs text-muted-foreground">
                        {unit.hm_updated_at
                          ? `HM ${formatDateTimeWita(unit.hm_updated_at)}`
                          : "Belum ada reading"}
                      </span>
                    </td>
                    <td data-label="Aksi" className="text-right">
                      <Link
                        href={`/units/${unit.id}`}
                        className="inline-flex min-h-10 items-center gap-1.5 rounded-lg border border-border px-3 text-sm font-medium transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                      >
                        Detail
                        <ChevronRight className="size-4" aria-hidden="true" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground" aria-live="polite">
          {range} · Halaman {page} dari {totalPages}
        </p>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={page <= 1 || loading}
            onClick={() => setPage((current) => Math.max(1, current - 1))}
            className="h-10 px-3"
            aria-label="Halaman sebelumnya"
          >
            <ChevronLeft aria-hidden="true" />
            Sebelumnya
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={page >= totalPages || loading}
            onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            className="h-10 px-3"
            aria-label="Halaman berikutnya"
          >
            Berikutnya
            <ChevronRight aria-hidden="true" />
          </Button>
        </div>
      </div>
    </div>
  );
}

function modelLabel(unit: Unit): string {
  if (unit.unit_model) {
    return `${unit.unit_model.manufacturer} ${unit.unit_model.model}`;
  }
  return unit.merk || "-";
}
