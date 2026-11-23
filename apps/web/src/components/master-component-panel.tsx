"use client";

import {
  CircleAlert,
  CircleCheck,
  Cog,
  Filter,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import { FilterBar } from "@/components/list/list-filter-bar";
import { SearchInput } from "@/components/list/list-search";
import { ActiveBadge } from "@/components/status-badges";
import { TablePagination } from "@/components/table-pagination";
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
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
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
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError, proxyGet, proxyPatch, proxyPost } from "@/lib/api-client";
import type { ComponentOption, DataResponse } from "@/lib/types";
import { formatNumber } from "@/lib/time";

type ComponentFilters = {
  q: string;
  activeOnly: boolean;
};

const emptyFilters: ComponentFilters = { q: "", activeOnly: false };

export function MasterComponentPanel() {
  const { user } = useSession();
  const [items, setItems] = useState<ComponentOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [filters, setFilters] = useState<ComponentFilters>(emptyFilters);
  const [query, setQuery] = useState<ComponentFilters>(emptyFilters);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<ComponentOption | null>(null);

  const canWrite = user.role === "admin" || user.role === "planner";

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await proxyGet<DataResponse<ComponentOption[]>>("/components");
      setItems(result.data);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Master komponen tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const filtered = useMemo(() => {
    const term = query.q.trim().toLowerCase();
    return items.filter((item) => {
      if (query.activeOnly && !item.is_active) return false;
      if (!term) return true;
      return [item.code, item.name, item.category, item.component_class]
        .join(" ")
        .toLowerCase()
        .includes(term);
    });
  }, [items, query]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const pageItems = filtered.slice((page - 1) * pageSize, page * pageSize);

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

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (item: ComponentOption) => {
    setEditing(item);
    setFormOpen(true);
  };

  const handleSaved = (nextMessage: string) => {
    setMessage(nextMessage);
    setFormOpen(false);
    setEditing(null);
    void load();
  };

  return (
    <div className="flex flex-col gap-4">
      {message ? (
        <Alert role="status" className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300">
          <CircleCheck aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription className="text-emerald-700 dark:text-emerald-300">
            {message}
          </AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Cog className="size-4 text-muted-foreground" aria-hidden="true" />
                Master komponen
              </CardTitle>
              <CardDescription>
                Referensi komponen, kelas, dan lifetime default untuk instalasi unit.
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="secondary" className="h-6 px-2.5">
                {filtered.length} komponen
              </Badge>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void load()}
                disabled={loading}
              >
                <RefreshCw className={loading ? "animate-spin" : undefined} aria-hidden="true" />
                Muat ulang
              </Button>
              {canWrite ? (
                <Button type="button" size="sm" onClick={openCreate}>
                  <Plus aria-hidden="true" /> Tambah komponen
                </Button>
              ) : null}
            </div>
          </div>

          <FilterBar
            role="search"
            aria-label="Cari master komponen"
            onSubmit={applyFilters}
            className="mt-2 grid gap-3 md:grid-cols-2 xl:grid-cols-4"
          >
            <Field className="xl:col-span-2">
              <FieldLabel htmlFor="component-search">Cari komponen</FieldLabel>
              <SearchInput
                id="component-search"
                value={filters.q}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, q: event.target.value }))
                }
                placeholder="Kode, nama, kategori, atau kelas..."
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="component-active">Status</FieldLabel>
              <NativeSelect
                id="component-active"
                value={filters.activeOnly ? "1" : ""}
                onChange={(event) =>
                  setFilters((current) => ({
                    ...current,
                    activeOnly: event.target.value === "1",
                  }))
                }
              >
                <NativeSelectOption value="">Semua status</NativeSelectOption>
                <NativeSelectOption value="1">Hanya aktif</NativeSelectOption>
              </NativeSelect>
            </Field>
            <div className="flex items-end gap-2">
              <Button type="submit">
                <Filter aria-hidden="true" /> Terapkan
              </Button>
              <Button type="button" variant="outline" onClick={resetFilters}>
                <RotateCcw aria-hidden="true" /> Reset
              </Button>
            </div>
          </FilterBar>
        </CardHeader>

        <CardContent className="flex flex-col gap-3">
          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Master komponen tidak dapat dimuat</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          {loading && items.length === 0 ? (
            <div className="flex flex-col gap-2">
              {Array.from({ length: 5 }).map((_, index) => (
                <Skeleton key={index} className="h-12 w-full" />
              ))}
            </div>
          ) : filtered.length === 0 ? (
            <Empty className="border-0 py-8">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Cog aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada komponen.</EmptyTitle>
                <EmptyDescription>
                  {query.q || query.activeOnly
                    ? "Tidak ada komponen yang cocok dengan filter."
                    : "Tambahkan komponen agar dapat dipasang pada unit."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="overflow-x-auto" aria-busy={loading}>
              <Table>
                <TableCaption className="sr-only">Daftar master komponen</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>Komponen</TableHead>
                    <TableHead>Kategori</TableHead>
                    <TableHead>Kelas</TableHead>
                    <TableHead>Lifetime default</TableHead>
                    <TableHead>Status</TableHead>
                    {canWrite ? <TableHead className="text-right">Aksi</TableHead> : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="min-w-56">
                        <p className="font-medium">{item.name}</p>
                        <p className="font-mono text-xs text-muted-foreground">{item.code}</p>
                      </TableCell>
                      <TableCell className="text-sm">{item.category || "-"}</TableCell>
                      <TableCell>
                        {item.component_class ? (
                          <Badge variant="outline" className="h-5 px-2 capitalize">
                            {item.component_class}
                          </Badge>
                        ) : (
                          <span className="text-sm text-muted-foreground">-</span>
                        )}
                      </TableCell>
                      <TableCell className="min-w-48 whitespace-nowrap text-sm">
                        <p>
                          HM {item.default_lifetime_hm !== undefined ? formatNumber(item.default_lifetime_hm) : "-"}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {item.default_lifetime_days !== undefined
                            ? `${formatNumber(item.default_lifetime_days)} hari`
                            : "Hari -"}
                          {" · "}
                          KM {item.default_lifetime_km !== undefined ? formatNumber(item.default_lifetime_km) : "-"}
                        </p>
                      </TableCell>
                      <TableCell>
                        <ActiveBadge active={item.is_active} />
                      </TableCell>
                      {canWrite ? (
                        <TableCell className="text-right">
                          <Button
                            type="button"
                            size="xs"
                            variant="outline"
                            onClick={() => openEdit(item)}
                          >
                            <Pencil aria-hidden="true" /> Edit
                          </Button>
                        </TableCell>
                      ) : null}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          <TablePagination
            page={page}
            pageSize={pageSize}
            total={filtered.length}
            totalPages={totalPages}
            loading={loading}
            itemLabel="komponen"
            onPageChange={setPage}
            onPageSizeChange={(value) => {
              setPageSize(value);
              setPage(1);
            }}
          />
        </CardContent>
      </Card>

      {formOpen ? (
        <ComponentFormSheet
          key={editing?.id ?? "new"}
          component={editing}
          onOpenChange={(open) => {
            setFormOpen(open);
            if (!open) setEditing(null);
          }}
          onSaved={handleSaved}
        />
      ) : null}
    </div>
  );
}

function ComponentFormSheet({
  component,
  onOpenChange,
  onSaved,
}: {
  component: ComponentOption | null;
  onOpenChange: (open: boolean) => void;
  onSaved: (message: string) => void;
}) {
  const [code, setCode] = useState(component?.code ?? "");
  const [name, setName] = useState(component?.name ?? "");
  const [category, setCategory] = useState(component?.category ?? "");
  const [componentClass, setComponentClass] = useState(component?.component_class ?? "");
  const [lifetimeHM, setLifetimeHM] = useState(
    component?.default_lifetime_hm?.toString() ?? "",
  );
  const [lifetimeDays, setLifetimeDays] = useState(
    component?.default_lifetime_days?.toString() ?? "",
  );
  const [lifetimeKM, setLifetimeKM] = useState(
    component?.default_lifetime_km?.toString() ?? "",
  );
  const [isActive, setIsActive] = useState(component?.is_active ?? true);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    const nextErrors: Record<string, string> = {};
    if (!code.trim()) nextErrors.code = "Kode komponen wajib diisi";
    if (!name.trim()) nextErrors.name = "Nama komponen wajib diisi";
    const lifeHM = parseOptionalNonNegative(lifetimeHM, "default_lifetime_hm", nextErrors);
    const lifeKM = parseOptionalNonNegative(lifetimeKM, "default_lifetime_km", nextErrors);
    const lifeDays = parseOptionalNonNegativeInt(lifetimeDays, "default_lifetime_days", nextErrors);
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    const body = {
      code: code.trim(),
      name: name.trim(),
      category: category.trim(),
      component_class: componentClass,
      default_lifetime_hm: lifeHM,
      default_lifetime_days: lifeDays,
      default_lifetime_km: lifeKM,
      is_active: isActive,
    };

    setPending(true);
    try {
      if (component) {
        await proxyPatch<DataResponse<ComponentOption>>(`/components/${component.id}`, body);
        onSaved("Komponen berhasil diperbarui.");
      } else {
        await proxyPost<DataResponse<ComponentOption>>("/components", body);
        onSaved("Komponen berhasil ditambahkan.");
      }
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
      } else {
        setError("Komponen tidak dapat disimpan.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Sheet open onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{component ? "Edit komponen" : "Tambah komponen"}</SheetTitle>
          <SheetDescription>
            Master komponen dipakai sebagai default lifetime saat pemasangan di unit.
          </SheetDescription>
        </SheetHeader>

        <form className="grid gap-4 px-4" onSubmit={(event) => void submit(event)} noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.code)}>
              <FieldLabel htmlFor="component-code">
                Kode <span aria-hidden="true" className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="component-code"
                value={code}
                onChange={(event) => setCode(event.target.value)}
                placeholder="ENG-01"
              />
              {fieldErrors.code ? <FieldError>{fieldErrors.code}</FieldError> : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.name)}>
              <FieldLabel htmlFor="component-name">
                Nama <span aria-hidden="true" className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="component-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Engine"
              />
              {fieldErrors.name ? <FieldError>{fieldErrors.name}</FieldError> : null}
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="component-category">Kategori</FieldLabel>
              <Input
                id="component-category"
                value={category}
                onChange={(event) => setCategory(event.target.value)}
                placeholder="Engine"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="component-class">Kelas</FieldLabel>
              <NativeSelect
                id="component-class"
                value={componentClass}
                onChange={(event) => setComponentClass(event.target.value)}
              >
                <NativeSelectOption value="">Tidak ditentukan</NativeSelectOption>
                <NativeSelectOption value="major">Major</NativeSelectOption>
                <NativeSelectOption value="minor">Minor</NativeSelectOption>
              </NativeSelect>
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.default_lifetime_hm)}>
              <FieldLabel htmlFor="component-lifetime-hm">Lifetime HM</FieldLabel>
              <Input
                id="component-lifetime-hm"
                type="number"
                min="0"
                step="0.1"
                value={lifetimeHM}
                onChange={(event) => setLifetimeHM(event.target.value)}
              />
              {fieldErrors.default_lifetime_hm ? (
                <FieldError>{fieldErrors.default_lifetime_hm}</FieldError>
              ) : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.default_lifetime_days)}>
              <FieldLabel htmlFor="component-lifetime-days">Lifetime hari</FieldLabel>
              <Input
                id="component-lifetime-days"
                type="number"
                min="0"
                step="1"
                value={lifetimeDays}
                onChange={(event) => setLifetimeDays(event.target.value)}
              />
              {fieldErrors.default_lifetime_days ? (
                <FieldError>{fieldErrors.default_lifetime_days}</FieldError>
              ) : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.default_lifetime_km)}>
              <FieldLabel htmlFor="component-lifetime-km">Lifetime KM</FieldLabel>
              <Input
                id="component-lifetime-km"
                type="number"
                min="0"
                step="0.1"
                value={lifetimeKM}
                onChange={(event) => setLifetimeKM(event.target.value)}
              />
              {fieldErrors.default_lifetime_km ? (
                <FieldError>{fieldErrors.default_lifetime_km}</FieldError>
              ) : null}
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="component-status">Status</FieldLabel>
            <NativeSelect
              id="component-status"
              value={isActive ? "1" : "0"}
              onChange={(event) => setIsActive(event.target.value === "1")}
            >
              <NativeSelectOption value="1">Aktif</NativeSelectOption>
              <NativeSelectOption value="0">Nonaktif</NativeSelectOption>
            </NativeSelect>
            <FieldDescription>
              Komponen nonaktif tidak dapat dipilih untuk instalasi baru.
            </FieldDescription>
          </Field>

          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Komponen tidak dapat disimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <SheetFooter className="px-0 sm:flex-row sm:justify-end">
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={pending}
            >
              Batal
            </Button>
            <Button type="submit" size="sm" disabled={pending}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" /> Menyimpan...
                </>
              ) : component ? (
                "Simpan perubahan"
              ) : (
                "Tambah komponen"
              )}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}

function parseOptionalNonNegative(
  value: string,
  field: string,
  errors: Record<string, string>,
): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || parsed < 0) {
    errors[field] = "Nilai harus angka dan tidak negatif";
    return undefined;
  }
  return parsed;
}

function parseOptionalNonNegativeInt(
  value: string,
  field: string,
  errors: Record<string, string>,
): number | undefined {
  const parsed = parseOptionalNonNegative(value, field, errors);
  if (parsed === undefined) return undefined;
  if (!Number.isInteger(parsed)) {
    errors[field] = "Nilai harus bilangan bulat";
    return undefined;
  }
  return parsed;
}
