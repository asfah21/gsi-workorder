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

export type UnitImportRowResult = {
  row: number;
  code: string;
  status: "created" | "skipped" | "failed";
  message?: string;
};

export type UnitImportResult = {
  total: number;
  created: number;
  skipped: number;
  failed: number;
  rows: UnitImportRowResult[];
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

export type ComponentOption = {
  id: number;
  code: string;
  name: string;
  category: string;
  component_class: string;
  default_lifetime_hm?: number;
  default_lifetime_days?: number;
  default_lifetime_km?: number;
  is_active: boolean;
};

export type ComponentInstallation = {
  id: number;
  unit_id: number;
  unit_code: string;
  component_id: number;
  component_code: string;
  component_name: string;
  component_class: string;
  serial_number: string;
  installed_at: string;
  installed_hm: number;
  installed_km: number;
  removed_at?: string;
  removed_hm?: number;
  removed_km?: number;
  lifetime_hm?: number;
  lifetime_days?: number;
  lifetime_km?: number;
  status: "active" | "removed" | "transferred" | "scrapped";
  source_wo_id?: number;
  notes: string;
  created_by?: number;
  created_at: string;
  updated_at: string;
};

export type ComponentLifetime = {
  installation_id: number;
  unit_id: number;
  unit_code: string;
  unit_current_hm: number;
  unit_current_km: number;
  component_id: number;
  component_code: string;
  component_name: string;
  category: string;
  component_class: string;
  serial_number: string;
  installed_at: string;
  installed_hm: number;
  installed_km: number;
  removed_at?: string;
  removed_hm?: number;
  removed_km?: number;
  status: "active" | "removed" | "transferred" | "scrapped";
  lifetime_hm?: number;
  lifetime_days?: number;
  lifetime_km?: number;
  used_hm: number;
  used_km: number;
  used_days: number;
  remaining_hm?: number;
  remaining_days?: number;
  remaining_km?: number;
  lifetime_state: "inactive" | "unset" | "safe" | "upcoming" | "due" | "overdue";
};

export type UnitMaintenanceHistory = {
  wo_id: number;
  unit_id: number;
  unit_code: string;
  wo_code: string;
  source: string;
  maintenance_type: WorkOrderMaintenanceType;
  status: string;
  priority: WorkOrderPriority | string;
  title: string;
  description: string;
  p2h_report_id?: number;
  candidate_id?: number;
  pm_schedule_id?: number;
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
  elapsed_minutes?: number;
  checklist_total: number;
  checklist_done: number;
  checklist_not_ok: number;
  findings_count: number;
  usage_count: number;
  usage_qty: number;
  material_status: string;
};

export type UnitComponentHistory = {
  unit_id: number;
  unit_code: string;
  wo_id: number;
  wo_code: string;
  wo_title: string;
  maintenance_type: WorkOrderMaintenanceType;
  wo_status: string;
  closed_at?: string;
  component_id?: number;
  component_code: string;
  component_name: string;
  is_fallback: boolean;
  fallback_sparepart_id?: number;
  sparepart_codes: string;
  sparepart_names: string;
  sparepart_count: number;
  qty: number;
  used_at: string;
  hm?: number;
  km?: number;
  replacement_no: number;
  previous_used_at?: string;
  days_since_previous?: number;
  hm_since_previous?: number;
  km_since_previous?: number;
};

export type UnitComponentInterval = {
  unit_id: number;
  unit_code: string;
  component_id?: number;
  fallback_sparepart_id?: number;
  component_code: string;
  component_name: string;
  is_fallback: boolean;
  replacement_count: number;
  first_replaced_at: string;
  last_replaced_at: string;
  avg_interval_days?: number;
  avg_interval_hm?: number;
  avg_interval_km?: number;
  total_qty: number;
  last_hm?: number;
  last_km?: number;
};

export type UnitDowntimeEvent = {
  wo_id: number;
  unit_id: number;
  unit_code: string;
  wo_code: string;
  title: string;
  maintenance_type: WorkOrderMaintenanceType;
  priority: string;
  assignee_id?: number;
  assignee_name: string;
  supervisor_id?: number;
  supervisor_name: string;
  started_at?: string;
  downtime_start?: string;
  downtime_end: string;
  downtime_class: "planned" | "unplanned";
  downtime_minutes?: number;
  waiting_minutes: number;
};

export type UnitDowntimeSummary = {
  total_minutes: number;
  planned_minutes: number;
  unplanned_minutes: number;
  waiting_minutes: number;
  mttr_minutes: number;
  events: number;
  planned_events: number;
  unplanned_events: number;
};

export type UnitDowntimeResponse = {
  summary: UnitDowntimeSummary;
  events: UnitDowntimeEvent[];
};

export type DashboardSummary = {
  units_total: number;
  units_active: number;
  units_breakdown: number;
  unit_models: number;
  engine_models: number;
  readings_last_30_days: number;
  pm_overdue?: number;
  pm_due?: number;
  pm_due_soon?: number;
  material_shortage?: number;
  stock_low?: number;
  stock_critical?: number;
  waiting_part_units?: number;
  wo_open?: number;
  wo_in_progress?: number;
  breakdown_open?: number;
  downtime_minutes_30d?: number;
  mttr_minutes_30d?: number;
};

export type ReportColumn = {
  key: string;
  label: string;
};

export type ReportTable = {
  columns: ReportColumn[];
  rows: Array<Record<string, unknown>>;
  meta: { page: number; limit: number; total: number };
};

export type DashboardPM = {
  summary: Record<string, number>;
  due: ReportTable;
};

export type DashboardMaterial = {
  summary: Record<string, number>;
  requirements: ReportTable;
  aging: ReportTable;
};

export type DashboardInventory = {
  summary: Record<string, number>;
  low_stock: ReportTable;
  movements: ReportTable;
};

export type DashboardMaintenance = {
  summary: Record<string, number>;
  activity: ReportTable;
  downtime: ReportTable;
  waiting: ReportTable;
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

export type SparepartNumberType = "oem" | "aftermarket" | "superseded";
export type CompatibilityStatus =
  | "pending_verification"
  | "approved_compatible"
  | "conditional_compatible"
  | "not_compatible";

export type SparepartNumber = {
  id: number;
  sparepart_id: number;
  part_number: string;
  brand: string;
  number_type: SparepartNumberType;
  is_primary: boolean;
  notes: string;
  created_at: string;
};

export type SparepartAlternative = {
  id: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  alternative_sparepart_id: number;
  alternative_sparepart_code: string;
  alternative_sparepart_name: string;
  relation_type: "equivalent" | "superseded" | "substitute";
  verification_status: CompatibilityStatus;
  notes: string;
  verified_by?: number;
  verified_by_name: string;
  verified_at?: string;
  direction: "outgoing" | "incoming";
  created_at: string;
  updated_at: string;
};

export type SparepartCompatibility = {
  id: number;
  sparepart_id: number;
  unit_model_id?: number;
  unit_model_name: string;
  engine_model_id?: number;
  engine_model_name: string;
  configuration: string;
  status: CompatibilityStatus;
  condition_note: string;
  source: string;
  notes: string;
  verified_by?: number;
  verified_by_name: string;
  verified_at?: string;
  created_at: string;
  updated_at: string;
};

export type Sparepart = {
  id: number;
  code: string;
  name: string;
  category: string;
  uom: string;
  description: string;
  is_active: boolean;
  primary_part_number: string;
  compatibility_count: number;
  approved_compatibility_count: number;
  numbers: SparepartNumber[];
  alternatives: SparepartAlternative[];
  compatibility: SparepartCompatibility[];
  created_at: string;
  updated_at: string;
};

export type PMPlanMaterial = {
  id: number;
  pm_plan_id: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  qty: number;
  is_optional: boolean;
  notes: string;
};

export type Warehouse = {
  id: number;
  code: string;
  name: string;
  site_location: string;
  is_active: boolean;
  location_count: number;
  created_at: string;
  updated_at: string;
};

export type WarehouseLocation = {
  id: number;
  warehouse_id: number;
  warehouse_code: string;
  warehouse_name: string;
  code: string;
  name: string;
  location_type: "rack" | "bin" | "yard" | "staging";
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type StockPolicy = {
  id: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  warehouse_location_id: number;
  warehouse_id: number;
  warehouse_code: string;
  location_code: string;
  location_name: string;
  min_qty: number;
  critical_qty: number;
  reorder_qty?: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type InventoryBalance = {
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  category: string;
  uom: string;
  sparepart_active: boolean;
  warehouse_location_id: number;
  warehouse_id: number;
  warehouse_code: string;
  warehouse_name: string;
  location_code: string;
  location_name: string;
  location_type: string;
  on_hand: number;
  reserved: number;
  available: number;
  policy_id?: number;
  min_qty: number;
  critical_qty: number;
  reorder_qty?: number;
  stock_status: "no_policy" | "ok" | "low" | "critical";
  updated_at?: string;
};

export type InventoryTransactionType =
  | "receipt"
  | "issue"
  | "return"
  | "transfer"
  | "adjustment"
  | "opname";

export type InventoryTransactionItem = {
  id: number;
  transaction_id: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  from_location_id?: number;
  from_location: string;
  to_location_id?: number;
  to_location: string;
  qty: number;
  notes: string;
};

export type InventoryTransaction = {
  id: number;
  code: string;
  type: InventoryTransactionType;
  status: "posted" | "reversed";
  reference_type: string;
  reference_id?: number;
  reference_no: string;
  reversal_of_id?: number;
  idempotency_key: string;
  posted_by?: number;
  posted_by_name: string;
  posted_at: string;
  notes: string;
  item_count: number;
  items: InventoryTransactionItem[];
  created_at: string;
};

export type InventoryMovement = {
  id: number;
  transaction_id: number;
  transaction_code: string;
  type: InventoryTransactionType;
  status: "posted" | "reversed";
  reference_no: string;
  posted_at: string;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  from_location_id?: number;
  from_location: string;
  to_location_id?: number;
  to_location: string;
  qty: number;
  notes: string;
};

export type OpnameStatus = "draft" | "counting" | "review" | "posted" | "cancelled";

export type OpnameItem = {
  id: number;
  session_id: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  location_id: number;
  warehouse_code: string;
  location_code: string;
  location_name: string;
  system_qty: number;
  counted_qty?: number;
  difference?: number;
  notes: string;
};

export type OpnameSession = {
  id: number;
  code: string;
  warehouse_id: number;
  warehouse_code: string;
  warehouse_name: string;
  status: OpnameStatus;
  started_at?: string;
  posted_at?: string;
  posted_by?: number;
  posted_by_name: string;
  notes: string;
  cancel_reason: string;
  cancelled_at?: string;
  created_by?: number;
  item_count: number;
  counted_count: number;
  items: OpnameItem[];
  created_at: string;
  updated_at: string;
};

export type InventoryReconciliation = {
  rows: {
    sparepart_id: number;
    sparepart_code: string;
    sparepart_name: string;
    warehouse_location_id: number;
    warehouse_code: string;
    location_code: string;
    balance_on_hand: number;
    ledger_on_hand: number;
    difference: number;
    balance_reserved: number;
    reservation_reserved: number;
    reserved_difference: number;
  }[];
  mismatch: number;
  checked_at: string;
};

export type MaterialRequirementStatus =
  | "draft"
  | "planned"
  | "checking_stock"
  | "ready"
  | "partially_ready"
  | "shortage"
  | "reserved"
  | "issued"
  | "completed"
  | "cancelled";

export type MaterialRequirementItem = {
  id: number;
  material_requirement_id: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  qty_required: number;
  qty_reserved: number;
  qty_issued: number;
  qty_returned: number;
  net_issued: number;
  available: number;
  coverage: number;
  shortage: number;
  is_optional: boolean;
  preferred_warehouse_id?: number;
  preferred_warehouse_code: string;
  notes: string;
};

export type MaterialUsage = {
  id: number;
  wo_id: number;
  material_requirement_item_id?: number;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  component_id?: number;
  component_name: string;
  warehouse_location_id: number;
  warehouse_code: string;
  location_code: string;
  qty: number;
  hm?: number;
  km?: number;
  movement_type: "issue" | "return";
  inventory_transaction_item_id: number;
  return_of_usage_id?: number;
  used_by?: number;
  used_by_name: string;
  used_at: string;
  notes: string;
};

export type MaterialRequirement = {
  id: number;
  code: string;
  wo_id: number;
  wo_code: string;
  wo_title: string;
  unit_code: string;
  source: string;
  status: MaterialRequirementStatus;
  status_changed_at: string;
  shortage_since?: string;
  requested_by?: number;
  requested_by_name: string;
  requested_at?: string;
  notes: string;
  item_count: number;
  required_count: number;
  shortage_count: number;
  derived_status: MaterialRequirementStatus;
  items: MaterialRequirementItem[];
  usages: MaterialUsage[];
  created_at: string;
  updated_at: string;
};

export type MaterialRequirementUpdate = {
  id: number;
  material_requirement_id: number;
  user_id?: number;
  user_name: string;
  status_from: string;
  status_to: string;
  note: string;
  created_at: string;
};

export type InventoryReservation = {
  id: number;
  material_requirement_item_id: number;
  material_requirement_id: number;
  requirement_code: string;
  wo_id: number;
  wo_code: string;
  sparepart_id: number;
  sparepart_code: string;
  sparepart_name: string;
  uom: string;
  warehouse_location_id: number;
  warehouse_code: string;
  location_code: string;
  location_name: string;
  qty: number;
  status: "active" | "consumed" | "released" | "cancelled";
  reserved_by?: number;
  reserved_by_name: string;
  reserved_at: string;
  released_at?: string;
  consumed_at?: string;
  notes: string;
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
