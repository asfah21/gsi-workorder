"use client";

import { CircleAlert, Loader2, Plus } from "lucide-react";
import { useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
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
import { Textarea } from "@/components/ui/textarea";
import { ApiError, proxyPost } from "@/lib/api-client";
import type { DataResponse, Unit, WOCandidate, WorkCategory } from "@/lib/types";

const categories: WorkCategory[] = [
  "Service",
  "Scheduled",
  "Unscheduled",
  "Incident",
  "UBOP",
];

const priorities = [
  { value: "low", label: "Low" },
  { value: "medium", label: "Medium" },
  { value: "high", label: "High" },
  { value: "critical", label: "Critical" },
] as const;

export function WorkRequestCreateSheet({
  open,
  units,
  unitsLoading,
  unitsError,
  onOpenChange,
  onSaved,
}: {
  open: boolean;
  units: Unit[];
  unitsLoading: boolean;
  unitsError: string;
  onOpenChange: (open: boolean) => void;
  onSaved: (message: string) => void;
}) {
  const [unitId, setUnitId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState<WorkCategory>("Service");
  const [priority, setPriority] = useState<(typeof priorities)[number]["value"]>("medium");
  const [isBreakdown, setIsBreakdown] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    const nextErrors: Record<string, string> = {};
    if (!unitId) nextErrors.unit_id = "Unit wajib dipilih";
    if (!title.trim()) nextErrors.title = "Judul wajib diisi";
    if (!category) nextErrors.work_category = "Kategori wajib dipilih";
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setPending(true);
    try {
      await proxyPost<DataResponse<WOCandidate>>("/wo-candidates", {
        unit_id: Number(unitId),
        title: title.trim(),
        description: description.trim(),
        work_category: category,
        is_breakdown: isBreakdown,
        suggested_priority: priority,
      });
      onSaved("Work Request berhasil dibuat.");
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
      } else {
        setError("Work Request tidak dapat dibuat.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Buat Work Request</SheetTitle>
          <SheetDescription>
            Ajukan pekerjaan non-P2H. Dispatch/planner akan memvalidasi sebelum konversi ke WO.
          </SheetDescription>
        </SheetHeader>

        <form className="grid gap-4 px-4" onSubmit={(event) => void submit(event)} noValidate>
          <Field data-invalid={Boolean(fieldErrors.unit_id)}>
            <FieldLabel htmlFor="wr-unit">
              Unit <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <NativeSelect
              id="wr-unit"
              value={unitId}
              onChange={(event) => setUnitId(event.target.value)}
              disabled={unitsLoading}
            >
              <NativeSelectOption value="">
                {unitsLoading ? "Memuat unit..." : "Pilih unit"}
              </NativeSelectOption>
              {units.map((unit) => (
                <NativeSelectOption key={unit.id} value={String(unit.id)}>
                  {unit.code} — {unit.unit_type || "unit"}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {fieldErrors.unit_id ? <FieldError>{fieldErrors.unit_id}</FieldError> : null}
            {unitsError ? <FieldError>{unitsError}</FieldError> : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.title)}>
            <FieldLabel htmlFor="wr-title">
              Judul <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="wr-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              maxLength={200}
              placeholder="Contoh: Kebocoran hydraulic hose"
            />
            {fieldErrors.title ? <FieldError>{fieldErrors.title}</FieldError> : null}
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.work_category)}>
              <FieldLabel htmlFor="wr-category">
                Kategori <span aria-hidden="true" className="text-destructive">*</span>
              </FieldLabel>
              <NativeSelect
                id="wr-category"
                value={category}
                onChange={(event) => setCategory(event.target.value as WorkCategory)}
              >
                {categories.map((value) => (
                  <NativeSelectOption key={value} value={value}>
                    {value}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
              {fieldErrors.work_category ? (
                <FieldError>{fieldErrors.work_category}</FieldError>
              ) : null}
            </Field>
            <Field>
              <FieldLabel htmlFor="wr-priority">Prioritas</FieldLabel>
              <NativeSelect
                id="wr-priority"
                value={priority}
                onChange={(event) =>
                  setPriority(event.target.value as (typeof priorities)[number]["value"])
                }
              >
                {priorities.map((item) => (
                  <NativeSelectOption key={item.value} value={item.value}>
                    {item.label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="wr-breakdown">Kondisi breakdown</FieldLabel>
            <NativeSelect
              id="wr-breakdown"
              value={isBreakdown ? "1" : "0"}
              onChange={(event) => setIsBreakdown(event.target.value === "1")}
            >
              <NativeSelectOption value="0">Bukan breakdown</NativeSelectOption>
              <NativeSelectOption value="1">Breakdown</NativeSelectOption>
            </NativeSelect>
            <FieldDescription>
              Backlog default hanya menampilkan Work Request non-breakdown yang belum dikonversi.
            </FieldDescription>
          </Field>

          <Field data-invalid={Boolean(fieldErrors.description)}>
            <FieldLabel htmlFor="wr-description">Deskripsi</FieldLabel>
            <Textarea
              id="wr-description"
              rows={4}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              maxLength={2000}
              placeholder="Gejala, lokasi, dan informasi pendukung..."
            />
            {fieldErrors.description ? <FieldError>{fieldErrors.description}</FieldError> : null}
          </Field>

          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Work Request tidak dapat dibuat</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <SheetFooter className="px-0 sm:flex-row sm:justify-end">
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={pending}
            >
              Batal
            </Button>
            <Button type="submit" size="sm" disabled={pending || unitsLoading || units.length === 0}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" /> Menyimpan...
                </>
              ) : (
                <>
                  <Plus aria-hidden="true" /> Buat Work Request
                </>
              )}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
