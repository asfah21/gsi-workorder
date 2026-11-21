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

export type WorkOrderRecord = {
  id: number;
  code: string;
  unit_id?: number;
  source: string;
  p2h_report_id?: number;
  candidate_id?: number;
  title: string;
  description: string;
  priority: "low" | "medium" | "high" | "critical";
  status: string;
  assignee_id?: number;
  created_by?: number;
  created_at: string;
  updated_at: string;
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
  updated_at: string;
  history: SettingHistory[];
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
