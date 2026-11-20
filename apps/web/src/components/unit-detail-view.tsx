"use client";

import {
  Activity,
  ArrowLeft,
  CalendarClock,
  CircleAlert,
  CircleCheck,
  ClipboardList,
  Gauge,
  Loader2,
  MapPin,
  Pencil,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { UnitStatusBadge } from "@/components/status-badges";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type { DataResponse, MeterReading, Unit } from "@/lib/types";
import {
  formatDateTimeWita,
  formatNumber,
  toDatetimeLocalWita,
  witaLocalToISOString,
} from "@/lib/time";

const sourceLabels: Record<MeterReading["source_type"], string> = {
  p2h: "P2H",
  wo: "WO",
  manual: "Manual",
  sync: "Sync",
};

export function UnitDetailView({ unitId }: { unitId: number }) {
  const { user } = useSession();
  const [unit, setUnit] = useState<Unit | null>(null);
  const [readings, setReadings] = useState<MeterReading[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

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
          className="inline-flex min-h-11 items-center gap-2 text-sm font-medium text-primary underline-offset-4 hover:underline"
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
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link
            href="/units"
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Unit
          </Link>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">{unit.code}</h1>
            <UnitStatusBadge active={unit.is_active} breakdown={unit.is_breakdown} />
          </div>
          <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
        </div>
        {canWrite ? (
          <Link
            href={`/units/${unit.id}/edit`}
            className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-border bg-background px-4 text-sm font-medium shadow-sm transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            <Pencil className="size-4" aria-hidden="true" />
            Edit unit
          </Link>
        ) : null}
      </div>

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

      <Card className="shadow-sm">
        <CardHeader className="border-b">
          <CardTitle>Identitas unit</CardTitle>
          <CardDescription>Data master dan model yang terpasang.</CardDescription>
        </CardHeader>
        <CardContent className="pt-4">
          <dl className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
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

      {canReadMeter ? <MeterReadingForm unit={unit} onSaved={() => void load()} /> : null}

      <Card className="overflow-hidden shadow-sm">
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
          <CardContent className="py-12 text-center">
            <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-muted text-muted-foreground">
              <ClipboardList className="size-6" aria-hidden="true" />
            </span>
            <p className="mt-4 font-medium">Belum ada reading.</p>
            <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
              Reading HM/KM yang disimpan akan muncul di sini.
            </p>
          </CardContent>
        ) : (
          <div
            className="responsive-table-wrapper"
            tabIndex={0}
            role="region"
            aria-labelledby="riwayat-reading"
          >
            <table className="responsive-table">
              <caption id="riwayat-reading" className="sr-only">
                Riwayat pembacaan HM dan KM unit {unit.code}
              </caption>
              <thead>
                <tr>
                  <th scope="col">Waktu</th>
                  <th scope="col">Tipe</th>
                  <th scope="col" className="text-right">Nilai</th>
                  <th scope="col">Sumber</th>
                  <th scope="col">Catatan</th>
                </tr>
              </thead>
              <tbody>
                {readings.map((reading) => (
                  <tr key={reading.id}>
                    <td data-label="Waktu">{formatDateTimeWita(reading.recorded_at)}</td>
                    <td data-label="Tipe">
                      <Badge variant="outline" className="h-6 px-2.5 uppercase">
                        {reading.reading_type}
                      </Badge>
                    </td>
                    <td data-label="Nilai" className="text-right font-medium tabular-nums">
                      {formatNumber(reading.value)}
                    </td>
                    <td data-label="Sumber">
                      <Badge variant="secondary" className="h-6 px-2.5">
                        {sourceLabels[reading.source_type]}
                      </Badge>
                    </td>
                    <td data-label="Catatan" className="text-muted-foreground">
                      {reading.notes || "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
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

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setFieldErrors({});

    if (value.trim() === "" || !Number.isFinite(Number(value))) {
      setFieldErrors({ value: "Nilai HM/KM wajib diisi" });
      return;
    }
    if (!recordedAt) {
      setFieldErrors({ recorded_at: "Waktu reading wajib diisi" });
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
      } else {
        setError("Reading tidak dapat disimpan.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Card className="shadow-sm">
      <CardHeader className="border-b">
        <CardTitle>Tambah reading manual</CardTitle>
        <CardDescription>
          Reading yang lebih rendah dari nilai saat ini hanya dapat dikoreksi admin dengan
          alasan.
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-4">
        <form className="grid gap-5 sm:grid-cols-2" onSubmit={(event) => void submit(event)} noValidate>
          <fieldset className="space-y-2 sm:col-span-2">
            <legend className="text-sm font-medium">Tipe reading</legend>
            <div className="grid max-w-md grid-cols-2 gap-2">
              {(["hm", "km"] as const).map((type) => (
                <label
                  key={type}
                  className="flex min-h-11 cursor-pointer items-center justify-center rounded-lg border border-input bg-background px-3 text-sm font-medium transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary has-[:checked]:text-primary-foreground has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50"
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
          </fieldset>

          <div className="space-y-2">
            <Label htmlFor="reading-value">Nilai</Label>
            <Input
              id="reading-value"
              name="value"
              type="number"
              inputMode="decimal"
              min="0"
              step="0.1"
              required
              value={value}
              onChange={(event) => setValue(event.target.value)}
              aria-invalid={Boolean(fieldErrors.value)}
              aria-describedby={fieldErrors.value ? "reading-value-error" : undefined}
            />
            {fieldErrors.value ? (
              <p id="reading-value-error" className="text-sm text-destructive">
                {fieldErrors.value}
              </p>
            ) : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="reading-time">Waktu (WITA)</Label>
            <Input
              id="reading-time"
              name="recorded_at"
              type="datetime-local"
              required
              value={recordedAt}
              onChange={(event) => setRecordedAt(event.target.value)}
              aria-invalid={Boolean(fieldErrors.recorded_at)}
              aria-describedby={fieldErrors.recorded_at ? "reading-time-error" : undefined}
            />
            {fieldErrors.recorded_at ? (
              <p id="reading-time-error" className="text-sm text-destructive">
                {fieldErrors.recorded_at}
              </p>
            ) : null}
          </div>

          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="reading-notes">Catatan</Label>
            <textarea
              id="reading-notes"
              name="notes"
              rows={3}
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
              aria-invalid={Boolean(fieldErrors.notes)}
              aria-describedby={fieldErrors.notes ? "reading-notes-error" : "reading-notes-hint"}
              className="w-full resize-y rounded-lg border border-input bg-background px-3 py-2 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20"
            />
            {fieldErrors.notes ? (
              <p id="reading-notes-error" className="text-sm text-destructive">
                {fieldErrors.notes}
              </p>
            ) : (
              <p id="reading-notes-hint" className="text-xs text-muted-foreground">
                Wajib minimal 5 karakter untuk koreksi reading yang lebih rendah.
              </p>
            )}
          </div>

          {error ? (
            <Alert
              variant="destructive"
              className="border-destructive/30 bg-destructive/5 sm:col-span-2"
            >
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Reading tidak dapat disimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          {message ? (
            <Alert className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 sm:col-span-2 dark:text-emerald-300">
              <CircleCheck aria-hidden="true" />
              <AlertTitle>Berhasil</AlertTitle>
              <AlertDescription className="text-emerald-700 dark:text-emerald-300">
                {message}
              </AlertDescription>
            </Alert>
          ) : null}

          <div className="sm:col-span-2">
            <Button type="submit" disabled={pending} className="h-11 px-4">
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
  sky: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
  emerald: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  violet: "bg-violet-500/10 text-violet-700 dark:text-violet-300",
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
    <div className="rounded-xl border border-border bg-card p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">{label}</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
          <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
        </div>
        <span className={`grid size-10 shrink-0 place-items-center rounded-xl ${metricTones[tone]}`}>
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
      <dt className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
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
