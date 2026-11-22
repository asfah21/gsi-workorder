"use client";

import { CircleAlert, Loader2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Field as FieldRoot,
  FieldError,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
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

type UnitFormProps = {
  unit?: Unit;
  idPrefix?: string;
  onSuccess?: (unit: Unit) => void;
  onCancel?: () => void;
  footer?: (state: { pending: boolean; isEdit: boolean }) => React.ReactNode;
};

export function UnitForm({
  unit,
  idPrefix = "unit",
  onSuccess,
  onCancel,
  footer,
}: UnitFormProps) {
  const router = useRouter();
  const [form, setForm] = useState<FormState>(() => initialForm(unit));
  const [unitModels, setUnitModels] = useState<UnitModel[]>([]);
  const [engineModels, setEngineModels] = useState<EngineModel[]>([]);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const isEdit = Boolean(unit);
  const fieldId = (key: string) => `${idPrefix}-${key}`;
  const errorId = (key: string) => `${fieldId(key)}-error`;

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

  const fieldState = (key: keyof FormState) => ({
    "aria-invalid": Boolean(fieldErrors[key]),
    "aria-describedby": fieldErrors[key] ? errorId(key) : undefined,
  });

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setFieldErrors({});

    if (!form.code.trim()) {
      setFieldErrors({ code: "Kode unit wajib diisi." });
      requestAnimationFrame(() => document.getElementById(fieldId("code"))?.focus());
      return;
    }

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
      if (onSuccess) {
        onSuccess(result.data);
      } else {
        router.replace(`/units/${result.data.id}`);
        router.refresh();
      }
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
        const firstField = Object.keys(submitError.fields)[0];
        requestAnimationFrame(() => {
          const target = firstField
            ? document.getElementById(fieldId(firstField))
            : document.getElementById(fieldId("form-error"));
          target?.focus();
        });
      } else {
        setError("Unit tidak dapat disimpan.");
        requestAnimationFrame(() => document.getElementById(fieldId("form-error"))?.focus());
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <form
      className="space-y-6"
      onSubmit={(event) => void handleSubmit(event)}
      aria-busy={pending}
      noValidate
    >
      {error ? (
        <Alert
          id={fieldId("form-error")}
          tabIndex={-1}
          variant="destructive"
          className="border-destructive/30 bg-destructive/5"
        >
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Unit tidak dapat disimpan</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <FieldSet className="gap-4">
        <FieldLegend className="text-base font-medium">Identitas unit</FieldLegend>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Kode unit" htmlFor={fieldId("code")} required error={fieldErrors.code}>
            <Input
              id={fieldId("code")}
              name="code"
              type="text"
              required
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              value={form.code}
              onChange={(event) => update("code", event.target.value)}
              {...fieldState("code")}
            />
          </Field>
          <Field label="Tipe unit" htmlFor={fieldId("unit_type")} error={fieldErrors.unit_type}>
            <Input
              id={fieldId("unit_type")}
              name="unit_type"
              type="text"
              autoComplete="off"
              value={form.unit_type}
              onChange={(event) => update("unit_type", event.target.value)}
              {...fieldState("unit_type")}
            />
          </Field>
          <Field label="Merk" htmlFor={fieldId("merk")} error={fieldErrors.merk}>
            <Input
              id={fieldId("merk")}
              name="merk"
              type="text"
              autoComplete="off"
              value={form.merk}
              onChange={(event) => update("merk", event.target.value)}
              {...fieldState("merk")}
            />
          </Field>
          <Field label="Kelas unit" htmlFor={fieldId("class_unit")} error={fieldErrors.class_unit}>
            <Input
              id={fieldId("class_unit")}
              name="class_unit"
              type="text"
              value={form.class_unit}
              onChange={(event) => update("class_unit", event.target.value)}
              {...fieldState("class_unit")}
            />
          </Field>
          <Field label="Nomor polisi" htmlFor={fieldId("license_plate")} error={fieldErrors.license_plate}>
            <Input
              id={fieldId("license_plate")}
              name="license_plate"
              type="text"
              autoComplete="off"
              autoCapitalize="characters"
              value={form.license_plate}
              onChange={(event) => update("license_plate", event.target.value)}
              {...fieldState("license_plate")}
            />
          </Field>
          <Field label="Lokasi" htmlFor={fieldId("site_location")} error={fieldErrors.site_location}>
            <Input
              id={fieldId("site_location")}
              name="site_location"
              type="text"
              autoComplete="off"
              value={form.site_location}
              onChange={(event) => update("site_location", event.target.value)}
              {...fieldState("site_location")}
            />
          </Field>
        </div>
      </FieldSet>

      <FieldSet className="gap-4">
        <FieldLegend className="text-base font-medium">Model dan serial</FieldLegend>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Unit model" htmlFor={fieldId("unit_model_id")} error={fieldErrors.unit_model_id}>
            <NativeSelect
              id={fieldId("unit_model_id")}
              name="unit_model_id"
              value={form.unit_model_id}
              onChange={(event) => update("unit_model_id", event.target.value)}
              className="w-full"
              {...fieldState("unit_model_id")}
            >
              <NativeSelectOption value="">Tidak dipilih</NativeSelectOption>
              {unitModels.map((model) => (
                <NativeSelectOption key={model.id} value={model.id}>
                  {model.manufacturer} — {model.model}
                  {model.is_active ? "" : " (nonaktif)"}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field label="Engine model" htmlFor={fieldId("engine_model_id")} error={fieldErrors.engine_model_id}>
            <NativeSelect
              id={fieldId("engine_model_id")}
              name="engine_model_id"
              value={form.engine_model_id}
              onChange={(event) => update("engine_model_id", event.target.value)}
              className="w-full"
              {...fieldState("engine_model_id")}
            >
              <NativeSelectOption value="">Tidak dipilih</NativeSelectOption>
              {engineModels.map((model) => (
                <NativeSelectOption key={model.id} value={model.id}>
                  {model.manufacturer} — {model.model}
                  {model.is_active ? "" : " (nonaktif)"}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field label="Serial number unit" htmlFor={fieldId("serial_number")} error={fieldErrors.serial_number}>
            <Input
              id={fieldId("serial_number")}
              name="serial_number"
              type="text"
              autoComplete="off"
              autoCapitalize="characters"
              spellCheck={false}
              value={form.serial_number}
              onChange={(event) => update("serial_number", event.target.value)}
              {...fieldState("serial_number")}
            />
          </Field>
          <Field
            label="Serial number engine"
            htmlFor={fieldId("engine_serial_number")}
            error={fieldErrors.engine_serial_number}
          >
            <Input
              id={fieldId("engine_serial_number")}
              name="engine_serial_number"
              type="text"
              autoComplete="off"
              autoCapitalize="characters"
              spellCheck={false}
              value={form.engine_serial_number}
              onChange={(event) => update("engine_serial_number", event.target.value)}
              {...fieldState("engine_serial_number")}
            />
          </Field>
        </div>
      </FieldSet>

      <FieldSet className="gap-3">
        <FieldLegend className="text-base font-medium">Status</FieldLegend>
        <label
          htmlFor={fieldId("is_active")}
          className="flex h-9 w-fit items-center gap-2 text-sm"
        >
          <input
            id={fieldId("is_active")}
            type="checkbox"
            name="is_active"
            checked={form.is_active}
            onChange={(event) => update("is_active", event.target.checked)}
            className="size-4 accent-primary"
          />
          Unit aktif
        </label>
        <label
          htmlFor={fieldId("is_breakdown")}
          className="flex h-9 w-fit items-center gap-2 text-sm"
        >
          <input
            id={fieldId("is_breakdown")}
            type="checkbox"
            name="is_breakdown"
            checked={form.is_breakdown}
            onChange={(event) => update("is_breakdown", event.target.checked)}
            className="size-4 accent-primary"
          />
          Tandai breakdown
        </label>
      </FieldSet>

      {footer ? (
        footer({ pending, isEdit })
      ) : (
        <div className="flex flex-wrap gap-3">
          <Button type="submit" disabled={pending}>
            {pending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                Menyimpan...
              </>
            ) : isEdit ? (
              "Simpan perubahan"
            ) : (
              "Buat unit"
            )}
          </Button>
          {onCancel ? (
            <Button type="button" variant="outline" onClick={onCancel}>
              Batal
            </Button>
          ) : (
            <Link
              href={unit ? `/units/${unit.id}` : "/units"}
              className={buttonVariants({ variant: "outline" })}
            >
              Batal
            </Link>
          )}
        </div>
      )}
    </form>
  );
}

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
    <FieldRoot data-invalid={Boolean(error)}>
      <FieldLabel htmlFor={htmlFor}>
        {label}
        {required ? (
          <span aria-hidden="true" className="text-destructive">
            *
          </span>
        ) : null}
      </FieldLabel>
      {children}
      {error ? (
        <FieldError id={`${htmlFor}-error`}>{error}</FieldError>
      ) : null}
    </FieldRoot>
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
