export type Role = "admin" | "planner" | "mechanic" | "storekeeper" | "viewer";

export type User = {
  id: number;
  nik: string;
  name: string;
  email: string;
  role: Role;
  is_active: boolean;
};

export type UnitModel = {
  id: number;
  manufacturer: string;
  model: string;
  category: string;
  notes: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type EngineModel = {
  id: number;
  manufacturer: string;
  model: string;
  fuel_type: string;
  displacement_cc: number | null;
  notes: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type ModelSummary = {
  id: number;
  manufacturer: string;
  model: string;
};

export type Unit = {
  id: number;
  odoo_unit_id?: number;
  code: string;
  unit_type: string;
  merk: string;
  class_unit: string;
  license_plate: string;
  site_location: string;
  unit_model_id?: number;
  engine_model_id?: number;
  unit_model?: ModelSummary;
  engine_model?: ModelSummary;
  serial_number: string;
  engine_serial_number: string;
  current_hm: number;
  current_km: number;
  hm_updated_at?: string;
  km_updated_at?: string;
  is_breakdown: boolean;
  is_active: boolean;
  synced_at?: string;
};

export type MeterReading = {
  id: number;
  unit_id: number;
  reading_type: "hm" | "km";
  value: number;
  recorded_at: string;
  source_type: "p2h" | "wo" | "manual" | "sync";
  source_id?: number;
  notes: string;
  created_at: string;
};

export type DashboardSummary = {
  units_total: number;
  units_active: number;
  units_breakdown: number;
  unit_models: number;
  engine_models: number;
  readings_last_30_days: number;
};

export type P2HFinding = {
  id: number;
  odoo_line_id: number;
  question_id?: number;
  component: string;
  category: string;
  condition: "rusak" | "tidak_ada";
  remark: string;
};

export type P2HReport = {
  id: number;
  odoo_report_id: number;
  name: string;
  report_date: string;
  unit_id?: number;
  unit_code: string;
  unit_type: string;
  shift: string;
  foreman_name: string;
  employee_name: string;
  location: string;
  state: string;
  approval_status: string;
  km_akhir?: number;
  hm_awal?: number;
  notes: string;
  checked_items: number;
  findings_count: number;
  rusak_count: number;
  detail_synced_at?: string;
  synced_at?: string;
  candidate_id?: number;
  candidate_status: string;
  findings: P2HFinding[];
};

export type WOCandidate = {
  id: number;
  p2h_report_id: number;
  unit_id?: number;
  unit_code: string;
  unit_type: string;
  report_name: string;
  report_date: string;
  suggested_priority: "low" | "medium" | "high" | "critical";
  status: "pending" | "approved" | "skipped" | "auto_created" | "archived";
  findings_count: number;
  rusak_count: number;
  decided_by?: number;
  decided_by_name: string;
  decided_at?: string;
  skip_reason: string;
  decision_note: string;
  wo_id?: number;
  wo_code: string;
  notified_at?: string;
  created_at: string;
};

export type UserOption = {
  id: number;
  nik: string;
  name: string;
  role: Role;
  is_active: boolean;
};

export type WorkOrderPriority = "low" | "medium" | "high" | "critical";
export type WorkOrderMaintenanceType =
  | "preventive"
  | "corrective"
  | "breakdown"
  | "inspection";

export type WorkOrderRecord = {
  id: number;
  code: string;
  unit_id?: number;
  unit_code: string;
  source: string;
  p2h_report_id?: number;
  candidate_id?: number;
  pm_schedule_id?: number;
  title: string;
  description: string;
  priority: WorkOrderPriority;
  status: string;
  maintenance_type: WorkOrderMaintenanceType;
  assignee_id?: number;
  assignee_name: string;
  supervisor_id?: number;
  supervisor_name: string;
  created_by?: number;
  created_by_name: string;
  due_date?: string;
  started_at?: string;
  closed_at?: string;
  created_at: string;
  updated_at: string;
};

export type WorkOrderFinding = {
  id: number;
  odoo_line_id: number;
  question_id?: number;
  component: string;
  category: string;
  condition: string;
  remark: string;
};

export type WorkOrderUpdate = {
  id: number;
  wo_id: number;
  user_id?: number;
  user_name: string;
  note: string;
  status_from: string;
  status_to: string;
  created_at: string;
};

export type WorkOrderAttachment = {
  id: number;
  wo_id: number;
  file_path: string;
  mime: string;
  size_bytes: number;
  uploaded_by?: number;
  created_at: string;
};

export type WorkOrderChecklistItem = {
  id: number;
  wo_id: number;
  sequence: number;
  component_id?: number;
  component: string;
  task: string;
  standard: string;
  result: "pending" | "ok" | "not_ok" | "na";
  checked_by?: number;
  checked_at?: string;
};

export type WorkOrderDetail = WorkOrderRecord & {
  findings: WorkOrderFinding[];
  updates: WorkOrderUpdate[];
  attachments: WorkOrderAttachment[];
  checklist: WorkOrderChecklistItem[];
};

export type CandidateWorkOrder = {
  id: number;
  code: string;
  title: string;
  description: string;
  priority: "low" | "medium" | "high" | "critical";
  status: string;
  assignee_id?: number;
  assignee_name: string;
  created_at: string;
};

export type CandidateDetail = {
  candidate: WOCandidate;
  report: P2HReport;
  work_order: CandidateWorkOrder | null;
};

export type CandidateApproveResponse = {
  candidate: WOCandidate;
  work_order: WorkOrderRecord;
};

export type CandidateBulkResponse = {
  approved: CandidateApproveResponse[];
  skipped: WOCandidate[];
};

export type NotificationItem = {
  id: number;
  type: string;
  title: string;
  body: string;
  ref_type: string;
  ref_id?: number;
  read_at?: string;
  created_at: string;
};

export type NotificationListResponse = {
  data: NotificationItem[];
  meta: {
    page: number;
    limit: number;
    total: number;
    unread: number;
  };
};

export type AppSettings = {
  p2h_auto_wo: boolean;
  candidate_sla_hours: number;
  candidate_backfill_days: number;
  pm_auto_wo: boolean;
  pm_upcoming_days: number;
  pm_due_soon_days: number;
  updated_at: string;
  history: SettingHistory[];
};

export type PMPlanStatus = "draft" | "active" | "inactive";
export type PMIntervalRule = "first_reached" | "all_required";

export type PMChecklistItem = {
  id: number;
  pm_plan_id: number;
  sequence: number;
  component_id?: number;
  component: string;
  task: string;
  standard: string;
  created_at: string;
  updated_at: string;
};

export type PMPlan = {
  id: number;
  name: string;
  description: string;
  unit_id?: number;
  unit_code: string;
  unit_model_id?: number;
  unit_model_name: string;
  status: PMPlanStatus;
  effective_from: string;
  effective_to?: string;
  interval_hm?: number;
  interval_km?: number;
  interval_days?: number;
  interval_rule: PMIntervalRule;
  upcoming_days?: number;
  due_soon_days?: number;
  upcoming_hm?: number;
  upcoming_km?: number;
  due_soon_hm?: number;
  due_soon_km?: number;
  auto_generate_wo: boolean;
  default_priority: WorkOrderPriority;
  default_assignee_id?: number;
  default_assignee_name: string;
  default_supervisor_id?: number;
  default_supervisor_name: string;
  created_by?: number;
  created_by_name: string;
  checklist: PMChecklistItem[];
  created_at: string;
  updated_at: string;
};

export type PMSchedule = {
  id: number;
  pm_plan_id: number;
  plan_name: string;
  unit_id: number;
  unit_code: string;
  unit_model_name: string;
  cycle_no: number;
  status: string;
  bucket: string;
  days_to_due?: number;
  due_hm?: number;
  due_km?: number;
  due_date?: string;
  last_completed_hm?: number;
  last_completed_km?: number;
  last_completed_at?: string;
  override_reason: string;
  overridden_at?: string;
  skip_reason: string;
  skipped_at?: string;
  cancel_reason: string;
  cancelled_at?: string;
  work_order_id?: number;
  work_order_code: string;
  plan?: PMPlan;
  created_at: string;
  updated_at: string;
};

export type PMComplianceRow = {
  pm_plan_id: number;
  plan_name: string;
  target: string;
  total: number;
  completed: number;
  on_time: number;
  overdue: number;
  missed: number;
  completion_rate: number;
  on_time_rate: number;
};

export type PMCompliance = {
  rows: PMComplianceRow[];
  total: number;
  completed: number;
  on_time: number;
  overdue: number;
  missed: number;
  completion_rate: number;
  on_time_rate: number;
};

export type PMGenerateResult = {
  schedule: PMSchedule;
  work_order: WorkOrderRecord;
};

export type SettingHistory = {
  id: number;
  key: string;
  old_value: string;
  new_value: string;
  changed_by?: number;
  changed_at: string;
};

export type ListResponse<T> = {
  data: T[];
  meta: {
    page: number;
    limit: number;
    total: number;
  };
};

export type DataResponse<T> = {
  data: T;
};

export type ApiErrorBody = {
  error?: {
    code?: string;
    message?: string;
    fields?: Record<string, string>;
  };
};
