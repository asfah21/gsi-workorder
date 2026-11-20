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
