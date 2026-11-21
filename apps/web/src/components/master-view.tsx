"use client";

import {
  Boxes,
  CircleAlert,
  CircleCheck,
  Loader2,
  Search,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ActiveBadge } from "@/components/status-badges";
import { TablePagination } from "@/components/table-pagination";
import { TableSortHeader } from "@/components/table-sort-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
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
import { Textarea } from "@/components/ui/textarea";
import { useListParams } from "@/hooks/use-list-params";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  EngineModel,
  ListResponse,
  UnitModel,
} from "@/lib/types";
import { formatDateTimeWita } from "@/lib/time";

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

export function MasterView() {
  const { user } = useSession();
  const searchParams = useSearchParams();
  const umQ = searchParams.get("umQ") ?? "";
  const emQ = searchParams.get("emQ") ?? "";

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
  }, [umQ, unitList.order, unitList.page, unitList.pageSize, unitList.sort]);

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
  }, [emQ, engineList.order, engineList.page, engineList.pageSize, engineList.sort]);

  useEffect(() => {
    void loadUnitModels();
  }, [loadUnitModels]);

  useEffect(() => {
    void loadEngineModels();
  }, [loadEngineModels]);

  const submitUnitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const term = umQuery.trim();
    if (term.length > 100) {
      setUmQueryError("Kata kunci maksimal 100 karakter.");
      requestAnimationFrame(() => document.getElementById("um-search")?.focus());
      return;
    }
    setUmQueryError("");
    unitList.setFilters({ umQ: term || null });
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
    engineList.setFilters({ emQ: term || null });
  };

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="flex flex-row flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold">Master unit</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Model unit dan engine yang dipakai pada master unit.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <nav aria-label="Breadcrumb" className="hidden items-center gap-2 text-sm sm:flex">
              <span className="text-muted-foreground">Dashboard</span>
              <span className="text-muted-foreground" aria-hidden="true">
                /
              </span>
              <span aria-current="page">Master unit</span>
            </nav>
            <Badge variant="secondary" className="h-6 gap-1.5 px-2.5">
              <Boxes className="size-3.5" aria-hidden="true" />
              {unitTotal} unit model
            </Badge>
            <Badge variant="secondary" className="h-6 gap-1.5 px-2.5">
              <Settings2 className="size-3.5" aria-hidden="true" />
              {engineTotal} engine model
            </Badge>
          </div>
        </CardContent>
      </Card>

      {!canWrite ? (
        <Alert className="border-sky-500/30 bg-sky-500/5">
          <ShieldCheck aria-hidden="true" />
          <AlertTitle>Akses baca saja</AlertTitle>
          <AlertDescription>
            Role Anda dapat melihat master unit, tetapi tidak dapat menambah model.
          </AlertDescription>
        </Alert>
      ) : null}

      <section aria-labelledby="unit-model-heading" className="space-y-4">
        <div>
          <h2 id="unit-model-heading" className="text-base font-medium">
            Unit model
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Kombinasi manufacturer dan model harus unik.
          </p>
        </div>
        <div className="grid gap-4">
          {canWrite ? <UnitModelForm onSaved={() => void loadUnitModels()} /> : null}
          <Card className="gap-0 overflow-hidden">
            <CardHeader className="border-b">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <CardTitle>Daftar unit model</CardTitle>
                  <CardDescription>Referensi model untuk master unit.</CardDescription>
                </div>
                <form
                  role="search"
                  onSubmit={submitUnitSearch}
                  className="w-full sm:w-64"
                >
                  <label htmlFor="um-search" className="sr-only">
                    Cari unit model
                  </label>
                  <div className="relative">
                    <Search
                      className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                      aria-hidden="true"
                    />
                    <Input
                      id="um-search"
                      name="umQ"
                      type="search"
                      value={umQuery}
                      onChange={(event) => {
                        setUmQuery(event.target.value);
                        setUmQueryError("");
                      }}
                      aria-invalid={Boolean(umQueryError)}
                      aria-describedby={umQueryError ? "um-search-error" : undefined}
                      placeholder="Cari model..."
                      enterKeyHint="search"
                      autoCapitalize="none"
                      spellCheck={false}
                      className="pl-9 text-sm leading-5"
                    />
                    <button type="submit" className="sr-only">
                      Cari unit model
                    </button>
                  </div>
                  {umQueryError ? (
                    <FieldError id="um-search-error">{umQueryError}</FieldError>
                  ) : null}
                </form>
              </div>
            </CardHeader>

            {unitError ? (
              <div className="p-4">
                <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                  <CircleAlert aria-hidden="true" />
                  <AlertTitle>Master unit model tidak dapat dimuat</AlertTitle>
                  <AlertDescription>{unitError}</AlertDescription>
                </Alert>
              </div>
            ) : unitLoading && unitModels.length === 0 ? (
              <TableSkeleton />
            ) : unitModels.length === 0 ? (
              <EmptyModels label="unit model" />
            ) : (
              <div aria-busy={unitLoading}>
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

            {unitTotal > 0 ? (
              <CardFooter className="border-t">
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
              </CardFooter>
            ) : null}
          </Card>
        </div>
      </section>

      <section aria-labelledby="engine-model-heading" className="space-y-4">
        <div>
          <h2 id="engine-model-heading" className="text-base font-medium">
            Engine model
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Kombinasi manufacturer dan model harus unik.
          </p>
        </div>
        <div className="grid gap-4">
          {canWrite ? <EngineModelForm onSaved={() => void loadEngineModels()} /> : null}
          <Card className="gap-0 overflow-hidden">
            <CardHeader className="border-b">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <CardTitle>Daftar engine model</CardTitle>
                  <CardDescription>Referensi engine untuk master unit.</CardDescription>
                </div>
                <form
                  role="search"
                  onSubmit={submitEngineSearch}
                  className="w-full sm:w-64"
                >
                  <label htmlFor="em-search" className="sr-only">
                    Cari engine model
                  </label>
                  <div className="relative">
                    <Search
                      className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                      aria-hidden="true"
                    />
                    <Input
                      id="em-search"
                      name="emQ"
                      type="search"
                      value={emQuery}
                      onChange={(event) => {
                        setEmQuery(event.target.value);
                        setEmQueryError("");
                      }}
                      aria-invalid={Boolean(emQueryError)}
                      aria-describedby={emQueryError ? "em-search-error" : undefined}
                      placeholder="Cari model..."
                      enterKeyHint="search"
                      autoCapitalize="none"
                      spellCheck={false}
                      className="pl-9 text-sm leading-5"
                    />
                    <button type="submit" className="sr-only">
                      Cari engine model
                    </button>
                  </div>
                  {emQueryError ? (
                    <FieldError id="em-search-error">{emQueryError}</FieldError>
                  ) : null}
                </form>
              </div>
            </CardHeader>

            {engineError ? (
              <div className="p-4">
                <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                  <CircleAlert aria-hidden="true" />
                  <AlertTitle>Master engine model tidak dapat dimuat</AlertTitle>
                  <AlertDescription>{engineError}</AlertDescription>
                </Alert>
              </div>
            ) : engineLoading && engineModels.length === 0 ? (
              <TableSkeleton />
            ) : engineModels.length === 0 ? (
              <EmptyModels label="engine model" />
            ) : (
              <div aria-busy={engineLoading}>
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
                          {model.displacement_cc ? `${model.displacement_cc} cc` : "-"}
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

            {engineTotal > 0 ? (
              <CardFooter className="border-t">
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
              </CardFooter>
            ) : null}
          </Card>
        </div>
      </section>
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
    <Card>
      <CardHeader className="border-b">
        <CardTitle>Tambah unit model</CardTitle>
        <CardDescription>Tambahkan referensi model unit baru.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={(event) => void submit(event)} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Field>
              <FieldLabel htmlFor="um-manufacturer">
                Manufacturer <span aria-hidden="true" className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="um-manufacturer"
                name="manufacturer"
                required
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
                value={category}
                onChange={(event) => setCategory(event.target.value)}
              />
            </Field>
            <Field className="sm:col-span-2 lg:col-span-4">
              <FieldLabel htmlFor="um-notes">Catatan</FieldLabel>
              <Textarea
                id="um-notes"
                name="notes"
                rows={2}
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
              />
            </Field>
          </div>
          <label className="flex h-9 w-fit items-center gap-2 text-sm">
            <input
              type="checkbox"
              name="is_active"
              checked={active}
              onChange={(event) => setActive(event.target.checked)}
              className="size-4 accent-primary"
            />
            Aktif
          </label>
          {error ? (
            <Alert
              id="unit-model-form-error"
              tabIndex={-1}
              variant="destructive"
              className="border-destructive/30 bg-destructive/5"
            >
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Gagal menyimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
          {message ? (
            <Alert
              role="status"
              className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300"
            >
              <CircleCheck aria-hidden="true" />
              <AlertTitle>Berhasil</AlertTitle>
              <AlertDescription className="text-emerald-700 dark:text-emerald-300">
                {message}
              </AlertDescription>
            </Alert>
          ) : null}
          <Button type="submit" disabled={pending}>
            {pending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                Menyimpan...
              </>
            ) : (
              "Tambah unit model"
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
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
    <Card>
      <CardHeader className="border-b">
        <CardTitle>Tambah engine model</CardTitle>
        <CardDescription>Tambahkan referensi engine baru.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={(event) => void submit(event)} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Field>
              <FieldLabel htmlFor="em-manufacturer">
                Manufacturer <span aria-hidden="true" className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="em-manufacturer"
                name="manufacturer"
                required
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
                step="1"
                value={displacement}
                onChange={(event) => setDisplacement(event.target.value)}
              />
            </Field>
            <Field className="sm:col-span-2 lg:col-span-4">
              <FieldLabel htmlFor="em-notes">Catatan</FieldLabel>
              <Textarea
                id="em-notes"
                name="notes"
                rows={2}
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
              />
            </Field>
          </div>
          <label className="flex h-9 w-fit items-center gap-2 text-sm">
            <input
              type="checkbox"
              name="is_active"
              checked={active}
              onChange={(event) => setActive(event.target.checked)}
              className="size-4 accent-primary"
            />
            Aktif
          </label>
          {error ? (
            <Alert
              id="engine-model-form-error"
              tabIndex={-1}
              variant="destructive"
              className="border-destructive/30 bg-destructive/5"
            >
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Gagal menyimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
          {message ? (
            <Alert
              role="status"
              className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300"
            >
              <CircleCheck aria-hidden="true" />
              <AlertTitle>Berhasil</AlertTitle>
              <AlertDescription className="text-emerald-700 dark:text-emerald-300">
                {message}
              </AlertDescription>
            </Alert>
          ) : null}
          <Button type="submit" disabled={pending}>
            {pending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                Menyimpan...
              </>
            ) : (
              "Tambah engine model"
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

function TableSkeleton() {
  return (
    <div className="space-y-2 px-6 py-4" role="status" aria-label="Memuat master model">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-12 w-full" />
      ))}
    </div>
  );
}

function EmptyModels({ label }: { label: string }) {
  return (
    <CardContent className="py-10">
      <Empty className="border-0">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Boxes aria-hidden="true" />
          </EmptyMedia>
          <EmptyTitle>Belum ada {label}.</EmptyTitle>
          <EmptyDescription>
            Tambahkan referensi model agar dapat dipilih saat membuat unit.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    </CardContent>
  );
}
