"use client";

import {
  Boxes,
  CircleAlert,
  CircleCheck,
  Loader2,
  RefreshCw,
  Search,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { KpiCard } from "@/components/kpi-card";
import { PageHeader } from "@/components/layout/page-header";
import { FilterBar } from "@/components/list/list-filter-bar";
import { SearchInput } from "@/components/list/list-search";
import { ActiveBadge } from "@/components/status-badges";
import { TablePagination } from "@/components/table-pagination";
import { TableSortHeader } from "@/components/table-sort-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
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
import { Textarea } from "@/components/ui/textarea";
import { useListParams } from "@/hooks/use-list-params";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  EngineModel,
  ListResponse,
  UnitModel,
} from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

const unitModelSorts = [
  "manufacturer",
  "model",
  "category",
  "is_active",
  "updated_at",
] as const;
const engineModelSorts = [
  "manufacturer",
  "model",
  "fuel_type",
  "displacement_cc",
  "is_active",
  "updated_at",
] as const;

type MasterTab = "unit" | "engine";

export function MasterView() {
  const { user } = useSession();
  const searchParams = useSearchParams();
  const umQ = searchParams.get("umQ") ?? "";
  const emQ = searchParams.get("emQ") ?? "";
  const umActive = searchParams.get("umActive") === "1";
  const emActive = searchParams.get("emActive") === "1";

  const unitList = useListParams({
    prefix: "um",
    defaultSort: "manufacturer",
    allowedSorts: unitModelSorts,
  });
  const engineList = useListParams({
    prefix: "em",
    defaultSort: "manufacturer",
    allowedSorts: engineModelSorts,
  });

  const [tab, setTab] = useState<MasterTab>("unit");
  const [unitModels, setUnitModels] = useState<UnitModel[]>([]);
  const [unitTotal, setUnitTotal] = useState(0);
  const [unitLoading, setUnitLoading] = useState(true);
  const [unitError, setUnitError] = useState("");
  const [engineModels, setEngineModels] = useState<EngineModel[]>([]);
  const [engineTotal, setEngineTotal] = useState(0);
  const [engineLoading, setEngineLoading] = useState(true);
  const [engineError, setEngineError] = useState("");
  const [umQuery, setUmQuery] = useState(umQ);
  const [umQueryError, setUmQueryError] = useState("");
  const [emQuery, setEmQuery] = useState(emQ);
  const [emQueryError, setEmQueryError] = useState("");
  const [unitStats, setUnitStats] = useState<{ total: number; active: number } | null>(null);
  const [engineStats, setEngineStats] = useState<{ total: number; active: number } | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [statsError, setStatsError] = useState(false);

  const canWrite = user.role === "admin" || user.role === "planner";
  const unitTotalPages = Math.max(1, Math.ceil(unitTotal / unitList.pageSize));
  const engineTotalPages = Math.max(1, Math.ceil(engineTotal / engineList.pageSize));

  useEffect(() => {
    setUmQuery(umQ);
    setUmQueryError("");
  }, [umQ]);

  useEffect(() => {
    setEmQuery(emQ);
    setEmQueryError("");
  }, [emQ]);

  const loadUnitModels = useCallback(async () => {
    setUnitLoading(true);
    setUnitError("");
    const params = new URLSearchParams({
      page: String(unitList.page),
      limit: String(unitList.pageSize),
      sort: unitList.sort,
      order: unitList.order,
    });
    if (umQ.trim()) params.set("q", umQ.trim());
    if (umActive) params.set("active_only", "true");
    try {
      const result = await proxyGet<ListResponse<UnitModel>>(
        `/unit-models?${params.toString()}`,
      );
      setUnitModels(result.data);
      setUnitTotal(result.meta.total);
    } catch (loadError) {
      setUnitError(
        loadError instanceof ApiError
          ? loadError.message
          : "Master unit model tidak dapat dimuat.",
      );
    } finally {
      setUnitLoading(false);
    }
  }, [umActive, umQ, unitList.order, unitList.page, unitList.pageSize, unitList.sort]);

  const loadEngineModels = useCallback(async () => {
    setEngineLoading(true);
    setEngineError("");
    const params = new URLSearchParams({
      page: String(engineList.page),
      limit: String(engineList.pageSize),
      sort: engineList.sort,
      order: engineList.order,
    });
    if (emQ.trim()) params.set("q", emQ.trim());
    if (emActive) params.set("active_only", "true");
    try {
      const result = await proxyGet<ListResponse<EngineModel>>(
        `/engine-models?${params.toString()}`,
      );
      setEngineModels(result.data);
      setEngineTotal(result.meta.total);
    } catch (loadError) {
      setEngineError(
        loadError instanceof ApiError
          ? loadError.message
          : "Master engine model tidak dapat dimuat.",
      );
    } finally {
      setEngineLoading(false);
    }
  }, [emActive, emQ, engineList.order, engineList.page, engineList.pageSize, engineList.sort]);

  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const [unitAll, unitActiveResult, engineAll, engineActiveResult] = await Promise.all([
        proxyGet<ListResponse<UnitModel>>("/unit-models?page=1&limit=1"),
        proxyGet<ListResponse<UnitModel>>("/unit-models?page=1&limit=1&active_only=true"),
        proxyGet<ListResponse<EngineModel>>("/engine-models?page=1&limit=1"),
        proxyGet<ListResponse<EngineModel>>("/engine-models?page=1&limit=1&active_only=true"),
      ]);
      setUnitStats({ total: unitAll.meta.total, active: unitActiveResult.meta.total });
      setEngineStats({ total: engineAll.meta.total, active: engineActiveResult.meta.total });
      setStatsError(false);
    } catch {
      setUnitStats(null);
      setEngineStats(null);
      setStatsError(true);
    } finally {
      setStatsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadUnitModels();
  }, [loadUnitModels]);

  useEffect(() => {
    void loadEngineModels();
  }, [loadEngineModels]);

  useEffect(() => {
    void loadStats();
  }, [loadStats]);

  const refreshAll = useCallback(() => {
    void loadUnitModels();
    void loadEngineModels();
    void loadStats();
  }, [loadEngineModels, loadStats, loadUnitModels]);

  const submitUnitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const term = umQuery.trim();
    if (term.length > 100) {
      setUmQueryError("Kata kunci maksimal 100 karakter.");
      requestAnimationFrame(() => document.getElementById("um-search")?.focus());
      return;
    }
    setUmQueryError("");
    unitList.setFilters({ umQ: term || null, umActive: umActive ? "1" : null });
  };

  const submitEngineSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const term = emQuery.trim();
    if (term.length > 100) {
      setEmQueryError("Kata kunci maksimal 100 karakter.");
      requestAnimationFrame(() => document.getElementById("em-search")?.focus());
      return;
    }
    setEmQueryError("");
    engineList.setFilters({ emQ: term || null, emActive: emActive ? "1" : null });
  };

  const resetUnitSearch = () => {
    setUmQuery("");
    setUmQueryError("");
    unitList.setFilters({ umQ: null, umActive: null });
  };

  const resetEngineSearch = () => {
    setEmQuery("");
    setEmQueryError("");
    engineList.setFilters({ emQ: null, emActive: null });
  };

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Master unit"
        description="Referensi unit model dan engine model yang dipakai pada master unit."
        actions={
          <Button
            type="button"
            variant="outline"
            onClick={refreshAll}
            disabled={unitLoading || engineLoading || statsLoading}
          >
            <RefreshCw
              className={unitLoading || engineLoading || statsLoading ? "animate-spin" : undefined}
              aria-hidden="true"
            />
            Muat ulang
          </Button>
        }
      />

      {!canWrite ? (
        <Alert className="border-sky-500/30 bg-sky-500/5">
          <ShieldCheck aria-hidden="true" />
          <AlertTitle>Akses baca saja</AlertTitle>
          <AlertDescription>
            Role Anda dapat melihat master unit, tetapi tidak dapat menambah model.
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Unit model"
          value={unitStats?.total}
          loading={statsLoading}
          icon={Boxes}
          hint={statsError ? "Ringkasan tidak tersedia" : "Total referensi unit"}
        />
        <KpiCard
          label="Unit model aktif"
          value={unitStats?.active}
          loading={statsLoading}
          icon={CircleCheck}
          hint={statsError ? "Ringkasan tidak tersedia" : "Siap dipilih untuk unit"}
        />
        <KpiCard
          label="Engine model"
          value={engineStats?.total}
          loading={statsLoading}
          icon={Settings2}
          hint={statsError ? "Ringkasan tidak tersedia" : "Total referensi engine"}
        />
        <KpiCard
          label="Engine model aktif"
          value={engineStats?.active}
          loading={statsLoading}
          icon={CircleCheck}
          hint={statsError ? "Ringkasan tidak tersedia" : "Siap dipilih untuk unit"}
        />
      </div>

      <div className="flex gap-2 overflow-x-auto pb-1" role="tablist" aria-label="Master model">
        <Button
          type="button"
          size="sm"
          role="tab"
          aria-selected={tab === "unit"}
          variant={tab === "unit" ? "default" : "outline"}
          onClick={() => setTab("unit")}
          className="shrink-0"
        >
          <Boxes aria-hidden="true" /> Unit model
        </Button>
        <Button
          type="button"
          size="sm"
          role="tab"
          aria-selected={tab === "engine"}
          variant={tab === "engine" ? "default" : "outline"}
          onClick={() => setTab("engine")}
          className="shrink-0"
        >
          <Settings2 aria-hidden="true" /> Engine model
        </Button>
      </div>

      {tab === "unit" ? (
        <Card role="tabpanel" aria-label="Unit model">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Boxes className="size-4 text-muted-foreground" aria-hidden="true" />
              Daftar unit model
            </CardTitle>
            <CardDescription>Referensi model untuk master unit.</CardDescription>
            <FilterBar
              role="search"
              aria-label="Cari unit model"
              onSubmit={submitUnitSearch}
              className="mt-2 grid gap-3 md:grid-cols-2 xl:grid-cols-4"
            >
              <Field className="xl:col-span-2">
                <FieldLabel htmlFor="um-search">Cari unit model</FieldLabel>
                <SearchInput
                  id="um-search"
                  name="umQ"
                  value={umQuery}
                  onChange={(event) => {
                    setUmQuery(event.target.value);
                    setUmQueryError("");
                  }}
                  aria-invalid={Boolean(umQueryError)}
                  aria-describedby={umQueryError ? "um-search-error" : undefined}
                  placeholder="Manufacturer, model, atau kategori..."
                  autoCapitalize="none"
                  spellCheck={false}
                />
                {umQueryError ? (
                  <FieldError id="um-search-error">{umQueryError}</FieldError>
                ) : null}
              </Field>
              <Field>
                <FieldLabel htmlFor="um-status">Status</FieldLabel>
                <NativeSelect
                  id="um-status"
                  value={umActive ? "1" : ""}
                  onChange={(event) =>
                    unitList.setFilters({ umActive: event.target.value || null })
                  }
                >
                  <NativeSelectOption value="">Semua status</NativeSelectOption>
                  <NativeSelectOption value="1">Hanya aktif</NativeSelectOption>
                </NativeSelect>
              </Field>
              <div className="flex items-end gap-2">
                <Button type="submit">
                  <Search aria-hidden="true" /> Cari
                </Button>
                <Button type="button" variant="outline" onClick={resetUnitSearch}>
                  Reset
                </Button>
              </div>
            </FilterBar>
          </CardHeader>

          <CardContent className="flex flex-col gap-3">
            {unitError ? (
              <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                <CircleAlert aria-hidden="true" />
                <AlertTitle>Master unit model tidak dapat dimuat</AlertTitle>
                <AlertDescription>{unitError}</AlertDescription>
              </Alert>
            ) : null}

            {canWrite ? (
              <UnitModelForm
                onSaved={() => {
                  void loadUnitModels();
                  void loadStats();
                }}
              />
            ) : null}

            {unitLoading && unitModels.length === 0 ? (
              <TableSkeleton />
            ) : unitModels.length === 0 ? (
              <EmptyModels
                label="unit model"
                icon={Boxes}
                description="Tambahkan referensi model agar dapat dipilih saat membuat unit."
              />
            ) : (
              <div className="overflow-x-auto" aria-busy={unitLoading}>
                <Table>
                  <TableCaption className="sr-only">Daftar unit model</TableCaption>
                  <TableHeader>
                    <TableRow>
                      <TableSortHeader
                        label="Manufacturer"
                        sortKey="manufacturer"
                        prefix="um"
                        currentSort={unitList.sort}
                        currentOrder={unitList.order}
                        onSort={unitList.toggleSort}
                      />
                      <TableSortHeader
                        label="Model"
                        sortKey="model"
                        prefix="um"
                        currentSort={unitList.sort}
                        currentOrder={unitList.order}
                        onSort={unitList.toggleSort}
                      />
                      <TableSortHeader
                        label="Kategori"
                        sortKey="category"
                        prefix="um"
                        currentSort={unitList.sort}
                        currentOrder={unitList.order}
                        onSort={unitList.toggleSort}
                      />
                      <TableSortHeader
                        label="Status"
                        sortKey="is_active"
                        prefix="um"
                        currentSort={unitList.sort}
                        currentOrder={unitList.order}
                        onSort={unitList.toggleSort}
                      />
                      <TableSortHeader
                        label="Diperbarui"
                        sortKey="updated_at"
                        prefix="um"
                        currentSort={unitList.sort}
                        currentOrder={unitList.order}
                        onSort={unitList.toggleSort}
                      />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {unitModels.map((model) => (
                      <TableRow key={model.id}>
                        <TableHead scope="row" className="whitespace-normal">
                          {model.manufacturer}
                        </TableHead>
                        <TableCell className="font-medium">{model.model}</TableCell>
                        <TableCell>{model.category || "-"}</TableCell>
                        <TableCell>
                          <ActiveBadge active={model.is_active} />
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDateTimeWita(model.updated_at)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            <TablePagination
              page={unitList.page}
              pageSize={unitList.pageSize}
              total={unitTotal}
              totalPages={unitTotalPages}
              loading={unitLoading}
              itemLabel="model unit"
              onPageChange={unitList.setPage}
              onPageSizeChange={unitList.setPageSize}
            />
          </CardContent>
        </Card>
      ) : (
        <Card role="tabpanel" aria-label="Engine model">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings2 className="size-4 text-muted-foreground" aria-hidden="true" />
              Daftar engine model
            </CardTitle>
            <CardDescription>Referensi engine untuk master unit.</CardDescription>
            <FilterBar
              role="search"
              aria-label="Cari engine model"
              onSubmit={submitEngineSearch}
              className="mt-2 grid gap-3 md:grid-cols-2 xl:grid-cols-4"
            >
              <Field className="xl:col-span-2">
                <FieldLabel htmlFor="em-search">Cari engine model</FieldLabel>
                <SearchInput
                  id="em-search"
                  name="emQ"
                  value={emQuery}
                  onChange={(event) => {
                    setEmQuery(event.target.value);
                    setEmQueryError("");
                  }}
                  aria-invalid={Boolean(emQueryError)}
                  aria-describedby={emQueryError ? "em-search-error" : undefined}
                  placeholder="Manufacturer, model, atau bahan bakar..."
                  autoCapitalize="none"
                  spellCheck={false}
                />
                {emQueryError ? (
                  <FieldError id="em-search-error">{emQueryError}</FieldError>
                ) : null}
              </Field>
              <Field>
                <FieldLabel htmlFor="em-status">Status</FieldLabel>
                <NativeSelect
                  id="em-status"
                  value={emActive ? "1" : ""}
                  onChange={(event) =>
                    engineList.setFilters({ emActive: event.target.value || null })
                  }
                >
                  <NativeSelectOption value="">Semua status</NativeSelectOption>
                  <NativeSelectOption value="1">Hanya aktif</NativeSelectOption>
                </NativeSelect>
              </Field>
              <div className="flex items-end gap-2">
                <Button type="submit">
                  <Search aria-hidden="true" /> Cari
                </Button>
                <Button type="button" variant="outline" onClick={resetEngineSearch}>
                  Reset
                </Button>
              </div>
            </FilterBar>
          </CardHeader>

          <CardContent className="flex flex-col gap-3">
            {engineError ? (
              <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                <CircleAlert aria-hidden="true" />
                <AlertTitle>Master engine model tidak dapat dimuat</AlertTitle>
                <AlertDescription>{engineError}</AlertDescription>
              </Alert>
            ) : null}

            {canWrite ? (
              <EngineModelForm
                onSaved={() => {
                  void loadEngineModels();
                  void loadStats();
                }}
              />
            ) : null}

            {engineLoading && engineModels.length === 0 ? (
              <TableSkeleton />
            ) : engineModels.length === 0 ? (
              <EmptyModels
                label="engine model"
                icon={Settings2}
                description="Tambahkan referensi engine agar dapat dipilih saat membuat unit."
              />
            ) : (
              <div className="overflow-x-auto" aria-busy={engineLoading}>
                <Table>
                  <TableCaption className="sr-only">Daftar engine model</TableCaption>
                  <TableHeader>
                    <TableRow>
                      <TableSortHeader
                        label="Manufacturer"
                        sortKey="manufacturer"
                        prefix="em"
                        currentSort={engineList.sort}
                        currentOrder={engineList.order}
                        onSort={engineList.toggleSort}
                      />
                      <TableSortHeader
                        label="Model"
                        sortKey="model"
                        prefix="em"
                        currentSort={engineList.sort}
                        currentOrder={engineList.order}
                        onSort={engineList.toggleSort}
                      />
                      <TableSortHeader
                        label="Bahan bakar"
                        sortKey="fuel_type"
                        prefix="em"
                        currentSort={engineList.sort}
                        currentOrder={engineList.order}
                        onSort={engineList.toggleSort}
                      />
                      <TableSortHeader
                        label="Displacement"
                        sortKey="displacement_cc"
                        prefix="em"
                        currentSort={engineList.sort}
                        currentOrder={engineList.order}
                        onSort={engineList.toggleSort}
                        align="end"
                      />
                      <TableSortHeader
                        label="Status"
                        sortKey="is_active"
                        prefix="em"
                        currentSort={engineList.sort}
                        currentOrder={engineList.order}
                        onSort={engineList.toggleSort}
                      />
                      <TableSortHeader
                        label="Diperbarui"
                        sortKey="updated_at"
                        prefix="em"
                        currentSort={engineList.sort}
                        currentOrder={engineList.order}
                        onSort={engineList.toggleSort}
                      />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {engineModels.map((model) => (
                      <TableRow key={model.id}>
                        <TableHead scope="row" className="whitespace-normal">
                          {model.manufacturer}
                        </TableHead>
                        <TableCell className="font-medium">{model.model}</TableCell>
                        <TableCell>{model.fuel_type || "-"}</TableCell>
                        <TableCell className="text-right">
                          {model.displacement_cc ? `${formatNumber(model.displacement_cc)} cc` : "-"}
                        </TableCell>
                        <TableCell>
                          <ActiveBadge active={model.is_active} />
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDateTimeWita(model.updated_at)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            <TablePagination
              page={engineList.page}
              pageSize={engineList.pageSize}
              total={engineTotal}
              totalPages={engineTotalPages}
              loading={engineLoading}
              itemLabel="engine model"
              onPageChange={engineList.setPage}
              onPageSizeChange={engineList.setPageSize}
            />
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function UnitModelForm({ onSaved }: { onSaved: () => void }) {
  const [manufacturer, setManufacturer] = useState("");
  const [model, setModel] = useState("");
  const [category, setCategory] = useState("");
  const [notes, setNotes] = useState("");
  const [active, setActive] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setPending(true);
    try {
      await proxyPost<DataResponse<UnitModel>>("/unit-models", {
        manufacturer: manufacturer.trim(),
        model: model.trim(),
        category: category.trim(),
        notes: notes.trim(),
        is_active: active,
      });
      setManufacturer("");
      setModel("");
      setCategory("");
      setNotes("");
      setActive(true);
      setMessage("Unit model berhasil ditambahkan.");
      onSaved();
    } catch (submitError) {
      setError(
        submitError instanceof ApiError
          ? submitError.message
          : "Unit model tidak dapat disimpan.",
      );
      requestAnimationFrame(() => document.getElementById("unit-model-form-error")?.focus());
    } finally {
      setPending(false);
    }
  };

  return (
    <form
      onSubmit={(event) => void submit(event)}
      className="grid gap-3 rounded-lg border border-border p-3 sm:grid-cols-2 xl:grid-cols-4"
      noValidate
    >
      <div className="sm:col-span-2 xl:col-span-4">
        <p className="text-sm font-medium">Tambah unit model</p>
        <p className="text-xs text-muted-foreground">
          Referensi baru langsung tersedia untuk pembuatan unit.
        </p>
      </div>

      <Field>
        <FieldLabel htmlFor="um-manufacturer">
          Manufacturer <span aria-hidden="true" className="text-destructive">*</span>
        </FieldLabel>
        <Input
          id="um-manufacturer"
          name="manufacturer"
          required
          maxLength={100}
          autoComplete="off"
          value={manufacturer}
          onChange={(event) => setManufacturer(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="um-model">
          Model <span aria-hidden="true" className="text-destructive">*</span>
        </FieldLabel>
        <Input
          id="um-model"
          name="model"
          required
          maxLength={100}
          autoComplete="off"
          value={model}
          onChange={(event) => setModel(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="um-category">Kategori</FieldLabel>
        <Input
          id="um-category"
          name="category"
          maxLength={100}
          value={category}
          onChange={(event) => setCategory(event.target.value)}
        />
      </Field>
      <Field orientation="horizontal" className="w-fit self-end pb-2">
        <input
          id="um-active"
          type="checkbox"
          name="is_active"
          checked={active}
          onChange={(event) => setActive(event.target.checked)}
          className="size-4 accent-primary"
        />
        <FieldLabel htmlFor="um-active">Aktif</FieldLabel>
      </Field>
      <Field className="sm:col-span-2 xl:col-span-4">
        <FieldLabel htmlFor="um-notes">Catatan</FieldLabel>
        <Textarea
          id="um-notes"
          name="notes"
          rows={2}
          maxLength={2000}
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
        />
      </Field>

      {error ? (
        <Alert
          id="unit-model-form-error"
          tabIndex={-1}
          variant="destructive"
          className="border-destructive/30 bg-destructive/5 sm:col-span-2 xl:col-span-4"
        >
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal menyimpan</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      {message ? (
        <Alert
          role="status"
          className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300 sm:col-span-2 xl:col-span-4"
        >
          <CircleCheck aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription className="text-emerald-700 dark:text-emerald-300">
            {message}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="sm:col-span-2 xl:col-span-4">
        <Button type="submit" size="sm" disabled={pending}>
          {pending ? (
            <>
              <Loader2 className="animate-spin" aria-hidden="true" />
              Menyimpan...
            </>
          ) : (
            "Tambah unit model"
          )}
        </Button>
      </div>
    </form>
  );
}

function EngineModelForm({ onSaved }: { onSaved: () => void }) {
  const [manufacturer, setManufacturer] = useState("");
  const [model, setModel] = useState("");
  const [fuelType, setFuelType] = useState("");
  const [displacement, setDisplacement] = useState("");
  const [notes, setNotes] = useState("");
  const [active, setActive] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setPending(true);
    try {
      await proxyPost<DataResponse<EngineModel>>("/engine-models", {
        manufacturer: manufacturer.trim(),
        model: model.trim(),
        fuel_type: fuelType.trim(),
        displacement_cc: displacement ? Number(displacement) : null,
        notes: notes.trim(),
        is_active: active,
      });
      setManufacturer("");
      setModel("");
      setFuelType("");
      setDisplacement("");
      setNotes("");
      setActive(true);
      setMessage("Engine model berhasil ditambahkan.");
      onSaved();
    } catch (submitError) {
      setError(
        submitError instanceof ApiError
          ? submitError.message
          : "Engine model tidak dapat disimpan.",
      );
      requestAnimationFrame(() => document.getElementById("engine-model-form-error")?.focus());
    } finally {
      setPending(false);
    }
  };

  return (
    <form
      onSubmit={(event) => void submit(event)}
      className="grid gap-3 rounded-lg border border-border p-3 sm:grid-cols-2 xl:grid-cols-4"
      noValidate
    >
      <div className="sm:col-span-2 xl:col-span-4">
        <p className="text-sm font-medium">Tambah engine model</p>
        <p className="text-xs text-muted-foreground">
          Referensi baru langsung tersedia untuk pembuatan unit.
        </p>
      </div>

      <Field>
        <FieldLabel htmlFor="em-manufacturer">
          Manufacturer <span aria-hidden="true" className="text-destructive">*</span>
        </FieldLabel>
        <Input
          id="em-manufacturer"
          name="manufacturer"
          required
          maxLength={100}
          autoComplete="off"
          value={manufacturer}
          onChange={(event) => setManufacturer(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="em-model">
          Model <span aria-hidden="true" className="text-destructive">*</span>
        </FieldLabel>
        <Input
          id="em-model"
          name="model"
          required
          maxLength={100}
          autoComplete="off"
          value={model}
          onChange={(event) => setModel(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="em-fuel">Bahan bakar</FieldLabel>
        <Input
          id="em-fuel"
          name="fuel_type"
          maxLength={50}
          value={fuelType}
          onChange={(event) => setFuelType(event.target.value)}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor="em-displacement">Displacement (cc)</FieldLabel>
        <Input
          id="em-displacement"
          name="displacement_cc"
          type="number"
          inputMode="numeric"
          min="1"
          max="1000000"
          step="1"
          value={displacement}
          onChange={(event) => setDisplacement(event.target.value)}
        />
      </Field>
      <Field className="sm:col-span-2 xl:col-span-4">
        <FieldLabel htmlFor="em-notes">Catatan</FieldLabel>
        <Textarea
          id="em-notes"
          name="notes"
          rows={2}
          maxLength={2000}
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
        />
      </Field>

      {error ? (
        <Alert
          id="engine-model-form-error"
          tabIndex={-1}
          variant="destructive"
          className="border-destructive/30 bg-destructive/5 sm:col-span-2 xl:col-span-4"
        >
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal menyimpan</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      {message ? (
        <Alert
          role="status"
          className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300 sm:col-span-2 xl:col-span-4"
        >
          <CircleCheck aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription className="text-emerald-700 dark:text-emerald-300">
            {message}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-wrap items-center gap-3 sm:col-span-2 xl:col-span-4">
        <Field orientation="horizontal" className="w-fit">
          <input
            id="em-active"
            type="checkbox"
            name="is_active"
            checked={active}
            onChange={(event) => setActive(event.target.checked)}
            className="size-4 accent-primary"
          />
          <FieldLabel htmlFor="em-active">Aktif</FieldLabel>
        </Field>
        <Button type="submit" size="sm" disabled={pending}>
          {pending ? (
            <>
              <Loader2 className="animate-spin" aria-hidden="true" />
              Menyimpan...
            </>
          ) : (
            "Tambah engine model"
          )}
        </Button>
      </div>
    </form>
  );
}

function TableSkeleton() {
  return (
    <div className="space-y-2 py-2" role="status" aria-label="Memuat master model">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-12 w-full" />
      ))}
    </div>
  );
}

function EmptyModels({
  label,
  icon: Icon,
  description,
}: {
  label: string;
  icon: LucideIcon;
  description: string;
}) {
  return (
    <Empty className="border-0 py-10">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Icon aria-hidden="true" />
        </EmptyMedia>
        <EmptyTitle>Belum ada {label}.</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
