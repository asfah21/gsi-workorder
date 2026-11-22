"use client";

import {
  ArrowDownToLine,
  ArrowUpFromLine,
  CheckCircle2,
  CircleAlert,
  Loader2,
  PackageSearch,
  Plus,
  RefreshCw,
  RotateCcw,
  ShieldCheck,
  Trash2,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
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
import { Textarea } from "@/components/ui/textarea";
import { ApiError, proxyGet, proxyPatch, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  InventoryReservation,
  ListResponse,
  MaterialRequirement,
  MaterialRequirementStatus,
  MaterialUsage,
  Sparepart,
  Warehouse,
} from "@/lib/types";
import { formatWita } from "@/lib/time";
import { cn } from "@/lib/utils";

type ItemDraft = {
  key: string;
  sparepart_id: string;
  qty_required: string;
  is_optional: boolean;
  preferred_warehouse_id: string;
  notes: string;
};

const statusLabels: Record<MaterialRequirementStatus, string> = {
  draft: "Draft",
  planned: "Planned",
  checking_stock: "Checking stock",
  ready: "Ready",
  partially_ready: "Partially ready",
  shortage: "Shortage",
  reserved: "Reserved",
  issued: "Issued",
  completed: "Completed",
  cancelled: "Cancelled",
};

function newItem(): ItemDraft {
  return {
    key: typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`,
    sparepart_id: "",
    qty_required: "1",
    is_optional: false,
    preferred_warehouse_id: "",
    notes: "",
  };
}

function makeKey() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `mat-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function WorkOrderMaterialPanel({ woID, woStatus }: { woID: number; woStatus: string }) {
  const { user } = useSession();
  const canPlan = user.role === "admin" || user.role === "planner";
  const canReserve = canPlan || user.role === "storekeeper";
  const canIssue = user.role === "admin" || user.role === "storekeeper";
  const terminal = woStatus === "done" || woStatus === "cancelled";

  const [requirement, setRequirement] = useState<MaterialRequirement | null>(null);
  const [reservations, setReservations] = useState<InventoryReservation[]>([]);
  const [spareparts, setSpareparts] = useState<Sparepart[]>([]);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const [createOpen, setCreateOpen] = useState(false);
  const [editingItems, setEditingItems] = useState(false);
  const [createNotes, setCreateNotes] = useState("");
  const [draftItems, setDraftItems] = useState<ItemDraft[]>([newItem()]);

  const [issueOpen, setIssueOpen] = useState(false);
  const [issueQtys, setIssueQtys] = useState<Record<number, string>>({});
  const [issueNotes, setIssueNotes] = useState("");
  const [issueKey, setIssueKey] = useState("");

  const [returnOpen, setReturnOpen] = useState(false);
  const [returnUsageID, setReturnUsageID] = useState("");
  const [returnQty, setReturnQty] = useState("1");
  const [returnNotes, setReturnNotes] = useState("");
  const [returnKey, setReturnKey] = useState("");
  const [cancelReason, setCancelReason] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await proxyGet<ListResponse<MaterialRequirement>>(
        `/material-requirements?wo_id=${woID}&limit=10`,
      );
      const current = result.data[0] ?? null;
      setRequirement(current);
      if (current) {
        const reservationResult = await proxyGet<ListResponse<InventoryReservation>>(
          `/inventory/reservations?material_requirement_id=${current.id}&limit=100`,
        );
        setReservations(reservationResult.data);
      } else {
        setReservations([]);
      }
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Material requirement tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, [woID]);

  const loadReferences = useCallback(async () => {
    try {
      const [sparepartResult, warehouseResult] = await Promise.all([
        proxyGet<ListResponse<Sparepart>>("/spareparts?limit=100&active_only=true&sort=code&order=asc"),
        proxyGet<ListResponse<Warehouse>>("/warehouses?limit=100&active_only=true"),
      ]);
      setSpareparts(sparepartResult.data);
      setWarehouses(warehouseResult.data);
    } catch {
      // Reference data is best-effort; the panel still works with existing items.
    }
  }, []);

  useEffect(() => {
    void load();
    void loadReferences();
  }, [load, loadReferences]);

  const issueUsages = useMemo(
    () => requirement?.usages.filter((usage) => usage.movement_type === "issue") ?? [],
    [requirement],
  );
  const activeReservations = reservations.filter((item) => item.status === "active");
  const shortageDays = requirement?.shortage_since
    ? Math.max(0, Math.floor((Date.now() - new Date(requirement.shortage_since).getTime()) / 86400000))
    : 0;

  const run = async (action: () => Promise<MaterialRequirement>, message: string) => {
    if (busy) return;
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      const result = await action();
      setRequirement(result);
      setSuccess(message);
      await load();
    } catch (actionError) {
      setError(actionError instanceof ApiError ? actionError.message : "Aksi material gagal.");
    } finally {
      setBusy(false);
    }
  };

  const openEditItems = () => {
    if (!requirement) return;
    setEditingItems(true);
    setCreateNotes(requirement.notes);
    setDraftItems(
      requirement.items.map((item) => ({
        key: makeKey(),
        sparepart_id: String(item.sparepart_id),
        qty_required: String(item.qty_required),
        is_optional: item.is_optional,
        preferred_warehouse_id: item.preferred_warehouse_id ? String(item.preferred_warehouse_id) : "",
        notes: item.notes,
      })),
    );
    setCreateOpen(true);
  };

  const closeItemForm = () => {
    setCreateOpen(false);
    setEditingItems(false);
    setCreateNotes("");
    setDraftItems([newItem()]);
  };

  const submitCreate = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (busy) return;
    const items = draftItems
      .filter((item) => item.sparepart_id && Number(item.qty_required) > 0)
      .map((item) => ({
        sparepart_id: Number(item.sparepart_id),
        qty_required: Number(item.qty_required),
        is_optional: item.is_optional,
        preferred_warehouse_id: item.preferred_warehouse_id ? Number(item.preferred_warehouse_id) : null,
        notes: item.notes.trim(),
      }));
    if (items.length === 0) {
      setError("Minimal satu item material wajib diisi.");
      return;
    }
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      const result = editingItems && requirement
        ? await proxyPatch<DataResponse<MaterialRequirement>>(
            `/material-requirements/${requirement.id}`,
            { notes: createNotes.trim(), items },
          )
        : await proxyPost<DataResponse<MaterialRequirement>>("/material-requirements", {
            wo_id: woID,
            source: "manual",
            notes: createNotes.trim(),
            items,
          });
      setRequirement(result.data);
      setCreateOpen(false);
      setEditingItems(false);
      setCreateNotes("");
      setDraftItems([newItem()]);
      setSuccess(editingItems ? "Item material diperbarui." : `Material requirement ${result.data.code} dibuat.`);
      await load();
    } catch (actionError) {
      setError(actionError instanceof ApiError ? actionError.message : "Material requirement tidak dapat disimpan.");
    } finally {
      setBusy(false);
    }
  };

  const transitionStatus = async (status: string) => {
    if (!requirement) return;
    await run(async () => {
      const result = await proxyPatch<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}`,
        { status },
      );
      return result.data;
    }, "Status material diperbarui.");
  };

  const checkStock = async () => {
    if (!requirement) return;
    await run(async () => {
      const result = await proxyPost<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}/check-stock`,
        {},
      );
      return result.data;
    }, "Readiness material dihitung ulang.");
  };

  const reserve = async () => {
    if (!requirement) return;
    await run(async () => {
      const result = await proxyPost<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}/reserve`,
        { idempotency_key: makeKey(), notes: "Reserve dari UI" },
      );
      return result.data;
    }, "Reservasi material diposting.");
  };

  const submitIssue = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!requirement || busy) return;
    const items = Object.entries(issueQtys)
      .map(([id, value]) => ({ id: Number(id), qty: Number(value) }))
      .filter((item) => item.id > 0 && Number.isFinite(item.qty) && item.qty > 0)
      .map((item) => ({ material_requirement_item_id: item.id, qty: item.qty, notes: issueNotes.trim() }));
    if (items.length === 0) {
      setError("Isi minimal satu qty issue.");
      return;
    }
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      const result = await proxyPost<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}/issue`,
        { idempotency_key: issueKey, notes: issueNotes.trim(), items },
      );
      setRequirement(result.data);
      setIssueOpen(false);
      setIssueQtys({});
      setIssueNotes("");
      setSuccess("Issue material berhasil diposting.");
      await load();
    } catch (actionError) {
      setError(actionError instanceof ApiError ? actionError.message : "Issue material gagal.");
    } finally {
      setBusy(false);
    }
  };

  const openIssue = () => {
    setIssueKey(makeKey());
    setIssueQtys({});
    setIssueNotes("");
    setIssueOpen(true);
    setReturnOpen(false);
  };

  const openReturn = (usage?: MaterialUsage) => {
    setReturnKey(makeKey());
    setReturnUsageID(usage ? String(usage.id) : "");
    setReturnQty("1");
    setReturnNotes("");
    setReturnOpen(true);
    setIssueOpen(false);
  };

  const submitReturn = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!requirement || busy) return;
    if (!returnUsageID || Number(returnQty) <= 0) {
      setError("Pilih usage issue dan qty return.");
      return;
    }
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      const result = await proxyPost<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}/return`,
        {
          idempotency_key: returnKey,
          items: [{ usage_id: Number(returnUsageID), qty: Number(returnQty), notes: returnNotes.trim() }],
        },
      );
      setRequirement(result.data);
      setReturnOpen(false);
      setSuccess("Return material berhasil diposting.");
      await load();
    } catch (actionError) {
      setError(actionError instanceof ApiError ? actionError.message : "Return material gagal.");
    } finally {
      setBusy(false);
    }
  };

  const cancelRequirement = async () => {
    if (!requirement || busy) return;
    if (cancelReason.trim().length < 5) {
      setError("Alasan pembatalan minimal 5 karakter.");
      return;
    }
    await run(async () => {
      const result = await proxyPost<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}/cancel`,
        { reason: cancelReason.trim() },
      );
      setCancelReason("");
      return result.data;
    }, "Material requirement dibatalkan.");
  };

  const releaseReservation = async (reservation: InventoryReservation) => {
    if (!requirement || busy) return;
    await run(async () => {
      await proxyPost<DataResponse<InventoryReservation>>(
        `/inventory/reservations/${reservation.id}/release`,
        { reason: "Release dari UI" },
      );
      const refreshed = await proxyGet<DataResponse<MaterialRequirement>>(
        `/material-requirements/${requirement.id}`,
      );
      return refreshed.data;
    }, "Reservasi dilepas.");
  };

  const canCreate = canPlan && !terminal && !requirement;
  const canEditItems = canPlan && requirement && !terminal && (requirement.status === "draft" || requirement.status === "planned");
  const canEditStatus = canPlan && requirement && !terminal && (requirement.status === "draft" || requirement.status === "planned");
  const canCheck = canReserve && requirement && !terminal && !["draft", "cancelled", "completed"].includes(requirement.status);
  const canReserveNow = canReserve && requirement && !terminal && ["ready", "partially_ready", "shortage", "reserved", "issued"].includes(requirement.status);
  const canIssueNow = canIssue && requirement && !terminal && ["ready", "partially_ready", "shortage", "reserved", "issued"].includes(requirement.status);
  const canReturnNow = canIssue && requirement && issueUsages.length > 0;
  const canCancelNow = canPlan && requirement && !["completed", "cancelled"].includes(requirement.status);

  return (
    <div className="rounded-lg border border-border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="flex items-center gap-2 text-sm font-semibold">
            <PackageSearch className="size-4" aria-hidden="true" /> Material
          </h3>
          <p className="mt-1 text-xs text-muted-foreground">
            Planning dan reservasi tidak mengurangi stok fisik. Issue mengurangi on_hand.
          </p>
        </div>
        <Button type="button" size="xs" variant="outline" onClick={() => void load()} disabled={loading}>
          {loading ? <Loader2 className="animate-spin" aria-hidden="true" /> : <RefreshCw aria-hidden="true" />}
          Muat
        </Button>
      </div>

      {error ? <p className="mt-2 text-sm text-destructive">{error}</p> : null}
      {success ? <p className="mt-2 text-sm text-emerald-700 dark:text-emerald-300">{success}</p> : null}

      {loading ? (
        <div className="mt-3 flex flex-col gap-2">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
      ) : !requirement ? (
        <div className="mt-3 flex flex-col gap-2">
          <p className="text-sm text-muted-foreground">Belum ada material requirement untuk work order ini.</p>
          {canCreate ? (
            <div>
              <Button type="button" size="sm" onClick={() => setCreateOpen((open) => !open)}>
                <Plus aria-hidden="true" /> Buat material requirement
              </Button>
            </div>
          ) : null}
        </div>
      ) : (
        <div className="mt-3 flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="font-mono text-xs">{requirement.code}</span>
            <MaterialStatusBadge status={requirement.status} />
            {requirement.derived_status !== requirement.status ? (
              <Badge variant="outline">Readiness: {statusLabels[requirement.derived_status]}</Badge>
            ) : null}
            {requirement.source === "pm" ? <Badge variant="outline">Snapshot PM</Badge> : null}
            {requirement.shortage_since ? (
              <Badge variant="destructive">Shortage {shortageDays} hari</Badge>
            ) : null}
          </div>
          {requirement.notes ? <p className="text-sm text-muted-foreground">{requirement.notes}</p> : null}

          <div className="overflow-x-auto rounded-lg border border-border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Sparepart</TableHead>
                  <TableHead>Required</TableHead>
                  <TableHead>Reserved</TableHead>
                  <TableHead>Issued</TableHead>
                  <TableHead>Returned</TableHead>
                  <TableHead>Available</TableHead>
                  <TableHead>Coverage</TableHead>
                  <TableHead>Shortage</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {requirement.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <p className="font-medium">{item.sparepart_code}</p>
                      <p className="text-xs text-muted-foreground">
                        {item.sparepart_name}{item.is_optional ? " · opsional" : ""}
                      </p>
                    </TableCell>
                    <TableCell className="whitespace-nowrap">{item.qty_required} {item.uom}</TableCell>
                    <TableCell>{item.qty_reserved}</TableCell>
                    <TableCell>{item.qty_issued}</TableCell>
                    <TableCell>{item.qty_returned}</TableCell>
                    <TableCell>{item.available}</TableCell>
                    <TableCell>{item.coverage}</TableCell>
                    <TableCell className={cn(item.shortage > 0 && !item.is_optional && "font-medium text-destructive")}>
                      {item.shortage > 0 && !item.is_optional ? item.shortage : "-"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {activeReservations.length > 0 ? (
            <div className="rounded-lg border border-border p-2">
              <p className="text-xs font-medium">Reservasi aktif ({activeReservations.length})</p>
              <div className="mt-1 flex flex-col gap-1 text-xs text-muted-foreground">
                {activeReservations.map((reservation) => (
                  <div key={reservation.id} className="flex flex-wrap items-center justify-between gap-2">
                    <span>
                      {reservation.sparepart_code} · {reservation.warehouse_code}/{reservation.location_code} · {reservation.qty} {reservation.uom}
                    </span>
                    {canReserve ? (
                      <Button type="button" size="xs" variant="outline" onClick={() => void releaseReservation(reservation)} disabled={busy}>
                        Release
                      </Button>
                    ) : null}
                  </div>
                ))}
              </div>
            </div>
          ) : null}

          <div className="flex flex-wrap gap-2">
            {canEditItems ? (
              <Button type="button" size="sm" variant="outline" onClick={openEditItems} disabled={busy}>
                Edit item
              </Button>
            ) : null}
            {canEditStatus && requirement.status === "draft" ? (
              <Button type="button" size="sm" variant="outline" onClick={() => void transitionStatus("planned")} disabled={busy}>
                Rencanakan
              </Button>
            ) : null}
            {canCheck ? (
              <Button type="button" size="sm" variant="outline" onClick={() => void checkStock()} disabled={busy}>
                <RefreshCw aria-hidden="true" /> Check stock
              </Button>
            ) : null}
            {canReserveNow ? (
              <Button type="button" size="sm" onClick={() => void reserve()} disabled={busy}>
                <ShieldCheck aria-hidden="true" /> Reserve
              </Button>
            ) : null}
            {canIssueNow ? (
              <Button type="button" size="sm" onClick={openIssue} disabled={busy}>
                <ArrowDownToLine aria-hidden="true" /> Issue
              </Button>
            ) : null}
            {canReturnNow ? (
              <Button type="button" size="sm" variant="outline" onClick={() => openReturn()} disabled={busy}>
                <ArrowUpFromLine aria-hidden="true" /> Return
              </Button>
            ) : null}
          </div>

          {requirement.status === "shortage" || requirement.status === "partially_ready" ? (
            <Alert>
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Material belum siap</AlertTitle>
              <AlertDescription>Work order boleh masuk waiting part, tetapi belum boleh lanjut sampai material siap.</AlertDescription>
            </Alert>
          ) : null}

          {canCancelNow ? (
            <div className="flex flex-col gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-2">
              <Input
                value={cancelReason}
                onChange={(event) => setCancelReason(event.target.value)}
                placeholder="Alasan pembatalan material..."
                aria-label="Alasan pembatalan material"
              />
              <Button type="button" size="xs" variant="destructive" onClick={() => void cancelRequirement()} disabled={busy}>
                <XCircle aria-hidden="true" /> Batalkan material
              </Button>
            </div>
          ) : null}

          {requirement.usages.length > 0 ? (
            <div className="overflow-x-auto rounded-lg border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Waktu</TableHead>
                    <TableHead>Tipe</TableHead>
                    <TableHead>Sparepart</TableHead>
                    <TableHead>Lokasi</TableHead>
                    <TableHead>Qty</TableHead>
                    <TableHead>Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {requirement.usages.map((usage) => (
                    <TableRow key={usage.id}>
                      <TableCell className="whitespace-nowrap text-xs">{formatWita(usage.used_at)}</TableCell>
                      <TableCell>
                        <Badge variant={usage.movement_type === "issue" ? "secondary" : "outline"}>
                          {usage.movement_type === "issue" ? "Issue" : "Return"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <p className="font-medium">{usage.sparepart_code}</p>
                        <p className="text-xs text-muted-foreground">{usage.sparepart_name}</p>
                      </TableCell>
                      <TableCell className="text-sm">{usage.warehouse_code}/{usage.location_code}</TableCell>
                      <TableCell>{usage.qty}</TableCell>
                      <TableCell>
                        {usage.movement_type === "issue" && canIssue ? (
                          <Button type="button" size="xs" variant="outline" onClick={() => openReturn(usage)} disabled={busy || terminal}>
                            <RotateCcw aria-hidden="true" /> Return
                          </Button>
                        ) : null}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : null}
        </div>
      )}

      {createOpen && (canCreate || canEditItems) ? (
        <form className="mt-3 flex flex-col gap-3 rounded-lg border border-border p-3" onSubmit={submitCreate}>
          <p className="text-sm font-medium">{editingItems ? "Edit item material" : "Buat material requirement"}</p>
          <Field>
            <FieldLabel htmlFor="material-create-notes">Catatan</FieldLabel>
            <Input id="material-create-notes" value={createNotes} maxLength={1000} onChange={(event) => setCreateNotes(event.target.value)} />
          </Field>
          {draftItems.map((item, index) => (
            <div key={item.key} className="rounded-lg border border-border p-2">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-medium">Item {index + 1}</p>
                <Button
                  type="button"
                  size="xs"
                  variant="outline"
                  disabled={draftItems.length === 1}
                  onClick={() => setDraftItems((items) => items.filter((row) => row.key !== item.key))}
                >
                  <Trash2 aria-hidden="true" />
                </Button>
              </div>
              <div className="mt-2 grid gap-2 sm:grid-cols-2">
                <Field className="sm:col-span-2">
                  <FieldLabel htmlFor={`material-part-${item.key}`}>Sparepart</FieldLabel>
                  <NativeSelect
                    id={`material-part-${item.key}`}
                    value={item.sparepart_id}
                    onChange={(event) => setDraftItems((items) => items.map((row) => row.key === item.key ? { ...row, sparepart_id: event.target.value } : row))}
                    required
                  >
                    <NativeSelectOption value="">Pilih sparepart</NativeSelectOption>
                    {spareparts.map((sparepart) => (
                      <NativeSelectOption key={sparepart.id} value={sparepart.id}>
                        {sparepart.code} - {sparepart.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor={`material-qty-${item.key}`}>Qty required</FieldLabel>
                  <Input
                    id={`material-qty-${item.key}`}
                    type="number"
                    min={0.001}
                    step="0.001"
                    value={item.qty_required}
                    onChange={(event) => setDraftItems((items) => items.map((row) => row.key === item.key ? { ...row, qty_required: event.target.value } : row))}
                    required
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor={`material-warehouse-${item.key}`}>Preferred warehouse</FieldLabel>
                  <NativeSelect
                    id={`material-warehouse-${item.key}`}
                    value={item.preferred_warehouse_id}
                    onChange={(event) => setDraftItems((items) => items.map((row) => row.key === item.key ? { ...row, preferred_warehouse_id: event.target.value } : row))}
                  >
                    <NativeSelectOption value="">Semua gudang</NativeSelectOption>
                    {warehouses.map((warehouse) => (
                      <NativeSelectOption key={warehouse.id} value={warehouse.id}>{warehouse.code} - {warehouse.name}</NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <label className="flex items-center gap-2 text-sm sm:col-span-2">
                  <input
                    type="checkbox"
                    className="size-4 rounded border-border"
                    checked={item.is_optional}
                    onChange={(event) => setDraftItems((items) => items.map((row) => row.key === item.key ? { ...row, is_optional: event.target.checked } : row))}
                  />
                  Item opsional (tidak memblokir readiness)
                </label>
              </div>
            </div>
          ))}
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" variant="outline" onClick={() => setDraftItems((items) => [...items, newItem()])}>
              <Plus aria-hidden="true" /> Tambah item
            </Button>
            <Button type="submit" size="sm" disabled={busy}>
              {busy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
              Simpan
            </Button>
            <Button type="button" size="sm" variant="outline" onClick={closeItemForm}>Batal</Button>
          </div>
        </form>
      ) : null}

      {issueOpen && requirement ? (
        <form className="mt-3 flex flex-col gap-3 rounded-lg border border-border p-3" onSubmit={submitIssue}>
          <p className="text-sm font-medium">Issue material</p>
          {requirement.items.map((item) => (
            <div key={item.id} className="grid items-end gap-2 sm:grid-cols-[1fr_130px]">
              <div>
                <p className="text-sm font-medium">{item.sparepart_code} - {item.sparepart_name}</p>
                <p className="text-xs text-muted-foreground">
                  Required {item.qty_required}, issued {item.qty_issued}, reserved {item.qty_reserved}, {item.uom}
                </p>
              </div>
              <Field>
                <FieldLabel htmlFor={`issue-${item.id}`}>Qty issue</FieldLabel>
                <Input
                  id={`issue-${item.id}`}
                  type="number"
                  min={0}
                  step="0.001"
                  value={issueQtys[item.id] ?? ""}
                  onChange={(event) => setIssueQtys((current) => ({ ...current, [item.id]: event.target.value }))}
                />
              </Field>
            </div>
          ))}
          <Field>
            <FieldLabel htmlFor="issue-notes">Catatan</FieldLabel>
            <Textarea id="issue-notes" value={issueNotes} maxLength={1000} onChange={(event) => setIssueNotes(event.target.value)} rows={2} />
          </Field>
          <div className="flex gap-2">
            <Button type="submit" size="sm" disabled={busy}>
              {busy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <ArrowDownToLine aria-hidden="true" />}
              Posting issue
            </Button>
            <Button type="button" size="sm" variant="outline" onClick={() => setIssueOpen(false)}>Batal</Button>
          </div>
        </form>
      ) : null}

      {returnOpen && requirement ? (
        <form className="mt-3 flex flex-col gap-3 rounded-lg border border-border p-3" onSubmit={submitReturn}>
          <p className="text-sm font-medium">Return material</p>
          <Field>
            <FieldLabel htmlFor="return-usage">Usage issue asal</FieldLabel>
            <NativeSelect id="return-usage" value={returnUsageID} onChange={(event) => setReturnUsageID(event.target.value)} required>
              <NativeSelectOption value="">Pilih usage issue</NativeSelectOption>
              {issueUsages.map((usage) => (
                <NativeSelectOption key={usage.id} value={usage.id}>
                  {formatWita(usage.used_at)} · {usage.sparepart_code} · {usage.qty}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field>
            <FieldLabel htmlFor="return-qty">Qty return</FieldLabel>
            <Input id="return-qty" type="number" min={0.001} step="0.001" value={returnQty} onChange={(event) => setReturnQty(event.target.value)} required />
          </Field>
          <Field>
            <FieldLabel htmlFor="return-notes">Catatan</FieldLabel>
            <Input id="return-notes" value={returnNotes} maxLength={500} onChange={(event) => setReturnNotes(event.target.value)} />
          </Field>
          <div className="flex gap-2">
            <Button type="submit" size="sm" disabled={busy}>
              {busy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <ArrowUpFromLine aria-hidden="true" />}
              Posting return
            </Button>
            <Button type="button" size="sm" variant="outline" onClick={() => setReturnOpen(false)}>Batal</Button>
          </div>
        </form>
      ) : null}

      {canCreate ? null : !requirement && !canPlan ? (
        <p className="mt-3 text-xs text-muted-foreground">Hanya admin/planner yang dapat membuat material requirement.</p>
      ) : null}
    </div>
  );
}

function MaterialStatusBadge({ status }: { status: MaterialRequirementStatus }) {
  const styles: Record<MaterialRequirementStatus, string> = {
    draft: "bg-muted text-muted-foreground",
    planned: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
    checking_stock: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
    ready: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
    partially_ready: "bg-amber-500/10 text-amber-700 dark:text-amber-300",
    shortage: "bg-destructive/10 text-destructive",
    reserved: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
    issued: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
    completed: "bg-muted text-muted-foreground",
    cancelled: "bg-destructive/10 text-destructive",
  };
  return (
    <Badge variant="secondary" className={cn("h-5 px-2", styles[status])}>
      {statusLabels[status] || status}
    </Badge>
  );
}
