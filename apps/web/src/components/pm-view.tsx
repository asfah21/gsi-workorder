"use client";

import {
  CalendarClock,
  CheckCircle2,
  CircleAlert,
  ClipboardList,
  Loader2,
  PackageSearch,
  Pencil,
  Play,
  Plus,
  RefreshCw,
  ShieldCheck,
  SkipForward,
  Trash2,
  XCircle,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import { PageHeader } from "@/components/layout/page-header";
import { FilterBar } from "@/components/list/list-filter-bar";
import { SearchInput } from "@/components/list/list-search";
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
import { ApiError, proxyDelete, proxyGet, proxyPatch, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  ListResponse,
  PMCompliance,
  PMComplianceRow,
  PMPlan,
  PMPlanMaterial,
  PMSchedule,
  Sparepart,
  Unit,
  UnitModel,
  UserOption,
  WorkOrderPriority,
} from "@/lib/types";
import { formatWita, toDatetimeLocalWita } from "@/lib/time";
import { cn } from "@/lib/utils";

const planStatusLabels: Record<string, string> = {
  draft: "Draft",
  active: "Active",
  inactive: "Inactive",
};

const bucketLabels: Record<string, string> = {
  scheduled: "Scheduled",
  upcoming: "Upcoming",
  due_soon: "Due soon",
  due: "Due",
  overdue: "Overdue",
  wo_created: "WO created",
  completed: "Completed",
  skipped: "Skipped",
  cancelled: "Cancelled",
};

const priorityLabels: Record<WorkOrderPriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

type ChecklistDraft = {
  task: string;
  standard: string;
};

type PlanForm = {
  name: string;
  description: string;
  target_type: "unit" | "model";
  unit_id: string;
  unit_model_id: string;
  status: string;
  effective_from: string;
  effective_to: string;
  interval_hm: string;
  interval_km: string;
  interval_days: string;
  interval_rule: string;
  upcoming_days: string;
  due_soon_days: string;
  upcoming_hm: string;
  upcoming_km: string;
  due_soon_hm: string;
  due_soon_km: string;
  auto_generate_wo: boolean;
  default_priority: WorkOrderPriority;
  default_assignee_id: string;
  default_supervisor_id: string;
  checklist: ChecklistDraft[];
};

const emptyPlanForm: PlanForm = {
  name: "",
  description: "",
  target_type: "unit",
  unit_id: "",
  unit_model_id: "",
  status: "draft",
  effective_from: toDatetimeLocalWita().slice(0, 10),
  effective_to: "",
  interval_hm: "",
  interval_km: "",
  interval_days: "",
  interval_rule: "first_reached",
  upcoming_days: "30",
  due_soon_days: "7",
  upcoming_hm: "",
  upcoming_km: "",
  due_soon_hm: "",
  due_soon_km: "",
  auto_generate_wo: false,
  default_priority: "medium",
  default_assignee_id: "",
  default_supervisor_id: "",
  checklist: [{ task: "", standard: "" }],
};

export function PMView() {
  const { user } = useSession();
  const canManage = user.role === "admin" || user.role === "planner";

  const [plans, setPlans] = useState<PMPlan[]>([]);
  const [plansTotal, setPlansTotal] = useState(0);
  const [schedules, setSchedules] = useState<PMSchedule[]>([]);
  const [schedulesTotal, setSchedulesTotal] = useState(0);
  const [compliance, setCompliance] = useState<PMCompliance | null>(null);
  const [units, setUnits] = useState<Unit[]>([]);
  const [models, setModels] = useState<UnitModel[]>([]);
  const [users, setUsers] = useState<UserOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [schedulesLoading, setSchedulesLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const [bucket, setBucket] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [scheduleQuery, setScheduleQuery] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);

  const [planOpen, setPlanOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<PMPlan | null>(null);
  const [planForm, setPlanForm] = useState<PlanForm>(emptyPlanForm);
  const [planBusy, setPlanBusy] = useState(false);
  const [planError, setPlanError] = useState("");

  const [actionSchedule, setActionSchedule] = useState<PMSchedule | null>(null);
  const [actionKind, setActionKind] = useState<"override" | "skip" | "cancel" | null>(null);
  const [actionReason, setActionReason] = useState("");
  const [actionDueHM, setActionDueHM] = useState("");
  const [actionDueKM, setActionDueKM] = useState("");
  const [actionDueDate, setActionDueDate] = useState("");
  const [actionBusy, setActionBusy] = useState(false);
  const [actionError, setActionError] = useState("");

  const [materialPlan, setMaterialPlan] = useState<PMPlan | null>(null);
  const [materials, setMaterials] = useState<PMPlanMaterial[]>([]);
  const [materialOptions, setMaterialOptions] = useState<Sparepart[]>([]);
  const [materialLoading, setMaterialLoading] = useState(false);
  const [materialError, setMaterialError] = useState("");
  const [materialSparepartID, setMaterialSparepartID] = useState("");
  const [materialQty, setMaterialQty] = useState("1");
  const [materialOptional, setMaterialOptional] = useState(false);
  const [materialNotes, setMaterialNotes] = useState("");
  const [materialBusy, setMaterialBusy] = useState(false);

  const assigneeOptions = users.filter((item) => ["admin", "planner", "mechanic"].includes(item.role));
  const supervisorOptions = users.filter((item) => ["admin", "planner"].includes(item.role));

  const loadPlans = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [planResult, complianceResult] = await Promise.all([
        proxyGet<ListResponse<PMPlan>>("/pm/plans?limit=100&sort=name&order=asc"),
        proxyGet<DataResponse<PMCompliance>>("/pm/compliance"),
      ]);
      setPlans(planResult.data);
      setPlansTotal(planResult.meta.total);
      setCompliance(complianceResult.data);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "PM plan tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, []);

  const loadSchedules = useCallback(async () => {
    setSchedulesLoading(true);
    try {
      const terminalBucket = ["completed", "skipped", "cancelled"].includes(bucket);
      const params = new URLSearchParams({
        page: String(page),
        limit: String(pageSize),
        ...(bucket ? { bucket } : {}),
        ...(!showHistory && !terminalBucket ? { active_only: "true" } : {}),
        ...(scheduleQuery.trim() ? { q: scheduleQuery.trim() } : {}),
      });
      const result = await proxyGet<ListResponse<PMSchedule>>(`/pm/schedules?${params.toString()}`);
      setSchedules(result.data);
      setSchedulesTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Schedule PM tidak dapat dimuat.");
    } finally {
      setSchedulesLoading(false);
    }
  }, [bucket, page, pageSize, scheduleQuery, showHistory]);

  const loadReferences = useCallback(async () => {
    try {
      const [unitResult, modelResult, userResult] = await Promise.all([
        proxyGet<ListResponse<Unit>>("/units?limit=100&sort=code&order=asc"),
        proxyGet<ListResponse<UnitModel>>("/unit-models?limit=100&active_only=true&sort=manufacturer&order=asc"),
        canManage
          ? proxyGet<ListResponse<UserOption>>("/users?roles=admin,planner,mechanic&limit=1000")
          : Promise.resolve({ data: [], meta: { page: 1, limit: 1000, total: 0 } }),
      ]);
      setUnits(unitResult.data);
      setModels(modelResult.data);
      setUsers(userResult.data);
    } catch {
      // Reference data is best-effort; the list views remain usable.
    }
  }, [canManage]);

  useEffect(() => {
    void loadPlans();
    void loadReferences();
  }, [loadPlans, loadReferences]);

  useEffect(() => {
    void loadSchedules();
  }, [loadSchedules]);

  const openCreate = () => {
    setEditingPlan(null);
    setPlanForm({ ...emptyPlanForm, checklist: [{ task: "", standard: "" }] });
    setPlanError("");
    setPlanOpen(true);
  };

  const openEdit = (plan: PMPlan) => {
    setEditingPlan(plan);
    setPlanForm({
      name: plan.name,
      description: plan.description,
      target_type: plan.unit_id ? "unit" : "model",
      unit_id: plan.unit_id ? String(plan.unit_id) : "",
      unit_model_id: plan.unit_model_id ? String(plan.unit_model_id) : "",
      status: plan.status,
      effective_from: plan.effective_from.slice(0, 10),
      effective_to: plan.effective_to ? plan.effective_to.slice(0, 10) : "",
      interval_hm: plan.interval_hm != null ? String(plan.interval_hm) : "",
      interval_km: plan.interval_km != null ? String(plan.interval_km) : "",
      interval_days: plan.interval_days != null ? String(plan.interval_days) : "",
      interval_rule: plan.interval_rule,
      upcoming_days: plan.upcoming_days != null ? String(plan.upcoming_days) : "",
      due_soon_days: plan.due_soon_days != null ? String(plan.due_soon_days) : "",
      upcoming_hm: plan.upcoming_hm != null ? String(plan.upcoming_hm) : "",
      upcoming_km: plan.upcoming_km != null ? String(plan.upcoming_km) : "",
      due_soon_hm: plan.due_soon_hm != null ? String(plan.due_soon_hm) : "",
      due_soon_km: plan.due_soon_km != null ? String(plan.due_soon_km) : "",
      auto_generate_wo: plan.auto_generate_wo,
      default_priority: plan.default_priority,
      default_assignee_id: plan.default_assignee_id ? String(plan.default_assignee_id) : "",
      default_supervisor_id: plan.default_supervisor_id ? String(plan.default_supervisor_id) : "",
      checklist: plan.checklist.length > 0
        ? plan.checklist.map((item) => ({ task: item.task, standard: item.standard }))
        : [{ task: "", standard: "" }],
    });
    setPlanError("");
    setPlanOpen(true);
  };

  const submitPlan = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (planBusy) return;
    setPlanBusy(true);
    setPlanError("");
    try {
      const checklist = planForm.checklist
        .filter((item) => item.task.trim())
        .map((item, index) => ({
          sequence: index + 1,
          task: item.task.trim(),
          standard: item.standard.trim(),
        }));
      const common = {
        name: planForm.name.trim(),
        description: planForm.description.trim(),
        effective_from: planForm.effective_from,
        effective_to: planForm.effective_to || null,
        interval_hm: planForm.interval_hm ? Number(planForm.interval_hm) : null,
        interval_km: planForm.interval_km ? Number(planForm.interval_km) : null,
        interval_days: planForm.interval_days ? Number(planForm.interval_days) : null,
        interval_rule: planForm.interval_rule,
        upcoming_days: planForm.upcoming_days ? Number(planForm.upcoming_days) : null,
        due_soon_days: planForm.due_soon_days ? Number(planForm.due_soon_days) : null,
        upcoming_hm: planForm.upcoming_hm ? Number(planForm.upcoming_hm) : null,
        upcoming_km: planForm.upcoming_km ? Number(planForm.upcoming_km) : null,
        due_soon_hm: planForm.due_soon_hm ? Number(planForm.due_soon_hm) : null,
        due_soon_km: planForm.due_soon_km ? Number(planForm.due_soon_km) : null,
        auto_generate_wo: planForm.auto_generate_wo,
        default_priority: planForm.default_priority,
        default_assignee_id: planForm.default_assignee_id ? Number(planForm.default_assignee_id) : null,
        default_supervisor_id: planForm.default_supervisor_id ? Number(planForm.default_supervisor_id) : null,
        checklist,
      };

      if (editingPlan) {
        await proxyPatch<DataResponse<PMPlan>>(`/pm/plans/${editingPlan.id}`, {
          ...common,
          status: planForm.status,
          clear_effective_to: !planForm.effective_to,
          clear_interval_hm: !planForm.interval_hm,
          clear_interval_km: !planForm.interval_km,
          clear_interval_days: !planForm.interval_days,
          clear_upcoming_days: !planForm.upcoming_days,
          clear_due_soon_days: !planForm.due_soon_days,
          clear_upcoming_hm: !planForm.upcoming_hm,
          clear_upcoming_km: !planForm.upcoming_km,
          clear_due_soon_hm: !planForm.due_soon_hm,
          clear_due_soon_km: !planForm.due_soon_km,
          clear_default_assignee: !planForm.default_assignee_id,
          clear_default_supervisor: !planForm.default_supervisor_id,
        });
        setSuccess(`Plan ${planForm.name} diperbarui.`);
      } else {
        await proxyPost<DataResponse<PMPlan>>("/pm/plans", {
          ...common,
          unit_id: planForm.target_type === "unit" ? Number(planForm.unit_id) : null,
          unit_model_id: planForm.target_type === "model" ? Number(planForm.unit_model_id) : null,
          status: planForm.status,
        });
        setSuccess(`Plan ${planForm.name} dibuat.`);
      }
      setPlanOpen(false);
      await loadPlans();
      await loadSchedules();
    } catch (saveError) {
      setPlanError(saveError instanceof ApiError ? saveError.message : "Plan PM tidak dapat disimpan.");
    } finally {
      setPlanBusy(false);
    }
  };

  const loadMaterials = useCallback(async (planID: number) => {
    const result = await proxyGet<DataResponse<PMPlanMaterial[]>>(`/pm/plans/${planID}/materials`);
    setMaterials(result.data);
  }, []);

  const openMaterials = async (plan: PMPlan) => {
    setMaterialPlan(plan);
    setMaterialError("");
    setMaterialLoading(true);
    setMaterialSparepartID("");
    setMaterialQty("1");
    setMaterialOptional(false);
    setMaterialNotes("");
    try {
      const [, optionResult] = await Promise.all([
        loadMaterials(plan.id),
        proxyGet<ListResponse<Sparepart>>("/spareparts?limit=100&active_only=true&sort=code&order=asc"),
      ]);
      setMaterialOptions(optionResult.data);
    } catch (loadError) {
      setMaterialError(loadError instanceof ApiError ? loadError.message : "Material plan tidak dapat dimuat.");
    } finally {
      setMaterialLoading(false);
    }
  };

  const addMaterial = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!materialPlan || materialBusy) return;
    const qty = Number(materialQty);
    if (!materialSparepartID) {
      setMaterialError("Pilih sparepart terlebih dahulu.");
      return;
    }
    if (!Number.isFinite(qty) || qty <= 0) {
      setMaterialError("Qty material harus lebih dari 0.");
      return;
    }
    setMaterialBusy(true);
    setMaterialError("");
    try {
      await proxyPost<DataResponse<PMPlanMaterial>>(`/pm/plans/${materialPlan.id}/materials`, {
        sparepart_id: Number(materialSparepartID),
        qty,
        is_optional: materialOptional,
        notes: materialNotes.trim(),
      });
      setMaterialSparepartID("");
      setMaterialQty("1");
      setMaterialOptional(false);
      setMaterialNotes("");
      setSuccess(`Material plan ${materialPlan.name} diperbarui.`);
      await loadMaterials(materialPlan.id);
    } catch (saveError) {
      setMaterialError(saveError instanceof ApiError ? saveError.message : "Material plan tidak dapat disimpan.");
    } finally {
      setMaterialBusy(false);
    }
  };

  const removeMaterial = async (material: PMPlanMaterial) => {
    if (!materialPlan) return;
    if (!window.confirm(`Hapus ${material.sparepart_code} - ${material.sparepart_name} dari material plan?`)) return;
    setMaterialError("");
    try {
      await proxyDelete(`/pm/plans/${materialPlan.id}/materials/${material.id}`);
      setSuccess(`Material ${material.sparepart_code} dihapus dari plan.`);
      await loadMaterials(materialPlan.id);
    } catch (deleteError) {
      setMaterialError(deleteError instanceof ApiError ? deleteError.message : "Material plan tidak dapat dihapus.");
    }
  };

  const changePlanStatus = async (plan: PMPlan, status: string) => {
    setError("");
    setSuccess("");
    try {
      await proxyPost<DataResponse<PMPlan>>(`/pm/plans/${plan.id}/status`, { status });
      setSuccess(`Plan ${plan.name} menjadi ${planStatusLabels[status] || status}.`);
      await loadPlans();
      await loadSchedules();
    } catch (actionError) {
      setError(actionError instanceof ApiError ? actionError.message : "Status plan tidak dapat diubah.");
    }
  };

  const generateWO = async (schedule: PMSchedule) => {
    setError("");
    setSuccess("");
    try {
      const result = await proxyPost<DataResponse<{ work_order: { code: string } }>>(
        `/pm/schedules/${schedule.id}/generate-wo`,
        {},
      );
      setSuccess(`WO ${result.data.work_order.code} dibuat dari schedule ${schedule.unit_code}.`);
      await loadSchedules();
      await loadPlans();
    } catch (actionError) {
      setError(actionError instanceof ApiError ? actionError.message : "WO PM tidak dapat dibuat.");
    }
  };

  const openAction = (schedule: PMSchedule, kind: "override" | "skip" | "cancel") => {
    setActionSchedule(schedule);
    setActionKind(kind);
    setActionReason("");
    setActionDueHM(schedule.due_hm != null ? String(schedule.due_hm) : "");
    setActionDueKM(schedule.due_km != null ? String(schedule.due_km) : "");
    setActionDueDate(schedule.due_date ? schedule.due_date.slice(0, 10) : "");
    setActionError("");
  };

  const submitAction = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!actionSchedule || !actionKind || actionBusy) return;
    setActionBusy(true);
    setActionError("");
    try {
      if (actionKind === "override") {
        await proxyPost(`/pm/schedules/${actionSchedule.id}/override-due`, {
          due_hm: actionDueHM ? Number(actionDueHM) : null,
          due_km: actionDueKM ? Number(actionDueKM) : null,
          due_date: actionDueDate || null,
          reason: actionReason,
        });
        setSuccess("Due PM berhasil di-override dan alasan dicatat.");
      } else {
        await proxyPost(`/pm/schedules/${actionSchedule.id}/${actionKind}`, {
          reason: actionReason,
        });
        setSuccess(`Schedule ${actionKind === "skip" ? "di-skip" : "dibatalkan"}.`);
      }
      setActionSchedule(null);
      setActionKind(null);
      await loadSchedules();
      await loadPlans();
    } catch (actionError) {
      setActionError(actionError instanceof ApiError ? actionError.message : "Aksi PM gagal.");
    } finally {
      setActionBusy(false);
    }
  };

  const bucketCounts = useMemo(() => {
    return schedules.reduce<Record<string, number>>((acc, item) => {
      acc[item.bucket] = (acc[item.bucket] || 0) + 1;
      return acc;
    }, {});
  }, [schedules]);

  const totalPages = Math.max(1, Math.ceil(schedulesTotal / pageSize));

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Preventive Maintenance"
        description="Plan per unit/model, interval HM/KM/calendar, schedule, dan generate WO PM."
        actions={
          canManage ? (
            <Button type="button" onClick={openCreate}>
              <Plus aria-hidden="true" /> Buat plan PM
            </Button>
          ) : null
        }
      />

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
        <CardHeader>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <ClipboardList className="size-4" aria-hidden="true" /> Plan PM
              </CardTitle>
              <CardDescription>{loading ? "Memuat plan..." : `${plansTotal} plan`}</CardDescription>
            </div>
            <Badge variant="secondary" className="h-6 w-fit px-2.5">{plansTotal} plan</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : plans.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon"><CalendarClock aria-hidden="true" /></EmptyMedia>
                <EmptyTitle>Belum ada plan PM</EmptyTitle>
                <EmptyDescription>Buat plan untuk unit atau model, lalu aktifkan untuk membuat schedule.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Plan</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead>Interval</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Auto WO</TableHead>
                    <TableHead>Checklist</TableHead>
                    <TableHead>Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {plans.map((plan) => (
                    <TableRow key={plan.id}>
                      <TableCell>
                        <p className="font-medium">{plan.name}</p>
                        <p className="text-xs text-muted-foreground">{plan.interval_rule.replace("_", " ")}</p>
                      </TableCell>
                      <TableCell>{plan.unit_code || plan.unit_model_name || "-"}</TableCell>
                      <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                        {[
                          plan.interval_hm != null ? `HM ${plan.interval_hm}` : "",
                          plan.interval_km != null ? `KM ${plan.interval_km}` : "",
                          plan.interval_days != null ? `${plan.interval_days} hari` : "",
                        ].filter(Boolean).join(" · ") || "-"}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant="secondary"
                          className={cn(
                            "h-5 px-2",
                            plan.status === "active" && "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
                            plan.status === "draft" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
                            plan.status === "inactive" && "bg-muted text-muted-foreground",
                          )}
                        >
                          {planStatusLabels[plan.status] || plan.status}
                        </Badge>
                      </TableCell>
                      <TableCell>{plan.auto_generate_wo ? "Aktif" : "Nonaktif"}</TableCell>
                      <TableCell>{plan.checklist.length}</TableCell>
                      <TableCell>
                        {canManage ? (
                          <div className="flex flex-wrap gap-1">
                            <Button type="button" size="xs" variant="outline" onClick={() => openEdit(plan)}>
                              <Pencil aria-hidden="true" /> Edit
                            </Button>
                            <Button type="button" size="xs" variant="outline" onClick={() => void openMaterials(plan)}>
                              <PackageSearch aria-hidden="true" /> Material
                            </Button>
                            {plan.status !== "active" ? (
                              <Button type="button" size="xs" onClick={() => void changePlanStatus(plan, "active")}>
                                <Play aria-hidden="true" /> Aktifkan
                              </Button>
                            ) : (
                              <Button type="button" size="xs" variant="outline" onClick={() => void changePlanStatus(plan, "inactive")}>
                                <XCircle aria-hidden="true" /> Nonaktifkan
                              </Button>
                            )}
                          </div>
                        ) : (
                          <span className="text-xs text-muted-foreground">Read-only</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="gap-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <CalendarClock className="size-4" aria-hidden="true" /> Schedule PM
              </CardTitle>
              <CardDescription>
                {schedulesLoading ? "Memuat schedule..." : `${schedulesTotal} schedule · ${bucketCounts.overdue || 0} overdue`}
              </CardDescription>
            </div>
            <Badge variant="secondary" className="h-6 w-fit px-2.5">{bucketLabels[bucket] || "Semua"}</Badge>
          </div>
          <FilterBar
            onSubmit={(event) => {
              event.preventDefault();
              setPage(1);
              void loadSchedules();
            }}
          >
            <Field className="min-w-0 flex-1">
              <FieldLabel htmlFor="pm-search">Cari</FieldLabel>
              <SearchInput
                id="pm-search"
                value={scheduleQuery}
                onChange={(event) => setScheduleQuery(event.target.value)}
                placeholder="Plan atau unit"
              />
            </Field>
            <Field className="md:w-52">
              <FieldLabel htmlFor="pm-bucket">Bucket</FieldLabel>
              <NativeSelect
                id="pm-bucket"
                value={bucket}
                onChange={(event) => {
                  setBucket(event.target.value);
                  setPage(1);
                }}
              >
                <NativeSelectOption value="">Semua</NativeSelectOption>
                <NativeSelectOption value="upcoming">Upcoming</NativeSelectOption>
                <NativeSelectOption value="due_soon">Due soon</NativeSelectOption>
                <NativeSelectOption value="due">Due</NativeSelectOption>
                <NativeSelectOption value="overdue">Overdue</NativeSelectOption>
                <NativeSelectOption value="wo_created">WO created</NativeSelectOption>
                <NativeSelectOption value="completed">Completed</NativeSelectOption>
                <NativeSelectOption value="skipped">Skipped</NativeSelectOption>
                <NativeSelectOption value="cancelled">Cancelled</NativeSelectOption>
              </NativeSelect>
            </Field>
            <label className="flex h-9 cursor-pointer items-center gap-2 text-sm text-muted-foreground">
              <input
                type="checkbox"
                checked={showHistory}
                onChange={(event) => {
                  setShowHistory(event.target.checked);
                  setPage(1);
                }}
                className="size-4 rounded border-input accent-primary"
              />
              Riwayat
            </label>
            <Button type="button" variant="ghost" onClick={() => void loadSchedules()}>
              <RefreshCw aria-hidden="true" /> Refresh
            </Button>
          </FilterBar>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {schedulesLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : schedules.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon"><CalendarClock aria-hidden="true" /></EmptyMedia>
                <EmptyTitle>Tidak ada schedule</EmptyTitle>
                <EmptyDescription>Aktifkan plan atau ubah filter bucket.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Plan / Unit</TableHead>
                      <TableHead>Siklus</TableHead>
                      <TableHead>Due</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>WO</TableHead>
                      <TableHead>Aksi</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {schedules.map((schedule) => (
                      <TableRow key={schedule.id}>
                        <TableCell>
                          <p className="font-medium">{schedule.unit_code}</p>
                          <p className="text-xs text-muted-foreground">{schedule.plan_name}</p>
                        </TableCell>
                        <TableCell>#{schedule.cycle_no}</TableCell>
                        <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                          {[
                            schedule.due_hm != null ? `HM ${schedule.due_hm}` : "",
                            schedule.due_km != null ? `KM ${schedule.due_km}` : "",
                            schedule.due_date ? formatWita(schedule.due_date, { dateStyle: "medium" }) : "",
                          ].filter(Boolean).join(" · ") || "-"}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="secondary"
                            className={cn(
                              "h-5 px-2",
                              schedule.bucket === "overdue" && "bg-destructive/10 text-destructive",
                              schedule.bucket === "due" && "bg-amber-500/10 text-amber-700 dark:text-amber-300",
                              schedule.bucket === "due_soon" && "bg-orange-500/10 text-orange-700 dark:text-orange-300",
                              schedule.bucket === "wo_created" && "bg-sky-500/10 text-sky-700 dark:text-sky-300",
                              schedule.bucket === "completed" && "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
                            )}
                          >
                            {bucketLabels[schedule.bucket] || schedule.bucket}
                          </Badge>
                        </TableCell>
                        <TableCell>{schedule.work_order_code || "-"}</TableCell>
                        <TableCell>
                          {canManage && ["scheduled", "due", "overdue"].includes(schedule.status) ? (
                            <div className="flex flex-wrap gap-1">
                              <Button type="button" size="xs" onClick={() => void generateWO(schedule)}>
                                <Play aria-hidden="true" /> WO
                              </Button>
                              <Button type="button" size="xs" variant="outline" onClick={() => openAction(schedule, "override")}>
                                Override
                              </Button>
                              <Button type="button" size="xs" variant="ghost" onClick={() => openAction(schedule, "skip")}>
                                <SkipForward aria-hidden="true" /> Skip
                              </Button>
                            </div>
                          ) : (
                            <span className="text-xs text-muted-foreground">
                              {schedule.work_order_code ? `WO ${schedule.work_order_code}` : "-"}
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              <TablePagination
                page={page}
                pageSize={pageSize}
                total={schedulesTotal}
                totalPages={totalPages}
                loading={schedulesLoading}
                itemLabel="schedule"
                onPageChange={setPage}
                onPageSizeChange={(size) => {
                  setPageSize(size);
                  setPage(1);
                }}
              />
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="size-4" aria-hidden="true" /> PM compliance
          </CardTitle>
          <CardDescription>Dihitung dari siklus WO PM yang selesai, overdue, skipped, dan cancelled.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Metric label="Total siklus" value={compliance?.total ?? 0} />
            <Metric label="Selesai" value={compliance?.completed ?? 0} />
            <Metric label="Overdue" value={compliance?.overdue ?? 0} />
            <Metric label="Completion" value={`${compliance?.completion_rate ?? 0}%`} />
          </div>
          {compliance && compliance.rows.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Plan</TableHead>
                    <TableHead>Total</TableHead>
                    <TableHead>Selesai</TableHead>
                    <TableHead>On time</TableHead>
                    <TableHead>Overdue</TableHead>
                    <TableHead>Missed</TableHead>
                    <TableHead>Rate</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {compliance.rows.map((row: PMComplianceRow) => (
                    <TableRow key={row.pm_plan_id}>
                      <TableCell className="font-medium">{row.plan_name}</TableCell>
                      <TableCell>{row.total}</TableCell>
                      <TableCell>{row.completed}</TableCell>
                      <TableCell>{row.on_time}</TableCell>
                      <TableCell>{row.overdue}</TableCell>
                      <TableCell>{row.missed}</TableCell>
                      <TableCell>{row.completion_rate}%</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Belum ada data compliance.</p>
          )}
        </CardContent>
      </Card>

      <Sheet open={planOpen} onOpenChange={setPlanOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>{editingPlan ? "Edit plan PM" : "Buat plan PM"}</SheetTitle>
            <SheetDescription>
              {editingPlan
                ? "Perubahan interval memengaruhi siklus berikutnya; siklus yang sudah WO-created tidak diubah."
                : "Target plan tidak dapat diubah setelah dibuat. Aktifkan plan untuk membuat schedule."}
            </SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitPlan}>
            <Field>
              <FieldLabel htmlFor="pm-name">Nama plan</FieldLabel>
              <Input
                id="pm-name"
                value={planForm.name}
                onChange={(event) => setPlanForm((current) => ({ ...current, name: event.target.value }))}
                minLength={3}
                maxLength={200}
                required
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="pm-description">Deskripsi</FieldLabel>
              <Textarea
                id="pm-description"
                value={planForm.description}
                onChange={(event) => setPlanForm((current) => ({ ...current, description: event.target.value }))}
                maxLength={2000}
              />
            </Field>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="pm-target-type">Target</FieldLabel>
                <NativeSelect
                  id="pm-target-type"
                  value={planForm.target_type}
                  onChange={(event) => setPlanForm((current) => ({ ...current, target_type: event.target.value as "unit" | "model" }))}
                  disabled={Boolean(editingPlan)}
                >
                  <NativeSelectOption value="unit">Per unit</NativeSelectOption>
                  <NativeSelectOption value="model">Per model</NativeSelectOption>
                </NativeSelect>
              </Field>
              {planForm.target_type === "unit" ? (
                <Field>
                  <FieldLabel htmlFor="pm-unit">Unit</FieldLabel>
                  <NativeSelect
                    id="pm-unit"
                    value={planForm.unit_id}
                    onChange={(event) => setPlanForm((current) => ({ ...current, unit_id: event.target.value }))}
                    disabled={Boolean(editingPlan)}
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
              ) : (
                <Field>
                  <FieldLabel htmlFor="pm-model">Model unit</FieldLabel>
                  <NativeSelect
                    id="pm-model"
                    value={planForm.unit_model_id}
                    onChange={(event) => setPlanForm((current) => ({ ...current, unit_model_id: event.target.value }))}
                    disabled={Boolean(editingPlan)}
                    required
                  >
                    <NativeSelectOption value="">Pilih model</NativeSelectOption>
                    {models.map((model) => (
                      <NativeSelectOption key={model.id} value={String(model.id)}>
                        {model.manufacturer} {model.model}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
              )}
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
              <Field>
                <FieldLabel htmlFor="pm-status">Status</FieldLabel>
                <NativeSelect
                  id="pm-status"
                  value={planForm.status}
                  onChange={(event) => setPlanForm((current) => ({ ...current, status: event.target.value }))}
                >
                  <NativeSelectOption value="draft">Draft</NativeSelectOption>
                  <NativeSelectOption value="active">Active</NativeSelectOption>
                  <NativeSelectOption value="inactive">Inactive</NativeSelectOption>
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-effective-from">Efektif dari</FieldLabel>
                <Input
                  id="pm-effective-from"
                  type="date"
                  value={planForm.effective_from}
                  onChange={(event) => setPlanForm((current) => ({ ...current, effective_from: event.target.value }))}
                  required
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-effective-to">Efektif sampai</FieldLabel>
                <Input
                  id="pm-effective-to"
                  type="date"
                  value={planForm.effective_to}
                  onChange={(event) => setPlanForm((current) => ({ ...current, effective_to: event.target.value }))}
                />
              </Field>
            </div>

            <div className="grid gap-4 sm:grid-cols-4">
              <Field>
                <FieldLabel htmlFor="pm-interval-hm">Interval HM</FieldLabel>
                <Input id="pm-interval-hm" type="number" min={0.1} step="0.1" value={planForm.interval_hm} onChange={(event) => setPlanForm((current) => ({ ...current, interval_hm: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-interval-km">Interval KM</FieldLabel>
                <Input id="pm-interval-km" type="number" min={0.1} step="0.1" value={planForm.interval_km} onChange={(event) => setPlanForm((current) => ({ ...current, interval_km: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-interval-days">Interval hari</FieldLabel>
                <Input id="pm-interval-days" type="number" min={1} value={planForm.interval_days} onChange={(event) => setPlanForm((current) => ({ ...current, interval_days: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-rule">Aturan</FieldLabel>
                <NativeSelect id="pm-rule" value={planForm.interval_rule} onChange={(event) => setPlanForm((current) => ({ ...current, interval_rule: event.target.value }))}>
                  <NativeSelectOption value="first_reached">First reached</NativeSelectOption>
                  <NativeSelectOption value="all_required">All required</NativeSelectOption>
                </NativeSelect>
              </Field>
            </div>

            <div className="grid gap-4 sm:grid-cols-4">
              <Field>
                <FieldLabel htmlFor="pm-upcoming-days">Upcoming hari</FieldLabel>
                <Input id="pm-upcoming-days" type="number" min={0} max={365} value={planForm.upcoming_days} onChange={(event) => setPlanForm((current) => ({ ...current, upcoming_days: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-due-soon-days">Due soon hari</FieldLabel>
                <Input id="pm-due-soon-days" type="number" min={0} max={365} value={planForm.due_soon_days} onChange={(event) => setPlanForm((current) => ({ ...current, due_soon_days: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-upcoming-hm">Upcoming HM</FieldLabel>
                <Input id="pm-upcoming-hm" type="number" min={0} step="0.1" value={planForm.upcoming_hm} onChange={(event) => setPlanForm((current) => ({ ...current, upcoming_hm: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-due-soon-hm">Due soon HM</FieldLabel>
                <Input id="pm-due-soon-hm" type="number" min={0} step="0.1" value={planForm.due_soon_hm} onChange={(event) => setPlanForm((current) => ({ ...current, due_soon_hm: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-upcoming-km">Upcoming KM</FieldLabel>
                <Input id="pm-upcoming-km" type="number" min={0} step="0.1" value={planForm.upcoming_km} onChange={(event) => setPlanForm((current) => ({ ...current, upcoming_km: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-due-soon-km">Due soon KM</FieldLabel>
                <Input id="pm-due-soon-km" type="number" min={0} step="0.1" value={planForm.due_soon_km} onChange={(event) => setPlanForm((current) => ({ ...current, due_soon_km: event.target.value }))} />
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-priority">Prioritas WO</FieldLabel>
                <NativeSelect id="pm-priority" value={planForm.default_priority} onChange={(event) => setPlanForm((current) => ({ ...current, default_priority: event.target.value as WorkOrderPriority }))}>
                  {Object.entries(priorityLabels).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-auto-wo">Auto-WO plan</FieldLabel>
                <label className="flex h-9 items-center gap-2 rounded-md border border-input px-3 text-sm">
                  <input
                    id="pm-auto-wo"
                    type="checkbox"
                    checked={planForm.auto_generate_wo}
                    onChange={(event) => setPlanForm((current) => ({ ...current, auto_generate_wo: event.target.checked }))}
                    className="size-4 rounded border-input accent-primary"
                  />
                  {planForm.auto_generate_wo ? "Aktif" : "Nonaktif"}
                </label>
              </Field>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="pm-assignee">Default assignee</FieldLabel>
                <NativeSelect id="pm-assignee" value={planForm.default_assignee_id} onChange={(event) => setPlanForm((current) => ({ ...current, default_assignee_id: event.target.value }))}>
                  <NativeSelectOption value="">Belum di-assign</NativeSelectOption>
                  {assigneeOptions.map((item) => (
                    <NativeSelectOption key={item.id} value={String(item.id)}>{item.name} ({item.role})</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="pm-supervisor">Default supervisor</FieldLabel>
                <NativeSelect id="pm-supervisor" value={planForm.default_supervisor_id} onChange={(event) => setPlanForm((current) => ({ ...current, default_supervisor_id: event.target.value }))}>
                  <NativeSelectOption value="">Belum ada supervisor</NativeSelectOption>
                  {supervisorOptions.map((item) => (
                    <NativeSelectOption key={item.id} value={String(item.id)}>{item.name} ({item.role})</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
            </div>

            <div className="rounded-lg border border-border p-3">
              <div className="flex items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-semibold">Checklist template</p>
                  <p className="text-xs text-muted-foreground">Disalin sebagai snapshot saat WO PM dibuat.</p>
                </div>
                <Button
                  type="button"
                  size="xs"
                  variant="outline"
                  onClick={() => setPlanForm((current) => ({ ...current, checklist: [...current.checklist, { task: "", standard: "" }] }))}
                >
                  <Plus aria-hidden="true" /> Item
                </Button>
              </div>
              <div className="mt-3 flex flex-col gap-2">
                {planForm.checklist.map((item, index) => (
                  <div key={index} className="grid gap-2 sm:grid-cols-[1fr_1fr_auto]">
                    <Input
                      value={item.task}
                      onChange={(event) => setPlanForm((current) => ({
                        ...current,
                        checklist: current.checklist.map((row, rowIndex) => rowIndex === index ? { ...row, task: event.target.value } : row),
                      }))}
                      placeholder={`Task #${index + 1}`}
                      maxLength={500}
                    />
                    <Input
                      value={item.standard}
                      onChange={(event) => setPlanForm((current) => ({
                        ...current,
                        checklist: current.checklist.map((row, rowIndex) => rowIndex === index ? { ...row, standard: event.target.value } : row),
                      }))}
                      placeholder="Standard"
                      maxLength={500}
                    />
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="ghost"
                      onClick={() => setPlanForm((current) => ({
                        ...current,
                        checklist: current.checklist.length === 1
                          ? [{ task: "", standard: "" }]
                          : current.checklist.filter((_, rowIndex) => rowIndex !== index),
                      }))}
                      aria-label="Hapus item checklist"
                    >
                      <XCircle aria-hidden="true" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>

            {planError ? <FieldError>{planError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" disabled={planBusy}>
                {planBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                {editingPlan ? "Simpan perubahan" : "Simpan plan"}
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <Sheet open={materialPlan !== null} onOpenChange={(open) => { if (!open) setMaterialPlan(null); }}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Material plan</SheetTitle>
            <SheetDescription>
              {materialPlan ? `${materialPlan.name} · template kebutuhan sparepart untuk PM` : ""}
            </SheetDescription>
          </SheetHeader>
          <div className="flex flex-col gap-4 px-4 pb-4">
            <Alert>
              <PackageSearch aria-hidden="true" />
              <AlertTitle>Template saja</AlertTitle>
              <AlertDescription>
                Material plan belum melakukan reservasi atau cek kesiapan stok. Perhitungan readiness masuk M6.
              </AlertDescription>
            </Alert>
            {materialError ? <FieldError>{materialError}</FieldError> : null}
            {materialLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : materials.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon"><PackageSearch aria-hidden="true" /></EmptyMedia>
                  <EmptyTitle>Belum ada material</EmptyTitle>
                  <EmptyDescription>Tambahkan sparepart yang biasa dibutuhkan plan ini.</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Sparepart</TableHead>
                      <TableHead>Qty</TableHead>
                      <TableHead>Opsional</TableHead>
                      <TableHead>Catatan</TableHead>
                      <TableHead>Aksi</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {materials.map((material) => (
                      <TableRow key={material.id}>
                        <TableCell>
                          <p className="font-medium">{material.sparepart_code}</p>
                          <p className="text-xs text-muted-foreground">{material.sparepart_name}</p>
                        </TableCell>
                        <TableCell className="whitespace-nowrap">{material.qty} {material.uom}</TableCell>
                        <TableCell>{material.is_optional ? "Ya" : "Tidak"}</TableCell>
                        <TableCell className="max-w-48 truncate text-sm text-muted-foreground">{material.notes || "-"}</TableCell>
                        <TableCell>
                          <Button type="button" size="xs" variant="outline" onClick={() => void removeMaterial(material)}>
                            <Trash2 aria-hidden="true" /> Hapus
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}

            <form className="flex flex-col gap-3 rounded-lg border border-border p-3" onSubmit={addMaterial}>
              <p className="text-sm font-medium">Tambah material</p>
              <div className="grid gap-3 sm:grid-cols-[1fr_120px]">
                <Field>
                  <FieldLabel htmlFor="pm-material-sparepart">Sparepart</FieldLabel>
                  <NativeSelect id="pm-material-sparepart" value={materialSparepartID} onChange={(event) => setMaterialSparepartID(event.target.value)} required>
                    <NativeSelectOption value="">Pilih sparepart</NativeSelectOption>
                    {materialOptions.map((option) => (
                      <NativeSelectOption key={option.id} value={option.id}>
                        {option.code} - {option.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="pm-material-qty">Qty</FieldLabel>
                  <Input id="pm-material-qty" type="number" min={0.001} step="0.001" value={materialQty} onChange={(event) => setMaterialQty(event.target.value)} required />
                </Field>
              </div>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={materialOptional} onChange={(event) => setMaterialOptional(event.target.checked)} className="size-4 rounded border-border" />
                Material opsional
              </label>
              <Field>
                <FieldLabel htmlFor="pm-material-notes">Catatan</FieldLabel>
                <Input id="pm-material-notes" value={materialNotes} maxLength={500} onChange={(event) => setMaterialNotes(event.target.value)} placeholder="Opsional" />
              </Field>
              <SheetFooter className="px-0">
                <Button type="submit" disabled={materialBusy}>
                  {materialBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                  Tambah material
                </Button>
              </SheetFooter>
            </form>
          </div>
        </SheetContent>
      </Sheet>

      <Sheet open={actionKind !== null} onOpenChange={(open) => { if (!open) { setActionKind(null); setActionSchedule(null); } }}>
        <SheetContent className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>
              {actionKind === "override" ? "Override due" : actionKind === "skip" ? "Skip siklus" : "Batalkan schedule"}
            </SheetTitle>
            <SheetDescription>
              {actionSchedule ? `${actionSchedule.unit_code} · ${actionSchedule.plan_name} · siklus #${actionSchedule.cycle_no}` : ""}
            </SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitAction}>
            {actionKind === "override" ? (
              <>
                <div className="grid gap-3 sm:grid-cols-3">
                  <Field>
                    <FieldLabel htmlFor="pm-override-hm">Due HM</FieldLabel>
                    <Input id="pm-override-hm" type="number" min={0} step="0.1" value={actionDueHM} onChange={(event) => setActionDueHM(event.target.value)} />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="pm-override-km">Due KM</FieldLabel>
                    <Input id="pm-override-km" type="number" min={0} step="0.1" value={actionDueKM} onChange={(event) => setActionDueKM(event.target.value)} />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="pm-override-date">Due date</FieldLabel>
                    <Input id="pm-override-date" type="date" value={actionDueDate} onChange={(event) => setActionDueDate(event.target.value)} />
                  </Field>
                </div>
              </>
            ) : null}
            <Field>
              <FieldLabel htmlFor="pm-action-reason">Alasan</FieldLabel>
              <Textarea
                id="pm-action-reason"
                value={actionReason}
                onChange={(event) => setActionReason(event.target.value)}
                minLength={5}
                maxLength={500}
                required
                placeholder="Wajib minimal 5 karakter"
              />
            </Field>
            {actionError ? <FieldError>{actionError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" variant={actionKind === "cancel" ? "destructive" : "default"} disabled={actionBusy}>
                {actionBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : null}
                Simpan
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-lg border border-border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-xl font-semibold">{value}</p>
    </div>
  );
}
