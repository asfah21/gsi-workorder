"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { ApiError, proxyGet, proxyPatch, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  EngineModel,
  ListResponse,
  Unit,
  UnitModel,
} from "@/lib/types";

type FormState = {
  code: string;
  unit_type: string;
  merk: string;
  class_unit: string;
  license_plate: string;
  site_location: string;
  unit_model_id: string;
  engine_model_id: string;
  serial_number: string;
  engine_serial_number: string;
  is_breakdown: boolean;
  is_active: boolean;
};

export function UnitForm({ unit }: { unit?: Unit }) {
  const router = useRouter();
  const [form, setForm] = useState<FormState>(() => initialForm(unit));
  const [unitModels, setUnitModels] = useState<UnitModel[]>([]);
  const [engineModels, setEngineModels] = useState<EngineModel[]>([]);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const isEdit = Boolean(unit);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      proxyGet<ListResponse<UnitModel>>("/unit-models?limit=100"),
      proxyGet<ListResponse<EngineModel>>("/engine-models?limit=100"),
    ])
      .then(([units, engines]) => {
        if (!cancelled) {
          setUnitModels(units.data);
          setEngineModels(engines.data);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError("Master model tidak dapat dimuat.");
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const update = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((current) => ({ ...current, [key]: value }));
    setFieldErrors((current) => {
      if (!current[key]) {
        return current;
      }
      const next = { ...current };
      delete next[key];
      return next;
    });
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setFieldErrors({});
    setPending(true);

    const payload = {
      code: form.code.trim(),
      unit_type: form.unit_type.trim(),
      merk: form.merk.trim(),
      class_unit: form.class_unit.trim(),
      license_plate: form.license_plate.trim(),
      site_location: form.site_location.trim(),
      // PATCH uses 0 as the explicit "clear model" sentinel; POST uses null.
      unit_model_id: form.unit_model_id
        ? Number(form.unit_model_id)
        : unit
          ? 0
          : null,
      engine_model_id: form.engine_model_id
        ? Number(form.engine_model_id)
        : unit
          ? 0
          : null,
      serial_number: form.serial_number.trim(),
      engine_serial_number: form.engine_serial_number.trim(),
      is_breakdown: form.is_breakdown,
      is_active: form.is_active,
    };

    try {
      const result = unit
        ? await proxyPatch<DataResponse<Unit>>(`/units/${unit.id}`, payload)
        : await proxyPost<DataResponse<Unit>>("/units", payload);
      router.replace(`/units/${result.data.id}`);
      router.refresh();
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
      } else {
        setError("Unit tidak dapat disimpan.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <form className="space-y-6" onSubmit={(event) => void handleSubmit(event)} noValidate>
      {error ? (
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <fieldset className="grid gap-5 sm:grid-cols-2">
        <legend className="mb-4 text-base font-semibold">Identitas unit</legend>
        <Field label="Kode unit" htmlFor="code" required error={fieldErrors.code}>
          <input
            id="code"
            name="code"
            type="text"
            required
            autoComplete="off"
            spellCheck={false}
            value={form.code}
            onChange={(event) => update("code", event.target.value)}
            aria-invalid={Boolean(fieldErrors.code)}
            aria-describedby={fieldErrors.code ? "code-error" : undefined}
            className={inputClass}
          />
        </Field>
        <Field label="Tipe unit" htmlFor="unit_type" error={fieldErrors.unit_type}>
          <input
            id="unit_type"
            name="unit_type"
            type="text"
            autoComplete="off"
            value={form.unit_type}
            onChange={(event) => update("unit_type", event.target.value)}
            className={inputClass}
          />
        </Field>
        <Field label="Merk" htmlFor="merk" error={fieldErrors.merk}>
          <input
            id="merk"
            name="merk"
            type="text"
            value={form.merk}
            onChange={(event) => update("merk", event.target.value)}
            className={inputClass}
          />
        </Field>
        <Field label="Kelas unit" htmlFor="class_unit" error={fieldErrors.class_unit}>
          <input
            id="class_unit"
            name="class_unit"
            type="text"
            value={form.class_unit}
            onChange={(event) => update("class_unit", event.target.value)}
            className={inputClass}
          />
        </Field>
        <Field label="Nomor polisi" htmlFor="license_plate" error={fieldErrors.license_plate}>
          <input
            id="license_plate"
            name="license_plate"
            type="text"
            autoComplete="off"
            value={form.license_plate}
            onChange={(event) => update("license_plate", event.target.value)}
            className={inputClass}
          />
        </Field>
        <Field label="Lokasi" htmlFor="site_location" error={fieldErrors.site_location}>
          <input
            id="site_location"
            name="site_location"
            type="text"
            value={form.site_location}
            onChange={(event) => update("site_location", event.target.value)}
            className={inputClass}
          />
        </Field>
      </fieldset>

      <fieldset className="grid gap-5 sm:grid-cols-2">
        <legend className="mb-4 text-base font-semibold">Model</legend>
        <Field label="Unit model" htmlFor="unit_model_id" error={fieldErrors.unit_model_id}>
          <select
            id="unit_model_id"
            name="unit_model_id"
            value={form.unit_model_id}
            onChange={(event) => update("unit_model_id", event.target.value)}
            className={inputClass}
          >
            <option value="">Tidak dipilih</option>
            {unitModels.map((model) => (
              <option key={model.id} value={model.id}>
                {model.manufacturer} — {model.model}{model.is_active ? "" : " (nonaktif)"}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Engine model" htmlFor="engine_model_id" error={fieldErrors.engine_model_id}>
          <select
            id="engine_model_id"
            name="engine_model_id"
            value={form.engine_model_id}
            onChange={(event) => update("engine_model_id", event.target.value)}
            className={inputClass}
          >
            <option value="">Tidak dipilih</option>
            {engineModels.map((model) => (
              <option key={model.id} value={model.id}>
                {model.manufacturer} — {model.model}{model.is_active ? "" : " (nonaktif)"}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Serial number unit" htmlFor="serial_number" error={fieldErrors.serial_number}>
          <input
            id="serial_number"
            name="serial_number"
            type="text"
            value={form.serial_number}
            onChange={(event) => update("serial_number", event.target.value)}
            className={inputClass}
          />
        </Field>
        <Field label="Serial number engine" htmlFor="engine_serial_number" error={fieldErrors.engine_serial_number}>
          <input
            id="engine_serial_number"
            name="engine_serial_number"
            type="text"
            value={form.engine_serial_number}
            onChange={(event) => update("engine_serial_number", event.target.value)}
            className={inputClass}
          />
        </Field>
      </fieldset>

      <fieldset className="space-y-3">
        <legend className="mb-2 text-base font-semibold">Status</legend>
        <label className="flex min-h-11 items-center gap-3 text-sm">
          <input
            type="checkbox"
            checked={form.is_active}
            onChange={(event) => update("is_active", event.target.checked)}
            className="size-4 accent-primary"
          />
          Unit aktif
        </label>
        <label className="flex min-h-11 items-center gap-3 text-sm">
          <input
            type="checkbox"
            checked={form.is_breakdown}
            onChange={(event) => update("is_breakdown", event.target.checked)}
            className="size-4 accent-primary"
          />
          Tandai breakdown
        </label>
      </fieldset>

      <div className="flex flex-wrap gap-3">
        <button
          type="submit"
          disabled={pending}
          className="min-h-11 rounded-md bg-primary px-4 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          {pending ? "Menyimpan..." : isEdit ? "Simpan perubahan" : "Buat unit"}
        </button>
        <Link
          href={unit ? `/units/${unit.id}` : "/units"}
          className="inline-flex min-h-11 items-center rounded-md border border-border px-4 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          Batal
        </Link>
      </div>
    </form>
  );
}

const inputClass =
  "mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30";

function Field({
  label,
  htmlFor,
  required,
  error,
  children,
}: {
  label: string;
  htmlFor: string;
  required?: boolean;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label htmlFor={htmlFor} className="text-sm font-medium">
        {label}
        {required ? <span aria-hidden="true"> *</span> : null}
      </label>
      {children}
      {error ? (
        <p id={`${htmlFor}-error`} className="mt-1 text-sm text-destructive">
          {error}
        </p>
      ) : null}
    </div>
  );
}

function initialForm(unit?: Unit): FormState {
  return {
    code: unit?.code || "",
    unit_type: unit?.unit_type || "",
    merk: unit?.merk || "",
    class_unit: unit?.class_unit || "",
    license_plate: unit?.license_plate || "",
    site_location: unit?.site_location || "",
    unit_model_id: unit?.unit_model_id ? String(unit.unit_model_id) : "",
    engine_model_id: unit?.engine_model_id ? String(unit.engine_model_id) : "",
    serial_number: unit?.serial_number || "",
    engine_serial_number: unit?.engine_serial_number || "",
    is_breakdown: unit?.is_breakdown || false,
    is_active: unit?.is_active ?? true,
  };
}
