"use client";

import {
  CheckCircle2,
  CircleAlert,
  GitCompareArrows,
  Layers3,
  Loader2,
  PackageSearch,
  Pencil,
  Plus,
  Search,
  ShieldCheck,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
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
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
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
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { ApiError, proxyGet, proxyPatch, proxyPost } from "@/lib/api-client";
import type {
  CompatibilityStatus,
  DataResponse,
  EngineModel,
  ListResponse,
  Sparepart,
  SparepartAlternative,
  SparepartCompatibility,
  SparepartNumber,
  UnitModel,
} from "@/lib/types";
import { formatWita } from "@/lib/time";
import { cn } from "@/lib/utils";

const compatibilityLabels: Record<CompatibilityStatus, string> = {
  pending_verification: "Pending Verification",
  approved_compatible: "Approved Compatible",
  conditional_compatible: "Conditional Compatible",
  not_compatible: "Not Compatible",
};

const numberTypeLabels: Record<string, string> = {
  oem: "OEM",
  aftermarket: "Aftermarket",
  superseded: "Superseded",
};

const relationLabels: Record<string, string> = {
  equivalent: "Equivalent",
  superseded: "Superseded",
  substitute: "Substitute",
};

type SparepartForm = {
  code: string;
  name: string;
  category: string;
  uom: string;
  description: string;
  is_active: boolean;
};

const emptySparepartForm: SparepartForm = {
  code: "",
  name: "",
  category: "",
  uom: "pcs",
  description: "",
  is_active: true,
};

type NumberForm = {
  part_number: string;
  brand: string;
  number_type: string;
  is_primary: boolean;
  notes: string;
};

const emptyNumberForm: NumberForm = {
  part_number: "",
  brand: "",
  number_type: "oem",
  is_primary: false,
  notes: "",
};

type AlternativeForm = {
  alternative_sparepart_id: string;
  relation_type: string;
  notes: string;
};

const emptyAlternativeForm: AlternativeForm = {
  alternative_sparepart_id: "",
  relation_type: "equivalent",
  notes: "",
};

type CompatibilityForm = {
  unit_model_id: string;
  engine_model_id: string;
  configuration: string;
  source: string;
  notes: string;
};

const emptyCompatibilityForm: CompatibilityForm = {
  unit_model_id: "",
  engine_model_id: "",
  configuration: "",
  source: "",
  notes: "",
};

type DetailTab = "numbers" | "alternatives" | "compatibility";

type VerifyState = {
  kind: "alternative" | "compatibility";
  id: number;
  label: string;
  status: CompatibilityStatus;
  condition_note: string;
  notes: string;
};

export function SparepartsView() {
  const { user } = useSession();
  const canManage = user.role === "admin" || user.role === "planner";

  const [items, setItems] = useState<Sparepart[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [searchInput, setSearchInput] = useState("");
  const [query, setQuery] = useState("");
  const [activeOnly, setActiveOnly] = useState(true);
  const [statusFilter, setStatusFilter] = useState("");
  const [unitModelFilter, setUnitModelFilter] = useState("");
  const [engineModelFilter, setEngineModelFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const [unitModels, setUnitModels] = useState<UnitModel[]>([]);
  const [engineModels, setEngineModels] = useState<EngineModel[]>([]);
  const [sparepartOptions, setSparepartOptions] = useState<Sparepart[]>([]);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<Sparepart | null>(null);
  const [detailTab, setDetailTab] = useState<DetailTab>("numbers");
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");

  const [formOpen, setFormOpen] = useState(false);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [form, setForm] = useState<SparepartForm>(emptySparepartForm);
  const [formBusy, setFormBusy] = useState(false);
  const [formError, setFormError] = useState("");

  const [numberForm, setNumberForm] = useState<NumberForm>(emptyNumberForm);
  const [numberBusy, setNumberBusy] = useState(false);
  const [numberError, setNumberError] = useState("");

  const [alternativeForm, setAlternativeForm] = useState<AlternativeForm>(emptyAlternativeForm);
  const [alternativeBusy, setAlternativeBusy] = useState(false);
  const [alternativeError, setAlternativeError] = useState("");

  const [compatibilityForm, setCompatibilityForm] = useState<CompatibilityForm>(emptyCompatibilityForm);
  const [compatibilityBusy, setCompatibilityBusy] = useState(false);
  const [compatibilityError, setCompatibilityError] = useState("");

  const [verify, setVerify] = useState<VerifyState | null>(null);
  const [verifyBusy, setVerifyBusy] = useState(false);
  const [verifyError, setVerifyError] = useState("");

  const loadList = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: String(pageSize),
        sort: "code",
        order: "asc",
      });
      if (query.trim()) params.set("q", query.trim());
      if (activeOnly) params.set("active_only", "true");
      if (statusFilter) params.set("compatibility_status", statusFilter);
      if (unitModelFilter) params.set("unit_model_id", unitModelFilter);
      if (engineModelFilter) params.set("engine_model_id", engineModelFilter);
      const result = await proxyGet<ListResponse<Sparepart>>(`/spareparts?${params.toString()}`);
      setItems(result.data);
      setTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Sparepart tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, [activeOnly, engineModelFilter, page, pageSize, query, statusFilter, unitModelFilter]);

  const loadReferences = useCallback(async () => {
    try {
      const [models, engines, spareparts] = await Promise.all([
        proxyGet<ListResponse<UnitModel>>("/unit-models?limit=100&active_only=true&sort=manufacturer&order=asc"),
        proxyGet<ListResponse<EngineModel>>("/engine-models?limit=100&active_only=true&sort=manufacturer&order=asc"),
        proxyGet<ListResponse<Sparepart>>("/spareparts?limit=100&active_only=true&sort=code&order=asc"),
      ]);
      setUnitModels(models.data);
      setEngineModels(engines.data);
      setSparepartOptions(spareparts.data);
    } catch {
      // Reference data is best-effort.
    }
  }, []);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  useEffect(() => {
    void loadReferences();
  }, [loadReferences]);

  const refreshDetail = useCallback(async (id: number) => {
    setDetailLoading(true);
    setDetailError("");
    try {
      const result = await proxyGet<DataResponse<Sparepart>>(`/spareparts/${id}`);
      setDetail(result.data);
      return result.data;
    } catch (loadError) {
      setDetailError(loadError instanceof ApiError ? loadError.message : "Detail sparepart tidak dapat dimuat.");
      return null;
    } finally {
      setDetailLoading(false);
    }
  }, []);

  const openDetail = async (sparepart: Sparepart) => {
    setDetail(sparepart);
    setDetailOpen(true);
    setDetailTab("numbers");
    setNumberError("");
    setAlternativeError("");
    setCompatibilityError("");
    setNumberForm(emptyNumberForm);
    setAlternativeForm(emptyAlternativeForm);
    setCompatibilityForm(emptyCompatibilityForm);
    await refreshDetail(sparepart.id);
  };

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPage(1);
    setQuery(searchInput);
  };

  const resetFilters = () => {
    setSearchInput("");
    setQuery("");
    setActiveOnly(true);
    setStatusFilter("");
    setUnitModelFilter("");
    setEngineModelFilter("");
    setPage(1);
  };

  const openCreate = () => {
    setEditingID(null);
    setForm(emptySparepartForm);
    setFormError("");
    setFormOpen(true);
  };

  const openEdit = (sparepart: Sparepart) => {
    setEditingID(sparepart.id);
    setForm({
      code: sparepart.code,
      name: sparepart.name,
      category: sparepart.category,
      uom: sparepart.uom,
      description: sparepart.description,
      is_active: sparepart.is_active,
    });
    setFormError("");
    setFormOpen(true);
  };

  const submitForm = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (formBusy) return;
    setFormBusy(true);
    setFormError("");
    try {
      const payload = {
        code: form.code.trim(),
        name: form.name.trim(),
        category: form.category.trim(),
        uom: form.uom.trim() || "pcs",
        description: form.description.trim(),
        is_active: form.is_active,
      };
      const result = editingID
        ? await proxyPatch<DataResponse<Sparepart>>(`/spareparts/${editingID}`, payload)
        : await proxyPost<DataResponse<Sparepart>>("/spareparts", payload);
      setSuccess(editingID ? `Sparepart ${result.data.code} diperbarui.` : `Sparepart ${result.data.code} dibuat.`);
      setFormOpen(false);
      await loadList();
      await loadReferences();
      if (detail?.id === result.data.id) {
        await refreshDetail(result.data.id);
      }
    } catch (saveError) {
      setFormError(saveError instanceof ApiError ? saveError.message : "Sparepart tidak dapat disimpan.");
    } finally {
      setFormBusy(false);
    }
  };

  const addNumber = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!detail || numberBusy) return;
    setNumberBusy(true);
    setNumberError("");
    try {
      await proxyPost<DataResponse<SparepartNumber>>(`/spareparts/${detail.id}/numbers`, {
        part_number: numberForm.part_number.trim(),
        brand: numberForm.brand.trim(),
        number_type: numberForm.number_type,
        is_primary: numberForm.is_primary,
        notes: numberForm.notes.trim(),
      });
      setNumberForm(emptyNumberForm);
      setSuccess("Nomor part ditambahkan.");
      await refreshDetail(detail.id);
      await loadList();
    } catch (saveError) {
      setNumberError(saveError instanceof ApiError ? saveError.message : "Nomor part tidak dapat disimpan.");
    } finally {
      setNumberBusy(false);
    }
  };

  const setPrimaryNumber = async (number: SparepartNumber) => {
    if (!detail) return;
    setNumberError("");
    try {
      await proxyPatch<DataResponse<SparepartNumber>>(`/spareparts/${detail.id}/numbers/${number.id}`, {
        is_primary: true,
      });
      setSuccess(`Nomor ${number.part_number} menjadi nomor utama.`);
      await refreshDetail(detail.id);
      await loadList();
    } catch (saveError) {
      setNumberError(saveError instanceof ApiError ? saveError.message : "Nomor utama tidak dapat diubah.");
    }
  };

  const addAlternative = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!detail || alternativeBusy) return;
    if (!alternativeForm.alternative_sparepart_id) {
      setAlternativeError("Pilih sparepart alternatif.");
      return;
    }
    setAlternativeBusy(true);
    setAlternativeError("");
    try {
      await proxyPost<DataResponse<SparepartAlternative>>(`/spareparts/${detail.id}/alternatives`, {
        alternative_sparepart_id: Number(alternativeForm.alternative_sparepart_id),
        relation_type: alternativeForm.relation_type,
        notes: alternativeForm.notes.trim(),
      });
      setAlternativeForm(emptyAlternativeForm);
      setSuccess("Alternatif ditambahkan dan menunggu verifikasi.");
      await refreshDetail(detail.id);
    } catch (saveError) {
      setAlternativeError(saveError instanceof ApiError ? saveError.message : "Alternatif tidak dapat disimpan.");
    } finally {
      setAlternativeBusy(false);
    }
  };

  const addCompatibility = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!detail || compatibilityBusy) return;
    if (!compatibilityForm.unit_model_id && !compatibilityForm.engine_model_id && !compatibilityForm.configuration.trim()) {
      setCompatibilityError("Minimal satu target compatibility wajib diisi.");
      return;
    }
    setCompatibilityBusy(true);
    setCompatibilityError("");
    try {
      await proxyPost<DataResponse<SparepartCompatibility>>(`/spareparts/${detail.id}/compatibility`, {
        unit_model_id: compatibilityForm.unit_model_id ? Number(compatibilityForm.unit_model_id) : null,
        engine_model_id: compatibilityForm.engine_model_id ? Number(compatibilityForm.engine_model_id) : null,
        configuration: compatibilityForm.configuration.trim(),
        source: compatibilityForm.source.trim(),
        notes: compatibilityForm.notes.trim(),
      });
      setCompatibilityForm(emptyCompatibilityForm);
      setSuccess("Compatibility ditambahkan dan menunggu verifikasi.");
      await refreshDetail(detail.id);
      await loadList();
    } catch (saveError) {
      setCompatibilityError(saveError instanceof ApiError ? saveError.message : "Compatibility tidak dapat disimpan.");
    } finally {
      setCompatibilityBusy(false);
    }
  };

  const openVerifyAlternative = (alternative: SparepartAlternative) => {
    setVerify({
      kind: "alternative",
      id: alternative.id,
      label: `${alternative.alternative_sparepart_code} - ${alternative.alternative_sparepart_name}`,
      status: alternative.verification_status === "pending_verification" ? "approved_compatible" : alternative.verification_status,
      condition_note: "",
      notes: alternative.notes,
    });
    setVerifyError("");
  };

  const openVerifyCompatibility = (compatibility: SparepartCompatibility) => {
    setVerify({
      kind: "compatibility",
      id: compatibility.id,
      label: compatibility.unit_model_name || compatibility.engine_model_name || compatibility.configuration,
      status: compatibility.status === "pending_verification" ? "approved_compatible" : compatibility.status,
      condition_note: compatibility.condition_note,
      notes: compatibility.notes,
    });
    setVerifyError("");
  };

  const submitVerify = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!detail || !verify || verifyBusy) return;
    setVerifyBusy(true);
    setVerifyError("");
    try {
      if (verify.kind === "alternative") {
        await proxyPost<DataResponse<SparepartAlternative>>(
          `/spareparts/${detail.id}/alternatives/${verify.id}/verify`,
          { status: verify.status, notes: verify.notes.trim() },
        );
      } else {
        await proxyPost<DataResponse<SparepartCompatibility>>(
          `/spareparts/${detail.id}/compatibility/${verify.id}/verify`,
          { status: verify.status, condition_note: verify.condition_note.trim(), notes: verify.notes.trim() },
        );
      }
      setSuccess("Verifikasi tersimpan.");
      setVerify(null);
      await refreshDetail(detail.id);
      await loadList();
    } catch (saveError) {
      setVerifyError(saveError instanceof ApiError ? saveError.message : "Verifikasi tidak dapat disimpan.");
    } finally {
      setVerifyBusy(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Sparepart</h1>
          <p className="text-sm text-muted-foreground">
            Master sparepart, nomor OEM/aftermarket, alternatif, dan compatibility tanpa data harga.
          </p>
        </div>
        {canManage ? (
          <Button type="button" onClick={openCreate}>
            <Plus aria-hidden="true" /> Tambah sparepart
          </Button>
        ) : null}
      </div>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal</AlertTitle>
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
        <CardHeader className="gap-3">
          <div>
            <CardTitle className="flex items-center gap-2">
              <PackageSearch className="size-4" aria-hidden="true" /> Pencarian sparepart
            </CardTitle>
            <CardDescription>
              Cari kode, nama, nomor part, brand, unit model, engine model, atau status compatibility.
            </CardDescription>
          </div>
          <form className="grid gap-3 md:grid-cols-2 xl:grid-cols-6" onSubmit={submitSearch}>
            <Field className="xl:col-span-2">
              <FieldLabel htmlFor="sparepart-search">Kata kunci</FieldLabel>
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <Input
                  id="sparepart-search"
                  value={searchInput}
                  onChange={(event) => setSearchInput(event.target.value)}
                  placeholder="Kode, nama, part number, brand..."
                  className="pl-9"
                />
              </div>
            </Field>
            <Field>
              <FieldLabel htmlFor="sparepart-status">Compatibility</FieldLabel>
              <NativeSelect id="sparepart-status" value={statusFilter} onChange={(event) => { setStatusFilter(event.target.value); setPage(1); }}>
                <NativeSelectOption value="">Semua status</NativeSelectOption>
                {Object.entries(compatibilityLabels).map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field>
              <FieldLabel htmlFor="sparepart-unit-model">Unit model</FieldLabel>
              <NativeSelect id="sparepart-unit-model" value={unitModelFilter} onChange={(event) => { setUnitModelFilter(event.target.value); setPage(1); }}>
                <NativeSelectOption value="">Semua model</NativeSelectOption>
                {unitModels.map((model) => (
                  <NativeSelectOption key={model.id} value={model.id}>
                    {model.manufacturer} {model.model}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field>
              <FieldLabel htmlFor="sparepart-engine-model">Engine model</FieldLabel>
              <NativeSelect id="sparepart-engine-model" value={engineModelFilter} onChange={(event) => { setEngineModelFilter(event.target.value); setPage(1); }}>
                <NativeSelectOption value="">Semua engine</NativeSelectOption>
                {engineModels.map((model) => (
                  <NativeSelectOption key={model.id} value={model.id}>
                    {model.manufacturer} {model.model}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <div className="flex items-end gap-2">
              <Button type="submit"><Search aria-hidden="true" /> Cari</Button>
              <Button type="button" variant="outline" onClick={resetFilters}>Reset</Button>
            </div>
            <label className="flex items-center gap-2 text-sm md:col-span-2 xl:col-span-6">
              <input
                type="checkbox"
                checked={activeOnly}
                onChange={(event) => { setActiveOnly(event.target.checked); setPage(1); }}
                className="size-4 rounded border-border"
              />
              Hanya sparepart aktif
            </label>
          </form>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {loading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : items.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon"><PackageSearch aria-hidden="true" /></EmptyMedia>
                <EmptyTitle>Tidak ada sparepart</EmptyTitle>
                <EmptyDescription>Ubah filter atau tambahkan master sparepart baru.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Kode</TableHead>
                    <TableHead>Nama</TableHead>
                    <TableHead>Kategori</TableHead>
                    <TableHead>UOM</TableHead>
                    <TableHead>Nomor utama</TableHead>
                    <TableHead>Compatibility</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>
                        <button
                          type="button"
                          className="cursor-pointer font-medium text-left hover:underline"
                          onClick={() => void openDetail(item)}
                        >
                          {item.code}
                        </button>
                      </TableCell>
                      <TableCell className="max-w-72">
                        <button type="button" className="cursor-pointer text-left hover:underline" onClick={() => void openDetail(item)}>
                          {item.name}
                        </button>
                      </TableCell>
                      <TableCell>{item.category || "-"}</TableCell>
                      <TableCell>{item.uom}</TableCell>
                      <TableCell className="font-mono text-xs">{item.primary_part_number || "-"}</TableCell>
                      <TableCell>
                        <span className="text-sm">{item.approved_compatibility_count} approved / {item.compatibility_count} total</span>
                      </TableCell>
                      <TableCell><ActiveBadge active={item.is_active} /></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
          <TablePagination
            page={page}
            pageSize={pageSize}
            total={total}
            totalPages={totalPages}
            loading={loading}
            itemLabel="sparepart"
            onPageChange={setPage}
            onPageSizeChange={(size) => { setPageSize(size); setPage(1); }}
          />
        </CardContent>
      </Card>

      <Sheet open={formOpen} onOpenChange={setFormOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>{editingID ? "Edit sparepart" : "Tambah sparepart"}</SheetTitle>
            <SheetDescription>Master ini tidak menyimpan harga, cost, vendor price, atau data procurement.</SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitForm}>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="sparepart-code">Kode internal</FieldLabel>
                <Input id="sparepart-code" value={form.code} maxLength={60} onChange={(event) => setForm((current) => ({ ...current, code: event.target.value }))} required />
              </Field>
              <Field>
                <FieldLabel htmlFor="sparepart-uom">UOM</FieldLabel>
                <Input id="sparepart-uom" value={form.uom} maxLength={30} onChange={(event) => setForm((current) => ({ ...current, uom: event.target.value }))} required />
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="sparepart-name">Nama</FieldLabel>
              <Input id="sparepart-name" value={form.name} maxLength={200} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} required />
            </Field>
            <Field>
              <FieldLabel htmlFor="sparepart-category">Kategori</FieldLabel>
              <Input id="sparepart-category" value={form.category} maxLength={100} onChange={(event) => setForm((current) => ({ ...current, category: event.target.value }))} />
            </Field>
            <Field>
              <FieldLabel htmlFor="sparepart-description">Deskripsi</FieldLabel>
              <Textarea id="sparepart-description" value={form.description} maxLength={2000} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} rows={4} />
            </Field>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={form.is_active} onChange={(event) => setForm((current) => ({ ...current, is_active: event.target.checked }))} className="size-4 rounded border-border" />
              Sparepart aktif
            </label>
            {formError ? <FieldError>{formError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" disabled={formBusy}>
                {formBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                Simpan
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-5xl">
          <SheetHeader>
            <SheetTitle>{detail ? `${detail.code} · ${detail.name}` : "Detail sparepart"}</SheetTitle>
            <SheetDescription>
              {detail ? `${detail.category || "Tanpa kategori"} · ${detail.uom} · ${detail.is_active ? "Aktif" : "Nonaktif"}` : "Memuat detail..."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex flex-col gap-4 px-4 pb-4">
            {detailError ? <FieldError>{detailError}</FieldError> : null}
            {detailLoading && !detail ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : detail ? (
              <>
                <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border p-3">
                  <div className="text-sm text-muted-foreground">
                    <p>{detail.description || "Belum ada deskripsi."}</p>
                    <p className="mt-1">Dibuat {formatWita(detail.created_at)}</p>
                  </div>
                  {canManage ? (
                    <Button type="button" size="sm" variant="outline" onClick={() => openEdit(detail)}>
                      <Pencil aria-hidden="true" /> Edit master
                    </Button>
                  ) : null}
                </div>

                <div className="flex flex-wrap gap-2" role="tablist" aria-label="Detail sparepart">
                  <Button type="button" size="sm" variant={detailTab === "numbers" ? "default" : "outline"} role="tab" aria-selected={detailTab === "numbers"} onClick={() => setDetailTab("numbers")}>
                    <Layers3 aria-hidden="true" /> Nomor part ({detail.numbers.length})
                  </Button>
                  <Button type="button" size="sm" variant={detailTab === "alternatives" ? "default" : "outline"} role="tab" aria-selected={detailTab === "alternatives"} onClick={() => setDetailTab("alternatives")}>
                    <GitCompareArrows aria-hidden="true" /> Alternatif ({detail.alternatives.length})
                  </Button>
                  <Button type="button" size="sm" variant={detailTab === "compatibility" ? "default" : "outline"} role="tab" aria-selected={detailTab === "compatibility"} onClick={() => setDetailTab("compatibility")}>
                    <ShieldCheck aria-hidden="true" /> Compatibility ({detail.compatibility.length})
                  </Button>
                </div>

                {detailTab === "numbers" ? (
                  <div className="flex flex-col gap-3">
                    {detail.numbers.length === 0 ? (
                      <p className="text-sm text-muted-foreground">Belum ada nomor OEM/aftermarket.</p>
                    ) : (
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Nomor part</TableHead>
                              <TableHead>Brand</TableHead>
                              <TableHead>Tipe</TableHead>
                              <TableHead>Primary</TableHead>
                              <TableHead>Catatan</TableHead>
                              <TableHead>Aksi</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {detail.numbers.map((number) => (
                              <TableRow key={number.id}>
                                <TableCell className="font-mono text-xs">{number.part_number}</TableCell>
                                <TableCell>{number.brand || "-"}</TableCell>
                                <TableCell>{numberTypeLabels[number.number_type] || number.number_type}</TableCell>
                                <TableCell>{number.is_primary ? <Badge>Primary</Badge> : "-"}</TableCell>
                                <TableCell className="max-w-56 truncate text-sm text-muted-foreground">{number.notes || "-"}</TableCell>
                                <TableCell>
                                  {canManage && !number.is_primary ? (
                                    <Button type="button" size="xs" variant="outline" onClick={() => void setPrimaryNumber(number)}>
                                      Jadikan primary
                                    </Button>
                                  ) : null}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                    {canManage ? (
                      <form className="grid gap-3 rounded-lg border border-border p-3 md:grid-cols-2" onSubmit={addNumber}>
                        <Field>
                          <FieldLabel htmlFor="part-number">Nomor part</FieldLabel>
                          <Input id="part-number" value={numberForm.part_number} maxLength={120} onChange={(event) => setNumberForm((current) => ({ ...current, part_number: event.target.value }))} required />
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="part-brand">Brand</FieldLabel>
                          <Input id="part-brand" value={numberForm.brand} maxLength={100} onChange={(event) => setNumberForm((current) => ({ ...current, brand: event.target.value }))} />
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="part-type">Tipe nomor</FieldLabel>
                          <NativeSelect id="part-type" value={numberForm.number_type} onChange={(event) => setNumberForm((current) => ({ ...current, number_type: event.target.value }))}>
                            <NativeSelectOption value="oem">OEM</NativeSelectOption>
                            <NativeSelectOption value="aftermarket">Aftermarket</NativeSelectOption>
                            <NativeSelectOption value="superseded">Superseded</NativeSelectOption>
                          </NativeSelect>
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="part-notes">Catatan</FieldLabel>
                          <Input id="part-notes" value={numberForm.notes} maxLength={500} onChange={(event) => setNumberForm((current) => ({ ...current, notes: event.target.value }))} />
                        </Field>
                        <label className="flex items-center gap-2 text-sm md:col-span-2">
                          <input type="checkbox" checked={numberForm.is_primary} onChange={(event) => setNumberForm((current) => ({ ...current, is_primary: event.target.checked }))} className="size-4 rounded border-border" />
                          Jadikan nomor utama
                        </label>
                        {numberError ? <div className="md:col-span-2"><FieldError>{numberError}</FieldError></div> : null}
                        <div className="md:col-span-2">
                          <Button type="submit" size="sm" disabled={numberBusy}>
                            {numberBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                            Tambah nomor part
                          </Button>
                        </div>
                      </form>
                    ) : null}
                  </div>
                ) : null}

                {detailTab === "alternatives" ? (
                  <div className="flex flex-col gap-3">
                    {detail.alternatives.length === 0 ? (
                      <p className="text-sm text-muted-foreground">Belum ada cross-reference alternatif.</p>
                    ) : (
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Sparepart</TableHead>
                              <TableHead>Relasi</TableHead>
                              <TableHead>Status</TableHead>
                              <TableHead>Verifikator</TableHead>
                              <TableHead>Aksi</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {detail.alternatives.map((alternative) => {
                              const other = alternative.direction === "outgoing"
                                ? `${alternative.alternative_sparepart_code} - ${alternative.alternative_sparepart_name}`
                                : `${alternative.sparepart_code} - ${alternative.sparepart_name}`;
                              return (
                                <TableRow key={alternative.id}>
                                  <TableCell>
                                    <p className="font-medium">{other}</p>
                                    <p className="text-xs text-muted-foreground">{alternative.direction === "outgoing" ? "Keluar" : "Masuk"}</p>
                                  </TableCell>
                                  <TableCell>{relationLabels[alternative.relation_type] || alternative.relation_type}</TableCell>
                                  <TableCell><CompatibilityBadge status={alternative.verification_status} /></TableCell>
                                  <TableCell className="text-xs text-muted-foreground">
                                    {alternative.verified_by_name || "-"}
                                    {alternative.verified_at ? <span className="block">{formatWita(alternative.verified_at)}</span> : null}
                                  </TableCell>
                                  <TableCell>
                                    {canManage && alternative.direction === "outgoing" ? (
                                      <Button type="button" size="xs" variant="outline" onClick={() => openVerifyAlternative(alternative)}>
                                        <ShieldCheck aria-hidden="true" /> Verifikasi
                                      </Button>
                                    ) : null}
                                  </TableCell>
                                </TableRow>
                              );
                            })}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                    {canManage ? (
                      <form className="grid gap-3 rounded-lg border border-border p-3 md:grid-cols-2" onSubmit={addAlternative}>
                        <Field>
                          <FieldLabel htmlFor="alternative-sparepart">Sparepart alternatif</FieldLabel>
                          <NativeSelect id="alternative-sparepart" value={alternativeForm.alternative_sparepart_id} onChange={(event) => setAlternativeForm((current) => ({ ...current, alternative_sparepart_id: event.target.value }))} required>
                            <NativeSelectOption value="">Pilih sparepart</NativeSelectOption>
                            {sparepartOptions.filter((option) => option.id !== detail.id).map((option) => (
                              <NativeSelectOption key={option.id} value={option.id}>{option.code} - {option.name}</NativeSelectOption>
                            ))}
                          </NativeSelect>
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="alternative-relation">Tipe relasi</FieldLabel>
                          <NativeSelect id="alternative-relation" value={alternativeForm.relation_type} onChange={(event) => setAlternativeForm((current) => ({ ...current, relation_type: event.target.value }))}>
                            <NativeSelectOption value="equivalent">Equivalent</NativeSelectOption>
                            <NativeSelectOption value="superseded">Superseded</NativeSelectOption>
                            <NativeSelectOption value="substitute">Substitute</NativeSelectOption>
                          </NativeSelect>
                        </Field>
                        <Field className="md:col-span-2">
                          <FieldLabel htmlFor="alternative-notes">Catatan</FieldLabel>
                          <Input id="alternative-notes" value={alternativeForm.notes} maxLength={500} onChange={(event) => setAlternativeForm((current) => ({ ...current, notes: event.target.value }))} />
                        </Field>
                        {alternativeError ? <div className="md:col-span-2"><FieldError>{alternativeError}</FieldError></div> : null}
                        <div className="md:col-span-2">
                          <Button type="submit" size="sm" disabled={alternativeBusy}>
                            {alternativeBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                            Tambah alternatif
                          </Button>
                        </div>
                      </form>
                    ) : null}
                  </div>
                ) : null}

                {detailTab === "compatibility" ? (
                  <div className="flex flex-col gap-3">
                    {detail.compatibility.length === 0 ? (
                      <p className="text-sm text-muted-foreground">Belum ada compatibility.</p>
                    ) : (
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Target</TableHead>
                              <TableHead>Status</TableHead>
                              <TableHead>Kondisi</TableHead>
                              <TableHead>Verifikator</TableHead>
                              <TableHead>Aksi</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {detail.compatibility.map((compatibility) => (
                              <TableRow key={compatibility.id}>
                                <TableCell>
                                  <p className="font-medium">
                                    {[compatibility.unit_model_name, compatibility.engine_model_name, compatibility.configuration].filter(Boolean).join(" · ") || "-"}
                                  </p>
                                  <p className="text-xs text-muted-foreground">{compatibility.source || "Sumber belum dicatat"}</p>
                                </TableCell>
                                <TableCell><CompatibilityBadge status={compatibility.status} /></TableCell>
                                <TableCell className="max-w-64 text-sm text-muted-foreground">{compatibility.condition_note || compatibility.notes || "-"}</TableCell>
                                <TableCell className="text-xs text-muted-foreground">
                                  {compatibility.verified_by_name || "-"}
                                  {compatibility.verified_at ? <span className="block">{formatWita(compatibility.verified_at)}</span> : null}
                                </TableCell>
                                <TableCell>
                                  {canManage ? (
                                    <Button type="button" size="xs" variant="outline" onClick={() => openVerifyCompatibility(compatibility)}>
                                      <ShieldCheck aria-hidden="true" /> Verifikasi
                                    </Button>
                                  ) : null}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                    {canManage ? (
                      <form className="grid gap-3 rounded-lg border border-border p-3 md:grid-cols-2" onSubmit={addCompatibility}>
                        <Field>
                          <FieldLabel htmlFor="compat-unit-model">Unit model</FieldLabel>
                          <NativeSelect id="compat-unit-model" value={compatibilityForm.unit_model_id} onChange={(event) => setCompatibilityForm((current) => ({ ...current, unit_model_id: event.target.value }))}>
                            <NativeSelectOption value="">Tidak dibatasi</NativeSelectOption>
                            {unitModels.map((model) => (
                              <NativeSelectOption key={model.id} value={model.id}>{model.manufacturer} {model.model}</NativeSelectOption>
                            ))}
                          </NativeSelect>
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="compat-engine-model">Engine model</FieldLabel>
                          <NativeSelect id="compat-engine-model" value={compatibilityForm.engine_model_id} onChange={(event) => setCompatibilityForm((current) => ({ ...current, engine_model_id: event.target.value }))}>
                            <NativeSelectOption value="">Tidak dibatasi</NativeSelectOption>
                            {engineModels.map((model) => (
                              <NativeSelectOption key={model.id} value={model.id}>{model.manufacturer} {model.model}</NativeSelectOption>
                            ))}
                          </NativeSelect>
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="compat-configuration">Konfigurasi</FieldLabel>
                          <Input id="compat-configuration" value={compatibilityForm.configuration} maxLength={150} placeholder="4x4, standard cab, ..." onChange={(event) => setCompatibilityForm((current) => ({ ...current, configuration: event.target.value }))} />
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="compat-source">Sumber</FieldLabel>
                          <Input id="compat-source" value={compatibilityForm.source} maxLength={150} placeholder="Katalog, pengalaman, ..." onChange={(event) => setCompatibilityForm((current) => ({ ...current, source: event.target.value }))} />
                        </Field>
                        <Field className="md:col-span-2">
                          <FieldLabel htmlFor="compat-notes">Catatan</FieldLabel>
                          <Input id="compat-notes" value={compatibilityForm.notes} maxLength={500} onChange={(event) => setCompatibilityForm((current) => ({ ...current, notes: event.target.value }))} />
                        </Field>
                        {compatibilityError ? <div className="md:col-span-2"><FieldError>{compatibilityError}</FieldError></div> : null}
                        <div className="md:col-span-2">
                          <Button type="submit" size="sm" disabled={compatibilityBusy}>
                            {compatibilityBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                            Tambah compatibility
                          </Button>
                        </div>
                      </form>
                    ) : null}
                  </div>
                ) : null}
              </>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

      <Sheet open={verify !== null} onOpenChange={(open) => { if (!open) setVerify(null); }}>
        <SheetContent className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>Verifikasi compatibility</SheetTitle>
            <SheetDescription>{verify?.label || ""}</SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitVerify}>
            <Field>
              <FieldLabel htmlFor="verify-status">Status</FieldLabel>
              <NativeSelect id="verify-status" value={verify?.status || "approved_compatible"} onChange={(event) => setVerify((current) => current ? { ...current, status: event.target.value as CompatibilityStatus } : current)}>
                <NativeSelectOption value="approved_compatible">Approved Compatible</NativeSelectOption>
                <NativeSelectOption value="conditional_compatible">Conditional Compatible</NativeSelectOption>
                <NativeSelectOption value="not_compatible">Not Compatible</NativeSelectOption>
              </NativeSelect>
            </Field>
            {verify?.kind === "compatibility" ? (
              <Field>
                <FieldLabel htmlFor="verify-condition">Catatan kondisi</FieldLabel>
                <Textarea
                  id="verify-condition"
                  value={verify?.condition_note || ""}
                  onChange={(event) => setVerify((current) => current ? { ...current, condition_note: event.target.value } : current)}
                  required={verify?.status === "conditional_compatible"}
                  rows={3}
                  placeholder="Wajib untuk Conditional Compatible"
                />
              </Field>
            ) : null}
            <Field>
              <FieldLabel htmlFor="verify-notes">Catatan verifikasi</FieldLabel>
              <Textarea
                id="verify-notes"
                value={verify?.notes || ""}
                onChange={(event) => setVerify((current) => current ? { ...current, notes: event.target.value } : current)}
                required={verify?.kind === "alternative" && verify?.status === "conditional_compatible"}
                rows={3}
              />
            </Field>
            {verifyError ? <FieldError>{verifyError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" disabled={verifyBusy}>
                {verifyBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <ShieldCheck aria-hidden="true" />}
                Simpan verifikasi
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function CompatibilityBadge({ status }: { status: CompatibilityStatus }) {
  const styles: Record<CompatibilityStatus, string> = {
    pending_verification: "bg-amber-500/10 text-amber-700 dark:text-amber-300",
    approved_compatible: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
    conditional_compatible: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
    not_compatible: "bg-destructive/10 text-destructive",
  };
  return (
    <Badge variant="secondary" className={cn("h-5 whitespace-nowrap px-2", styles[status])}>
      {compatibilityLabels[status] || status}
    </Badge>
  );
}
