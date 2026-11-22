"use client";

import {
  Activity,
  ArrowLeft,
  CalendarClock,
  CircleAlert,
  CircleCheck,
  ClipboardList,
  Gauge,
  History,
  LayoutDashboard,
  Loader2,
  MapPin,
  PackageSearch,
  Pencil,
  Repeat2,
  TimerOff,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { PageHeader } from "@/components/layout/page-header";
import { UnitStatusBadge } from "@/components/status-badges";
import { UnitComponentPanel } from "@/components/unit-component-panel";
import { UnitDowntimePanel } from "@/components/unit-downtime-panel";
import { UnitIntervalPanel } from "@/components/unit-interval-panel";
import { UnitTimelinePanel } from "@/components/unit-timeline-panel";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type { DataResponse, MeterReading, Unit } from "@/lib/types";
import {
  formatDateTimeWita,
  formatNumber,
  toDatetimeLocalWita,
  witaLocalToISOString,
} from "@/lib/time";
import { cn } from "@/lib/utils";

const sourceLabels: Record<MeterReading["source_type"], string> = {
  p2h: "P2H",
  wo: "WO",
  manual: "Manual",
  sync: "Sync",
};

type UnitTab = "overview" | "timeline" | "components" | "meters" | "downtime" | "intervals";

const unitTabs: { id: UnitTab; label: string; icon: LucideIcon }[] = [
  { id: "overview", label: "Ringkasan", icon: LayoutDashboard },
  { id: "timeline", label: "Timeline", icon: History },
  { id: "components", label: "Sparepart", icon: PackageSearch },
  { id: "meters", label: "HM/KM", icon: Gauge },
  { id: "downtime", label: "Downtime", icon: TimerOff },
  { id: "intervals", label: "Interval", icon: Repeat2 },
];

export function UnitDetailView({ unitId }: { unitId: number }) {
  const { user } = useSession();
  const [unit, setUnit] = useState<Unit | null>(null);
  const [readings, setReadings] = useState<MeterReading[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [tab, setTab] = useState<UnitTab>("overview");

  const canWrite = user.role === "admin" || user.role === "planner";
  const canReadMeter = ["admin", "planner", "mechanic"].includes(user.role);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [unitResult, readingResult] = await Promise.all([
        proxyGet<DataResponse<Unit>>(`/units/${unitId}`),
        proxyGet<{ data: MeterReading[] }>(`/units/${unitId}/meter-readings?limit=50`),
      ]);
      setUnit(unitResult.data);
      setReadings(readingResult.data);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Detail unit tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [unitId]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading && !unit) {
    return <DetailSkeleton />;
  }

  if (error && !unit) {
    return (
      <div className="space-y-4">
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Detail unit tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
        <Link
          href="/units"
          className="inline-flex h-9 items-center gap-2 text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          Kembali ke daftar unit
        </Link>
      </div>
    );
  }

  if (!unit) {
    return null;
  }

  const subtitle =
    [unit.merk, unit.unit_type, unit.site_location].filter(Boolean).join(" · ") ||
    "Identitas unit";

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={
          <span className="flex flex-wrap items-center gap-2">
            {unit.code}
            <UnitStatusBadge active={unit.is_active} breakdown={unit.is_breakdown} />
          </span>
        }
        description={subtitle}
        actions={
          canWrite ? (
            <Link
              href={`/units/${unit.id}/edit`}
              className={buttonVariants({ variant: "outline" })}
            >
              <Pencil className="size-4" aria-hidden="true" />
              Edit unit
            </Link>
          ) : null
        }
      />

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Data terbaru tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-3">
        <MetricCard
          label="HM terakhir"
          value={unit.hm_updated_at ? formatNumber(unit.current_hm) : "—"}
          hint={
            unit.hm_updated_at
              ? `Diperbarui ${formatDateTimeWita(unit.hm_updated_at)}`
              : "Belum ada reading"
          }
          icon={Gauge}
          tone="sky"
        />
        <MetricCard
          label="KM terakhir"
          value={unit.km_updated_at ? formatNumber(unit.current_km) : "—"}
          hint={
            unit.km_updated_at
              ? `Diperbarui ${formatDateTimeWita(unit.km_updated_at)}`
              : "Belum ada reading"
          }
          icon={Activity}
          tone="emerald"
        />
        <MetricCard
          label="Sinkronisasi"
          value={unit.synced_at ? "Tersinkron" : "Data lokal"}
          hint={unit.synced_at ? formatDateTimeWita(unit.synced_at) : "Belum pernah sync"}
          icon={CalendarClock}
          tone="violet"
        />
      </div>

      <div
        className="flex gap-1 overflow-x-auto rounded-lg border border-border bg-muted/30 p-1"
        role="tablist"
        aria-label="Profil unit"
      >
        {unitTabs.map((item) => {
          const Icon = item.icon;
          const active = tab === item.id;
          return (
            <Button
              key={item.id}
              type="button"
              size="sm"
              variant={active ? "secondary" : "ghost"}
              className="shrink-0"
              role="tab"
              aria-selected={active}
              onClick={() => setTab(item.id)}
            >
              <Icon aria-hidden="true" />
              {item.label}
            </Button>
          );
        })}
      </div>

      {tab === "overview" ? (
        <Card>
          <CardHeader className="border-b">
            <CardTitle>Identitas unit</CardTitle>
            <CardDescription>Data master dan model yang terpasang.</CardDescription>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <Detail
                label="Unit model"
                value={
                  unit.unit_model
                    ? `${unit.unit_model.manufacturer} ${unit.unit_model.model}`
                    : "-"
                }
              />
              <Detail
                label="Engine model"
                value={
                  unit.engine_model
                    ? `${unit.engine_model.manufacturer} ${unit.engine_model.model}`
                    : "-"
                }
              />
              <Detail label="Nomor polisi" value={unit.license_plate || "-"} />
              <Detail label="Serial unit" value={unit.serial_number || "-"} />
              <Detail label="Serial engine" value={unit.engine_serial_number || "-"} />
              <Detail label="Kelas unit" value={unit.class_unit || "-"} />
              <Detail label="Lokasi" value={unit.site_location || "-"} icon={MapPin} />
            </dl>
          </CardContent>
        </Card>
      ) : null}

      {tab === "timeline" ? <UnitTimelinePanel unitId={unit.id} /> : null}
      {tab === "components" ? <UnitComponentPanel unitId={unit.id} /> : null}

      {tab === "meters" ? (
        <div className="flex flex-col gap-4">
          {canReadMeter ? <MeterReadingForm unit={unit} onSaved={() => void load()} /> : null}

          <Card className="gap-0 overflow-hidden">
            <CardHeader className="border-b">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <CardTitle>Riwayat HM/KM</CardTitle>
                  <CardDescription>Terbaru 50 reading, waktu WITA.</CardDescription>
                </div>
                <Badge variant="secondary" className="h-6 shrink-0 px-2.5">
                  {readings.length} reading
                </Badge>
              </div>
            </CardHeader>

            {readings.length === 0 ? (
              <CardContent className="py-10">
                <Empty className="border-0">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <ClipboardList aria-hidden="true" />
                    </EmptyMedia>
                    <EmptyTitle>Belum ada reading.</EmptyTitle>
                    <EmptyDescription>
                      Reading HM/KM yang disimpan akan muncul di sini.
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              </CardContent>
            ) : (
              <div>
                <Table>
                  <TableCaption id="riwayat-reading" className="sr-only">
                    Riwayat pembacaan HM dan KM unit {unit.code}
                  </TableCaption>
                  <TableHeader>
                    <TableRow>
                      <TableHead scope="col">Waktu</TableHead>
                      <TableHead scope="col">Tipe</TableHead>
                      <TableHead scope="col" className="text-right">Nilai</TableHead>
                      <TableHead scope="col">Sumber</TableHead>
                      <TableHead scope="col">Catatan</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {readings.map((reading) => (
                      <TableRow key={reading.id}>
                        <TableHead scope="row" className="whitespace-normal">
                          {formatDateTimeWita(reading.recorded_at)}
                        </TableHead>
                        <TableCell>
                          <Badge variant="outline" className="h-6 px-2.5 uppercase">
                            {reading.reading_type}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right font-medium">
                          {formatNumber(reading.value)}
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary" className="h-6 px-2.5">
                            {sourceLabels[reading.source_type]}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {reading.notes || "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </Card>
        </div>
      ) : null}

      {tab === "downtime" ? <UnitDowntimePanel unitId={unit.id} /> : null}
      {tab === "intervals" ? <UnitIntervalPanel unitId={unit.id} /> : null}
    </div>
  );
}

function MeterReadingForm({ unit, onSaved }: { unit: Unit; onSaved: () => void }) {
  const [readingType, setReadingType] = useState<"hm" | "km">("hm");
  const [value, setValue] = useState("");
  const [recordedAt, setRecordedAt] = useState(() => toDatetimeLocalWita());
  const [notes, setNotes] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  const fieldIds: Record<string, string> = {
    value: "reading-value",
    recorded_at: "reading-time",
    notes: "reading-notes",
  };

  const clearFieldError = (key: string) => {
    setFieldErrors((current) => {
      if (!current[key]) return current;
      const next = { ...current };
      delete next[key];
      return next;
    });
  };

  const focusField = (id: string) => {
    requestAnimationFrame(() => document.getElementById(id)?.focus());
  };

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setFieldErrors({});

    if (value.trim() === "" || !Number.isFinite(Number(value))) {
      setFieldErrors({ value: "Nilai HM/KM wajib diisi" });
      focusField("reading-value");
      return;
    }
    if (!recordedAt) {
      setFieldErrors({ recorded_at: "Waktu reading wajib diisi" });
      focusField("reading-time");
      return;
    }

    setPending(true);
    try {
      await proxyPost(`/units/${unit.id}/meter-readings`, {
        reading_type: readingType,
        value: Number(value),
        recorded_at: witaLocalToISOString(recordedAt),
        notes: notes.trim(),
      });
      setMessage("Reading tersimpan.");
      setValue("");
      setNotes("");
      setRecordedAt(toDatetimeLocalWita());
      onSaved();
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
        const firstField = Object.keys(submitError.fields)[0];
        focusField(firstField ? (fieldIds[firstField] ?? "reading-form-error") : "reading-form-error");
      } else {
        setError("Reading tidak dapat disimpan.");
        focusField("reading-form-error");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>Tambah reading manual</CardTitle>
        <CardDescription>
          Reading yang lebih rendah dari nilai saat ini hanya dapat dikoreksi admin dengan
          alasan.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form className="grid gap-4 sm:grid-cols-2" onSubmit={(event) => void submit(event)} noValidate>
          <FieldSet className="space-y-2 sm:col-span-2">
            <FieldLegend className="text-sm font-medium">Tipe reading</FieldLegend>
            <div className="grid max-w-md grid-cols-2 gap-2">
              {(["hm", "km"] as const).map((type) => (
                <label
                  key={type}
                  className="flex h-9 cursor-pointer items-center justify-center rounded-md border border-input bg-transparent px-3 text-sm font-medium shadow-xs transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary has-[:checked]:text-primary-foreground has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50"
                >
                  <input
                    type="radio"
                    name="reading_type"
                    value={type}
                    checked={readingType === type}
                    onChange={() => setReadingType(type)}
                    className="sr-only"
                  />
                  {type.toUpperCase()}
                </label>
              ))}
            </div>
          </FieldSet>

          <Field data-invalid={Boolean(fieldErrors.value)}>
            <FieldLabel htmlFor="reading-value">
              Nilai <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="reading-value"
              name="value"
              type="number"
              inputMode="decimal"
              min="0"
              step="0.1"
              required
              value={value}
              onChange={(event) => {
                setValue(event.target.value);
                clearFieldError("value");
              }}
              aria-invalid={Boolean(fieldErrors.value)}
              aria-describedby={fieldErrors.value ? "reading-value-error" : undefined}
            />
            {fieldErrors.value ? (
              <FieldError id="reading-value-error">{fieldErrors.value}</FieldError>
            ) : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.recorded_at)}>
            <FieldLabel htmlFor="reading-time">
              Waktu (WITA) <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="reading-time"
              name="recorded_at"
              type="datetime-local"
              required
              value={recordedAt}
              onChange={(event) => {
                setRecordedAt(event.target.value);
                clearFieldError("recorded_at");
              }}
              aria-invalid={Boolean(fieldErrors.recorded_at)}
              aria-describedby={fieldErrors.recorded_at ? "reading-time-error" : undefined}
            />
            {fieldErrors.recorded_at ? (
              <FieldError id="reading-time-error">{fieldErrors.recorded_at}</FieldError>
            ) : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.notes)} className="sm:col-span-2">
            <FieldLabel htmlFor="reading-notes">Catatan</FieldLabel>
            <Textarea
              id="reading-notes"
              name="notes"
              rows={3}
              value={notes}
              onChange={(event) => {
                setNotes(event.target.value);
                clearFieldError("notes");
              }}
              aria-invalid={Boolean(fieldErrors.notes)}
              aria-describedby={fieldErrors.notes ? "reading-notes-error" : "reading-notes-hint"}
            />
            {fieldErrors.notes ? (
              <FieldError id="reading-notes-error">{fieldErrors.notes}</FieldError>
            ) : (
              <FieldDescription id="reading-notes-hint">
                Wajib minimal 5 karakter untuk koreksi reading yang lebih rendah.
              </FieldDescription>
            )}
          </Field>

          {error ? (
            <Alert
              id="reading-form-error"
              tabIndex={-1}
              variant="destructive"
              className="border-destructive/30 bg-destructive/5 sm:col-span-2"
            >
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Reading tidak dapat disimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          {message ? (
            <Alert
              role="status"
              className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 sm:col-span-2 dark:text-emerald-300"
            >
              <CircleCheck aria-hidden="true" />
              <AlertTitle>Berhasil</AlertTitle>
              <AlertDescription className="text-emerald-700 dark:text-emerald-300">
                {message}
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="sm:col-span-2">
            <Button type="submit" disabled={pending}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" />
                  Menyimpan...
                </>
              ) : (
                "Simpan reading"
              )}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

const metricTones = {
  sky: "bg-muted text-muted-foreground",
  emerald: "bg-muted text-muted-foreground",
  violet: "bg-muted text-muted-foreground",
} as const;

function MetricCard({
  label,
  value,
  hint,
  icon: Icon,
  tone,
}: {
  label: string;
  value: string;
  hint: string;
  icon: LucideIcon;
  tone: keyof typeof metricTones;
}) {
  return (
    <div className="rounded-xl border border-border bg-background p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">{label}</p>
          <p className="mt-2 text-2xl font-semibold">{value}</p>
          <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
        </div>
        <span className={cn("grid size-9 shrink-0 place-items-center rounded-md", metricTones[tone])}>
          <Icon className="size-5" aria-hidden="true" />
        </span>
      </div>
    </div>
  );
}

function Detail({
  label,
  value,
  icon: Icon,
}: {
  label: string;
  value: string;
  icon?: LucideIcon;
}) {
  return (
    <div>
      <dt className="type-overline flex items-center gap-1.5 text-muted-foreground">
        {Icon ? <Icon className="size-3.5" aria-hidden="true" /> : null}
        {label}
      </dt>
      <dd className="mt-1 text-sm">{value}</dd>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-4 w-16" />
        <Skeleton className="mt-3 h-8 w-40" />
        <Skeleton className="mt-2 h-4 w-64" />
      </div>
      <div className="grid gap-4 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, index) => (
          <Skeleton key={index} className="h-28 rounded-xl" />
        ))}
      </div>
      <Skeleton className="h-56 rounded-xl" />
      <Skeleton className="h-72 rounded-xl" />
    </div>
  );
}
