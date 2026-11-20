"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { ListResponse, Unit } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const PAGE_SIZE = 25;

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
          setError(loadError instanceof ApiError ? loadError.message : "Daftar unit tidak dapat dimuat.");
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

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Unit</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Master unit, model, dan posisi HM/KM terakhir.
          </p>
        </div>
        {canWrite ? (
          <Link
            href="/units/new"
            className="inline-flex min-h-11 items-center rounded-md bg-primary px-4 text-sm font-semibold text-primary-foreground hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            Tambah unit
          </Link>
        ) : null}
      </div>

      <search>
        <form
          onSubmit={submitSearch}
          className="flex flex-col gap-3 rounded-xl border border-border bg-background p-4 sm:flex-row sm:items-end"
          aria-label="Cari dan filter unit"
        >
          <div className="min-w-0 flex-1">
            <label htmlFor="unit-search" className="text-sm font-medium">
              Cari unit
            </label>
            <input
              id="unit-search"
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Kode, nomor polisi, lokasi, atau merk"
              className="mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
            />
          </div>
          <div>
            <label htmlFor="unit-active-filter" className="text-sm font-medium">
              Status
            </label>
            <select
              id="unit-active-filter"
              value={activeFilter}
              onChange={(event) => {
                setActiveFilter(event.target.value);
                setPage(1);
              }}
              className="mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 sm:w-40"
            >
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
              <option value="all">Semua</option>
            </select>
          </div>
          <button
            type="submit"
            className="min-h-11 rounded-md border border-border px-4 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            Terapkan
          </button>
        </form>
      </search>

      {error ? (
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <div className="overflow-hidden rounded-xl border border-border bg-background">
        <div className="responsive-table-wrapper" tabIndex={0} role="region" aria-labelledby="unit-table-caption">
          <table className="responsive-table">
            <caption id="unit-table-caption" className="border-b border-border px-4 py-3 text-left text-sm text-muted-foreground">
              {loading ? "Memuat unit..." : range}
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
                    <Link href={`/units/${unit.id}`} className="font-semibold underline-offset-4 hover:underline">
                      {unit.code}
                    </Link>
                    <span className="block text-xs font-normal text-muted-foreground">
                      {unit.unit_type || "Tanpa tipe"}
                    </span>
                  </th>
                  <td data-label="Model">
                    {unit.unit_model
                      ? `${unit.unit_model.manufacturer} ${unit.unit_model.model}`
                      : unit.merk || "-"}
                  </td>
                  <td data-label="Lokasi">{unit.site_location || "-"}</td>
                  <td data-label="HM" className="text-right tabular-nums">
                    {formatNumber(unit.current_hm)}
                  </td>
                  <td data-label="KM" className="text-right tabular-nums">
                    {formatNumber(unit.current_km)}
                  </td>
                  <td data-label="Status">
                    <StatusBadge active={unit.is_active} breakdown={unit.is_breakdown} />
                    <span className="mt-1 block text-xs text-muted-foreground">
                      {unit.hm_updated_at ? `HM ${formatDateTimeWita(unit.hm_updated_at)}` : "Belum ada reading"}
                    </span>
                  </td>
                  <td data-label="Aksi" className="text-right">
                    <Link
                      href={`/units/${unit.id}`}
                      className="inline-flex min-h-10 items-center rounded-md border border-border px-3 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                    >
                      Detail
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {!loading && units.length === 0 && !error ? (
          <div className="px-4 py-12 text-center">
            <p className="font-medium">Belum ada unit.</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Tambahkan unit atau ubah filter pencarian.
            </p>
          </div>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">{range}</p>
        <div className="flex gap-2">
          <button
            type="button"
            disabled={page <= 1 || loading}
            onClick={() => setPage((current) => Math.max(1, current - 1))}
            className="min-h-10 rounded-md border border-border px-3 text-sm font-medium hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            Sebelumnya
          </button>
          <button
            type="button"
            disabled={page >= totalPages || loading}
            onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            className="min-h-10 rounded-md border border-border px-3 text-sm font-medium hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            Berikutnya
          </button>
        </div>
      </div>
    </div>
  );
}

function StatusBadge({ active, breakdown }: { active: boolean; breakdown: boolean }) {
  if (!active) {
    return (
      <span className="inline-flex rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-muted-foreground">
        Nonaktif
      </span>
    );
  }
  if (breakdown) {
    return (
      <span className="inline-flex rounded-full bg-destructive/10 px-2.5 py-1 text-xs font-medium text-destructive">
        Breakdown
      </span>
    );
  }
  return (
    <span className="inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-800">
      Aktif
    </span>
  );
}
