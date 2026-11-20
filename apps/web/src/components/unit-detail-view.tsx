"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type { DataResponse, MeterReading, Unit } from "@/lib/types";
import {
  formatDateTimeWita,
  formatNumber,
  toDatetimeLocalWita,
  witaLocalToISOString,
} from "@/lib/time";

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
      setError(loadError instanceof ApiError ? loadError.message : "Detail unit tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, [unitId]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading && !unit) {
    return <p role="status" className="text-sm text-muted-foreground">Memuat unit...</p>;
  }

  if (error && !unit) {
    return (
      <div className="space-y-3">
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
        <Link href="/units" className="text-sm font-medium underline underline-offset-4">
          Kembali ke daftar unit
        </Link>
      </div>
    );
  }

  if (!unit) {
    return null;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link href="/units" className="text-sm text-muted-foreground underline-offset-4 hover:underline">
            Unit
          </Link>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight">{unit.code}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {[unit.merk, unit.unit_type, unit.site_location].filter(Boolean).join(" · ") || "Identitas unit"}
          </p>
        </div>
        {canWrite ? (
          <Link
            href={`/units/${unit.id}/edit`}
            className="inline-flex min-h-11 items-center rounded-md border border-border px-4 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            Edit unit
          </Link>
        ) : null}
      </div>

      {error ? (
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-3">
        <Metric label="HM terakhir" value={formatNumber(unit.current_hm)} hint={formatDateTimeWita(unit.hm_updated_at)} />
        <Metric label="KM terakhir" value={formatNumber(unit.current_km)} hint={formatDateTimeWita(unit.km_updated_at)} />
        <Metric
          label="Status"
          value={!unit.is_active ? "Nonaktif" : unit.is_breakdown ? "Breakdown" : "Aktif"}
          hint={unit.synced_at ? `Sync ${formatDateTimeWita(unit.synced_at)}` : "Data lokal"}
        />
      </div>

      <section aria-labelledby="identitas-unit" className="rounded-xl border border-border bg-background p-5">
        <h2 id="identitas-unit" className="text-base font-semibold">Identitas</h2>
        <dl className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Detail label="Unit model" value={unit.unit_model ? `${unit.unit_model.manufacturer} ${unit.unit_model.model}` : "-"} />
          <Detail label="Engine model" value={unit.engine_model ? `${unit.engine_model.manufacturer} ${unit.engine_model.model}` : "-"} />
          <Detail label="Nomor polisi" value={unit.license_plate || "-"} />
          <Detail label="Serial unit" value={unit.serial_number || "-"} />
          <Detail label="Serial engine" value={unit.engine_serial_number || "-"} />
          <Detail label="Kelas unit" value={unit.class_unit || "-"} />
        </dl>
      </section>

      {canReadMeter ? <MeterReadingForm unit={unit} onSaved={() => void load()} /> : null}

      <section aria-labelledby="riwayat-reading" className="overflow-hidden rounded-xl border border-border bg-background">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
          <h2 id="riwayat-reading" className="text-base font-semibold">Riwayat HM/KM</h2>
          <p className="text-xs text-muted-foreground">Terbaru 50 reading, waktu WITA</p>
        </div>
        {readings.length === 0 ? (
          <p className="px-4 py-10 text-center text-sm text-muted-foreground">Belum ada reading.</p>
        ) : (
          <div className="responsive-table-wrapper" tabIndex={0} role="region" aria-labelledby="riwayat-reading">
            <table className="responsive-table">
              <caption className="sr-only">Riwayat pembacaan HM dan KM unit {unit.code}</caption>
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
                    <td data-label="Tipe">{reading.reading_type.toUpperCase()}</td>
                    <td data-label="Nilai" className="text-right tabular-nums">{formatNumber(reading.value)}</td>
                    <td data-label="Sumber">{reading.source_type}</td>
                    <td data-label="Catatan">{reading.notes || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
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
    <section aria-labelledby="form-reading" className="rounded-xl border border-border bg-background p-5">
      <h2 id="form-reading" className="text-base font-semibold">Tambah reading manual</h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Reading yang lebih rendah dari nilai saat ini hanya dapat dikoreksi admin dengan alasan.
      </p>
      <form className="mt-5 grid gap-4 sm:grid-cols-2" onSubmit={(event) => void submit(event)} noValidate>
        <div>
          <label htmlFor="reading-type" className="text-sm font-medium">Tipe</label>
          <select
            id="reading-type"
            value={readingType}
            onChange={(event) => setReadingType(event.target.value as "hm" | "km")}
            className="mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
          >
            <option value="hm">HM</option>
            <option value="km">KM</option>
          </select>
        </div>
        <div>
          <label htmlFor="reading-value" className="text-sm font-medium">Nilai</label>
          <input
            id="reading-value"
            type="number"
            min="0"
            step="0.1"
            required
            value={value}
            onChange={(event) => setValue(event.target.value)}
            aria-invalid={Boolean(fieldErrors.value)}
            aria-describedby={fieldErrors.value ? "reading-value-error" : undefined}
            className="mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          {fieldErrors.value ? <p id="reading-value-error" className="mt-1 text-sm text-destructive">{fieldErrors.value}</p> : null}
        </div>
        <div>
          <label htmlFor="reading-time" className="text-sm font-medium">Waktu (WITA)</label>
          <input
            id="reading-time"
            type="datetime-local"
            required
            value={recordedAt}
            onChange={(event) => setRecordedAt(event.target.value)}
            aria-invalid={Boolean(fieldErrors.recorded_at)}
            aria-describedby={fieldErrors.recorded_at ? "reading-time-error" : undefined}
            className="mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          {fieldErrors.recorded_at ? <p id="reading-time-error" className="mt-1 text-sm text-destructive">{fieldErrors.recorded_at}</p> : null}
        </div>
        <div className="sm:col-span-2">
          <label htmlFor="reading-notes" className="text-sm font-medium">Catatan</label>
          <textarea
            id="reading-notes"
            rows={3}
            value={notes}
            onChange={(event) => setNotes(event.target.value)}
            aria-invalid={Boolean(fieldErrors.notes)}
            aria-describedby={fieldErrors.notes ? "reading-notes-error" : "reading-notes-hint"}
            className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
          />
          {fieldErrors.notes ? (
            <p id="reading-notes-error" className="mt-1 text-sm text-destructive">{fieldErrors.notes}</p>
          ) : (
            <p id="reading-notes-hint" className="mt-1 text-xs text-muted-foreground">
              Wajib minimal 5 karakter untuk koreksi reading yang lebih rendah.
            </p>
          )}
        </div>
        {error ? (
          <p role="alert" className="sm:col-span-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            {error}
          </p>
        ) : null}
        {message ? (
          <p role="status" className="sm:col-span-2 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
            {message}
          </p>
        ) : null}
        <div className="sm:col-span-2">
          <button
            type="submit"
            disabled={pending}
            className="min-h-11 rounded-md bg-primary px-4 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            {pending ? "Menyimpan..." : "Simpan reading"}
          </button>
        </div>
      </form>
    </section>
  );
}

function Metric({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="rounded-xl border border-border bg-background p-5">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
      <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
    </div>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-sm">{value}</dd>
    </div>
  );
}
