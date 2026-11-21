"use client";

import {
  CheckCircle2,
  CircleAlert,
  Filter,
  Loader2,
  MessageSquare,
  Paperclip,
  Pause,
  Play,
  Plus,
  RefreshCw,
  Search,
  Wrench,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";

import { useSession } from "@/components/app-shell";
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
import { useListParams } from "@/hooks/use-list-params";
import {
  ApiError,
  proxyGet,
  proxyPatch,
  proxyPost,
  proxyUpload,
} from "@/lib/api-client";
import type {
  DataResponse,
  ListResponse,
  Unit,
  WorkOrderAttachment,
  WorkOrderDetail,
  WorkOrderMaintenanceType,
  WorkOrderPriority,
  WorkOrderRecord,
  WorkOrderUpdate,
} from "@/lib/types";
import { formatDateTimeWita, formatWita } from "@/lib/time";
import { cn } from "@/lib/utils";

const sortableColumns = ["created_at", "updated_at", "code", "title", "priority", "status"] as const;

const statusLabels: Record<string, string> = {
  open: "Open",
  in_progress: "In progress",
  waiting_part: "Waiting part",
  done: "Done",
  cancelled: "Cancelled",
};

const maintenanceLabels: Record<WorkOrderMaintenanceType, string> = {
  preventive: "Preventive",
  corrective: "Corrective",
  breakdown: "Breakdown",
  inspection: "Inspection",
};

const priorityLabels: Record<WorkOrderPriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

function WorkOrderStatusBadge({ status }: { status: string }) {
  return (
    <Badge
      variant="secondary"
      className={cn(
        "h-5 border-transparent px-2",
        status === "open" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
        status === "in_progress" && "bg-amber-500/10 text-amber-700 dark:text-amber-300",
        status === "waiting_part" && "bg-orange-500/10 text-orange-700 dark:text-orange-300",
        status === "done" && "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        status === "cancelled" && "bg-muted text-muted-foreground",
      )}
    >
      {statusLabels[status] || status || "-"}
    </Badge>
  );
}

function WorkOrderPriorityBadge({ priority }: { priority: WorkOrderPriority }) {
  return (
    <Badge
      variant="outline"
      className={cn(
        "h-5 px-2",
        priority === "critical" && "border-destructive/30 text-destructive",
        priority === "high" && "border-amber-500/30 text-amber-700 dark:text-amber-300",
        priority === "medium" && "border-sky-500/30 text-sky-700 dark:text-sky-300",
      )}
    >
      {priorityLabels[priority] || priority}
    </Badge>
  );
}

export function WorkOrdersView() {
  const { user } = useSession();
  const searchParams = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const status = searchParams.get("status") ?? "";
  const source = searchParams.get("source") ?? "";
  const list = useListParams({
    defaultSort: "created_at",
    allowedSorts: sortableColumns,
    defaultOrder: "desc",
  });

  const [items, setItems] = useState<WorkOrderRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [queryInput, setQueryInput] = useState(q);

  const [detailID, setDetailID] = useState<number | null>(null);
  const [detail, setDetail] = useState<WorkOrderDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [actionNote, setActionNote] = useState("");
  const [acting, setActing] = useState(false);

  const [createOpen, setCreateOpen] = useState(false);
  const [units, setUnits] = useState<Unit[]>([]);
  const [createError, setCreateError] = useState("");
  const [createBusy, setCreateBusy] = useState(false);
  const [createForm, setCreateForm] = useState({
    unit_id: "",
    title: "",
    description: "",
    priority: "medium",
    maintenance_type: "corrective",
    due_date: "",
  });

  const [comment, setComment] = useState("");
  const [attachment, setAttachment] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);

  const canCreate = user.role === "admin" || user.role === "planner";
  const canCancel = user.role === "admin" || user.role === "planner";

  const loadList = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({
        page: String(list.page),
        limit: String(list.pageSize),
        sort: list.sort,
        order: list.order,
      });
      if (q.trim()) params.set("q", q.trim());
      if (status) params.set("status", status);
      if (source) params.set("source", source);
      const result = await proxyGet<ListResponse<WorkOrderRecord>>(
        `/workorders?${params.toString()}`,
      );
      setItems(result.data);
      setTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Work order tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, [list.order, list.page, list.pageSize, list.sort, q, source, status]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  useEffect(() => {
    setQueryInput(q);
  }, [q]);

  const loadDetail = useCallback(async (id: number) => {
    setDetailLoading(true);
    setDetailError("");
    try {
      const result = await proxyGet<DataResponse<WorkOrderDetail>>(`/workorders/${id}`);
      setDetail(result.data);
    } catch (loadError) {
      setDetail(null);
      setDetailError(loadError instanceof ApiError ? loadError.message : "Detail work order tidak dapat dimuat.");
    } finally {
      setDetailLoading(false);
    }
  }, []);

  const openDetail = (id: number) => {
    setDetailID(id);
    setDetail(null);
    setActionNote("");
    setComment("");
    setAttachment(null);
    void loadDetail(id);
  };

  const refreshDetail = async (id: number) => {
    await loadDetail(id);
    await loadList();
  };

  const transition = async (nextStatus: string) => {
    if (!detail || acting) return;
    setActing(true);
    setDetailError("");
    try {
      await proxyPatch(`/workorders/${detail.id}`, {
        status: nextStatus,
        note: actionNote,
        reason: actionNote,
        ...(nextStatus === "in_progress" && !detail.assignee_id
          ? { assignee_id: user.id }
          : {}),
      });
      setSuccess(`Work order ${detail.code} dipindahkan ke ${statusLabels[nextStatus] || nextStatus}.`);
      setActionNote("");
      await refreshDetail(detail.id);
    } catch (actionError) {
      setDetailError(actionError instanceof ApiError ? actionError.message : "Status work order tidak dapat diubah.");
    } finally {
      setActing(false);
    }
  };

  const addComment = async () => {
    if (!detail || !comment.trim() || acting) return;
    setActing(true);
    setDetailError("");
    try {
      await proxyPost(`/workorders/${detail.id}/updates`, { note: comment });
      setComment("");
      setSuccess("Catatan work order ditambahkan.");
      await loadDetail(detail.id);
    } catch (commentError) {
      setDetailError(commentError instanceof ApiError ? commentError.message : "Catatan tidak dapat ditambahkan.");
    } finally {
      setActing(false);
    }
  };

  const uploadAttachment = async () => {
    if (!detail || !attachment || uploading) return;
    setUploading(true);
    setDetailError("");
    try {
      const form = new FormData();
      form.append("file", attachment);
      await proxyUpload<DataResponse<WorkOrderAttachment>>(
        `/workorders/${detail.id}/attachments`,
        form,
      );
      setAttachment(null);
      setSuccess("Lampiran berhasil diunggah.");
      await loadDetail(detail.id);
    } catch (uploadError) {
      setDetailError(uploadError instanceof ApiError ? uploadError.message : "Lampiran tidak dapat diunggah.");
    } finally {
      setUploading(false);
    }
  };

  const openCreate = async () => {
    setCreateOpen(true);
    setCreateError("");
    if (units.length === 0) {
      try {
        const result = await proxyGet<ListResponse<Unit>>(
          "/units?limit=100&sort=code&order=asc",
        );
        setUnits(result.data);
      } catch {
        setCreateError("Daftar unit tidak dapat dimuat.");
      }
    }
  };

  const submitCreate = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (createBusy) return;
    setCreateBusy(true);
    setCreateError("");
    try {
      const result = await proxyPost<DataResponse<WorkOrderRecord>>("/workorders", {
        unit_id: Number(createForm.unit_id),
        title: createForm.title,
        description: createForm.description,
        priority: createForm.priority,
        maintenance_type: createForm.maintenance_type,
        due_date: createForm.due_date,
      });
      setCreateOpen(false);
      setCreateForm({
        unit_id: "",
        title: "",
        description: "",
        priority: "medium",
        maintenance_type: "corrective",
        due_date: "",
      });
      setSuccess(`Work order ${result.data.code} dibuat.`);
      await loadList();
      openDetail(result.data.id);
    } catch (createFailure) {
      setCreateError(createFailure instanceof ApiError ? createFailure.message : "Work order tidak dapat dibuat.");
    } finally {
      setCreateBusy(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / list.pageSize));
  const canAct = detail && ["open", "in_progress", "waiting_part"].includes(detail.status);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Work Order</h1>
          <p className="text-sm text-muted-foreground">
            Work order manual, assignment, status, temuan P2H, dan lampiran.
          </p>
        </div>
        {canCreate ? (
          <Button type="button" onClick={() => void openCreate()}>
            <Plus aria-hidden="true" />
            Buat WO manual
          </Button>
        ) : null}
      </div>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal memuat work order</AlertTitle>
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
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>Daftar work order</CardTitle>
              <CardDescription>
                {loading ? "Memuat data..." : `${total} work order`}
              </CardDescription>
            </div>
            <Badge variant="secondary" className="h-6 w-fit px-2.5">
              {total} WO
            </Badge>
          </div>
          <form
            className="flex flex-col gap-2 lg:flex-row lg:items-end"
            onSubmit={(event) => {
              event.preventDefault();
              list.setFilters({ q: queryInput.trim() || null, status: status || null, source: source || null });
            }}
          >
            <Field className="min-w-0 flex-1">
              <FieldLabel htmlFor="wo-search">Cari</FieldLabel>
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <Input
                  id="wo-search"
                  value={queryInput}
                  onChange={(event) => setQueryInput(event.target.value)}
                  placeholder="Kode, judul, atau unit"
                  className="pl-9"
                />
              </div>
            </Field>
            <Field className="lg:w-44">
              <FieldLabel htmlFor="wo-status">Status</FieldLabel>
              <NativeSelect
                id="wo-status"
                value={status}
                onChange={(event) => list.setFilter("status", event.target.value || null)}
              >
                <NativeSelectOption value="">Semua status</NativeSelectOption>
                {Object.entries(statusLabels).map(([value, label]) => (
                  <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field className="lg:w-40">
              <FieldLabel htmlFor="wo-source">Sumber</FieldLabel>
              <NativeSelect
                id="wo-source"
                value={source}
                onChange={(event) => list.setFilter("source", event.target.value || null)}
              >
                <NativeSelectOption value="">Semua sumber</NativeSelectOption>
                <NativeSelectOption value="p2h">P2H</NativeSelectOption>
                <NativeSelectOption value="manual">Manual</NativeSelectOption>
              </NativeSelect>
            </Field>
            <div className="flex gap-2">
              <Button type="submit" variant="outline">
                <Filter aria-hidden="true" />
                Filter
              </Button>
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setQueryInput("");
                  list.setFilters({ q: null, status: null, source: null });
                }}
              >
                <RefreshCw aria-hidden="true" />
                Reset
              </Button>
            </div>
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
                <EmptyMedia variant="icon"><Wrench aria-hidden="true" /></EmptyMedia>
                <EmptyTitle>Belum ada work order</EmptyTitle>
                <EmptyDescription>Buat work order manual atau setujui usulan P2H.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <>
              <div className="hidden overflow-x-auto md:block">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableSortHeader label="Kode" sortKey="code" currentSort={list.sort} currentOrder={list.order} onSort={list.toggleSort} />
                      <TableHead>Unit</TableHead>
                      <TableSortHeader label="Judul" sortKey="title" currentSort={list.sort} currentOrder={list.order} onSort={list.toggleSort} />
                      <TableSortHeader label="Prioritas" sortKey="priority" currentSort={list.sort} currentOrder={list.order} onSort={list.toggleSort} />
                      <TableSortHeader label="Status" sortKey="status" currentSort={list.sort} currentOrder={list.order} onSort={list.toggleSort} />
                      <TableHead>Assignee</TableHead>
                      <TableSortHeader label="Dibuat" sortKey="created_at" currentSort={list.sort} currentOrder={list.order} onSort={list.toggleSort} />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow key={item.id} className="cursor-pointer" onClick={() => openDetail(item.id)}>
                        <TableCell className="font-medium">{item.code}</TableCell>
                        <TableCell>{item.unit_code || "-"}</TableCell>
                        <TableCell className="max-w-[28rem] truncate">{item.title}</TableCell>
                        <TableCell><WorkOrderPriorityBadge priority={item.priority} /></TableCell>
                        <TableCell><WorkOrderStatusBadge status={item.status} /></TableCell>
                        <TableCell>{item.assignee_name || "-"}</TableCell>
                        <TableCell className="whitespace-nowrap text-muted-foreground">{formatDateTimeWita(item.created_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="grid gap-2 md:hidden">
                {items.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => openDetail(item.id)}
                    className="rounded-lg border border-border p-3 text-left transition-colors hover:bg-muted/40"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="font-medium">{item.code}</p>
                        <p className="truncate text-sm text-muted-foreground">{item.title}</p>
                      </div>
                      <WorkOrderStatusBadge status={item.status} />
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span>{item.unit_code || "-"}</span>
                      <span>•</span>
                      <span>{item.assignee_name || "Belum di-assign"}</span>
                      <WorkOrderPriorityBadge priority={item.priority} />
                    </div>
                  </button>
                ))}
              </div>

              <TablePagination
                page={list.page}
                pageSize={list.pageSize}
                total={total}
                totalPages={totalPages}
                loading={loading}
                itemLabel="work order"
                onPageChange={list.setPage}
                onPageSizeChange={list.setPageSize}
              />
            </>
          )}
        </CardContent>
      </Card>

      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>Buat work order manual</SheetTitle>
            <SheetDescription>Work order manual memakai source manual dan status awal open.</SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitCreate}>
            <Field>
              <FieldLabel htmlFor="create-unit">Unit</FieldLabel>
              <NativeSelect
                id="create-unit"
                value={createForm.unit_id}
                onChange={(event) => setCreateForm((current) => ({ ...current, unit_id: event.target.value }))}
                required
              >
                <NativeSelectOption value="">Pilih unit</NativeSelectOption>
                {units.map((unit) => (
                  <NativeSelectOption key={unit.id} value={String(unit.id)}>
                    {unit.code} — {unit.merk || unit.unit_type || "unit"}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field>
              <FieldLabel htmlFor="create-title">Judul</FieldLabel>
              <Input
                id="create-title"
                value={createForm.title}
                onChange={(event) => setCreateForm((current) => ({ ...current, title: event.target.value }))}
                minLength={3}
                maxLength={200}
                required
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="create-description">Deskripsi</FieldLabel>
              <Textarea
                id="create-description"
                value={createForm.description}
                onChange={(event) => setCreateForm((current) => ({ ...current, description: event.target.value }))}
                maxLength={2000}
              />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="create-priority">Prioritas</FieldLabel>
                <NativeSelect
                  id="create-priority"
                  value={createForm.priority}
                  onChange={(event) => setCreateForm((current) => ({ ...current, priority: event.target.value }))}
                >
                  {Object.entries(priorityLabels).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="create-maintenance">Tipe maintenance</FieldLabel>
                <NativeSelect
                  id="create-maintenance"
                  value={createForm.maintenance_type}
                  onChange={(event) => setCreateForm((current) => ({ ...current, maintenance_type: event.target.value }))}
                >
                  {Object.entries(maintenanceLabels).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="create-due">Due date</FieldLabel>
              <Input
                id="create-due"
                type="date"
                value={createForm.due_date}
                onChange={(event) => setCreateForm((current) => ({ ...current, due_date: event.target.value }))}
              />
            </Field>
            {createError ? <FieldError>{createError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" disabled={createBusy}>
                {createBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                Simpan WO
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <Sheet open={detailID !== null} onOpenChange={(open) => { if (!open) { setDetailID(null); setDetail(null); } }}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          {detailLoading ? (
            <div className="flex flex-col gap-3 p-4">
              <Skeleton className="h-8 w-48" />
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-40 w-full" />
            </div>
          ) : detailError && !detail ? (
            <div className="p-4">
              <Alert variant="destructive">
                <CircleAlert aria-hidden="true" />
                <AlertTitle>Detail tidak dapat dimuat</AlertTitle>
                <AlertDescription>{detailError}</AlertDescription>
              </Alert>
            </div>
          ) : detail ? (
            <>
              <SheetHeader>
                <div className="flex flex-wrap items-center gap-2">
                  <SheetTitle>{detail.code}</SheetTitle>
                  <WorkOrderStatusBadge status={detail.status} />
                  <WorkOrderPriorityBadge priority={detail.priority} />
                </div>
                <SheetDescription>{detail.title}</SheetDescription>
              </SheetHeader>
              <div className="flex flex-col gap-5 px-4 pb-4">
                {detailError ? <FieldError>{detailError}</FieldError> : null}

                <div className="grid grid-cols-2 gap-3 rounded-lg border border-border p-3 text-sm">
                  <Info label="Unit" value={detail.unit_code || "-"} />
                  <Info label="Sumber" value={detail.source === "p2h" ? "P2H" : "Manual"} />
                  <Info label="Maintenance" value={maintenanceLabels[detail.maintenance_type] || detail.maintenance_type} />
                  <Info label="Assignee" value={detail.assignee_name || "Belum di-assign"} />
                  <Info label="Supervisor" value={detail.supervisor_name || "-"} />
                  <Info label="Due date" value={detail.due_date ? formatWita(detail.due_date, { dateStyle: "medium" }) : "-"} />
                  <Info label="Mulai" value={formatDateTimeWita(detail.started_at)} />
                  <Info label="Selesai" value={formatDateTimeWita(detail.closed_at)} />
                </div>

                {detail.description ? (
                  <div>
                    <h3 className="text-sm font-semibold">Deskripsi</h3>
                    <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">{detail.description}</p>
                  </div>
                ) : null}

                {canAct ? (
                  <div className="rounded-lg border border-border p-3">
                    <h3 className="text-sm font-semibold">Aksi status</h3>
                    <Field className="mt-2">
                      <FieldLabel htmlFor="wo-action-note">Catatan / alasan</FieldLabel>
                      <Textarea
                        id="wo-action-note"
                        value={actionNote}
                        onChange={(event) => setActionNote(event.target.value)}
                        placeholder="Wajib untuk waiting part, done, dan cancel (min. 5 karakter)"
                        maxLength={1000}
                      />
                    </Field>
                    <div className="mt-3 flex flex-wrap gap-2">
                      {detail.status === "open" ? (
                        <Button type="button" size="sm" onClick={() => void transition("in_progress")} disabled={acting}>
                          <Play aria-hidden="true" /> Mulai
                        </Button>
                      ) : null}
                      {detail.status === "in_progress" ? (
                        <>
                          <Button type="button" size="sm" variant="outline" onClick={() => void transition("waiting_part")} disabled={acting}>
                            <Pause aria-hidden="true" /> Waiting part
                          </Button>
                          <Button type="button" size="sm" onClick={() => void transition("done")} disabled={acting}>
                            <CheckCircle2 aria-hidden="true" /> Selesai
                          </Button>
                        </>
                      ) : null}
                      {detail.status === "waiting_part" ? (
                        <Button type="button" size="sm" onClick={() => void transition("in_progress")} disabled={acting}>
                          <Play aria-hidden="true" /> Lanjutkan
                        </Button>
                      ) : null}
                      {canCancel ? (
                        <Button type="button" size="sm" variant="destructive" onClick={() => void transition("cancelled")} disabled={acting}>
                          <XCircle aria-hidden="true" /> Cancel
                        </Button>
                      ) : null}
                    </div>
                  </div>
                ) : null}

                {detail.findings.length > 0 ? (
                  <div>
                    <h3 className="text-sm font-semibold">Temuan P2H</h3>
                    <div className="mt-2 overflow-x-auto rounded-lg border border-border">
                      <Table>
                        <TableHeader><TableRow><TableHead>Komponen</TableHead><TableHead>Kategori</TableHead><TableHead>Kondisi</TableHead><TableHead>Remark</TableHead></TableRow></TableHeader>
                        <TableBody>
                          {detail.findings.map((finding) => (
                            <TableRow key={finding.id}>
                              <TableCell>{finding.component}</TableCell>
                              <TableCell>{finding.category}</TableCell>
                              <TableCell>{finding.condition}</TableCell>
                              <TableCell>{finding.remark || "-"}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  </div>
                ) : null}

                <div>
                  <h3 className="text-sm font-semibold">Riwayat</h3>
                  <div className="mt-2 flex flex-col gap-2">
                    {detail.updates.length === 0 ? (
                      <p className="text-sm text-muted-foreground">Belum ada catatan.</p>
                    ) : detail.updates.map((update: WorkOrderUpdate) => (
                      <div key={update.id} className="rounded-lg border border-border p-2 text-sm">
                        <div className="flex flex-wrap items-center justify-between gap-1 text-xs text-muted-foreground">
                          <span>{update.user_name || "Sistem"}</span>
                          <span>{formatDateTimeWita(update.created_at)}</span>
                        </div>
                        {update.status_from || update.status_to ? (
                          <p className="mt-1 text-xs text-muted-foreground">
                            {statusLabels[update.status_from] || update.status_from || "-"} → {statusLabels[update.status_to] || update.status_to || "-"}
                          </p>
                        ) : null}
                        {update.note ? <p className="mt-1 whitespace-pre-wrap">{update.note}</p> : null}
                      </div>
                    ))}
                  </div>
                  <div className="mt-3 flex flex-col gap-2">
                    <Textarea
                      value={comment}
                      onChange={(event) => setComment(event.target.value)}
                      placeholder="Tambah catatan..."
                      maxLength={1000}
                    />
                    <Button type="button" size="sm" variant="outline" onClick={() => void addComment()} disabled={acting || !comment.trim()}>
                      <MessageSquare aria-hidden="true" /> Tambah catatan
                    </Button>
                  </div>
                </div>

                <div>
                  <h3 className="text-sm font-semibold">Lampiran</h3>
                  <div className="mt-2 flex flex-col gap-2">
                    {detail.attachments.length === 0 ? (
                      <p className="text-sm text-muted-foreground">Belum ada lampiran.</p>
                    ) : detail.attachments.map((item) => (
                      <a
                        key={item.id}
                        href={`/api/proxy/workorders/${detail.id}/attachments/${item.id}`}
                        target="_blank"
                        rel="noreferrer"
                        className="flex items-center gap-2 rounded-lg border border-border p-2 text-sm hover:bg-muted/40"
                      >
                        <Paperclip aria-hidden="true" className="size-4" />
                        <span className="truncate">{item.file_path.split("/").pop()}</span>
                        <span className="ml-auto text-xs text-muted-foreground">{Math.ceil(item.size_bytes / 1024)} KB</span>
                      </a>
                    ))}
                  </div>
                  <div className="mt-3 flex flex-col gap-2">
                    <Input
                      type="file"
                      accept="image/jpeg,image/png"
                      onChange={(event) => setAttachment(event.target.files?.[0] ?? null)}
                    />
                    <Button type="button" size="sm" variant="outline" onClick={() => void uploadAttachment()} disabled={uploading || !attachment}>
                      {uploading ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Paperclip aria-hidden="true" />}
                      Unggah JPG/PNG
                    </Button>
                  </div>
                </div>
              </div>
            </>
          ) : null}
        </SheetContent>
      </Sheet>
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="break-words font-medium">{value}</p>
    </div>
  );
}
