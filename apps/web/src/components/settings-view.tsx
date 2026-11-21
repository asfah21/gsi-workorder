"use client";

import { CircleAlert, Loader2, Save, Settings2, ShieldAlert } from "lucide-react";
import { useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError, proxyGet, proxyPatch } from "@/lib/api-client";
import type { AppSettings, DataResponse } from "@/lib/types";
import { formatDateTimeWita } from "@/lib/time";

const settingLabels: Record<string, string> = {
  p2h_auto_wo: "Auto-WO P2H",
  candidate_sla_hours: "SLA reminder (jam)",
  candidate_backfill_days: "Backfill (hari)",
};

export function SettingsView() {
  const { user } = useSession();
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [autoWO, setAutoWO] = useState(false);
  const [slaHours, setSlaHours] = useState("24");
  const [backfillDays, setBackfillDays] = useState("30");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const isAdmin = user.role === "admin";

  useEffect(() => {
    if (!isAdmin) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    proxyGet<DataResponse<AppSettings>>("/settings")
      .then((response) => {
        if (cancelled) return;
        const result = response.data;
        setSettings(result);
        setAutoWO(result.p2h_auto_wo);
        setSlaHours(String(result.candidate_sla_hours));
        setBackfillDays(String(result.candidate_backfill_days));
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(
            loadError instanceof ApiError
              ? loadError.message
              : "Pengaturan tidak dapat dimuat.",
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [isAdmin]);

  const save = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSaving(true);
    setError("");
    setSuccess("");
    setFieldErrors({});
    try {
      const response = await proxyPatch<DataResponse<AppSettings>>("/settings", {
        p2h_auto_wo: autoWO,
        candidate_sla_hours: Number(slaHours),
        candidate_backfill_days: Number(backfillDays),
      });
      const result = response.data;
      setSettings(result);
      setAutoWO(result.p2h_auto_wo);
      setSlaHours(String(result.candidate_sla_hours));
      setBackfillDays(String(result.candidate_backfill_days));
      setSuccess("Pengaturan disimpan dan perubahan dicatat di riwayat.");
    } catch (saveError) {
      if (saveError instanceof ApiError) {
        setError(saveError.message);
        setFieldErrors(saveError.fields);
      } else {
        setError("Pengaturan tidak dapat disimpan.");
      }
    } finally {
      setSaving(false);
    }
  };

  if (!isAdmin) {
    return (
      <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
        <ShieldAlert aria-hidden="true" />
        <AlertTitle>Akses admin diperlukan</AlertTitle>
        <AlertDescription>
          Halaman pengaturan hanya dapat dibuka oleh admin.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="flex flex-row flex-wrap items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold">Pengaturan</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Kontrol global P2H dan kandidat work order. Setiap perubahan diaudit.
            </p>
          </div>
          <Badge variant="secondary" className="h-6 px-2.5">
            Admin
          </Badge>
        </CardContent>
      </Card>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      {success ? (
        <Alert className="border-emerald-500/30 bg-emerald-500/5">
          <Badge variant="secondary" className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300">
            Tersimpan
          </Badge>
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-12">
        <Card className="lg:col-span-7">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings2 className="size-4" aria-hidden="true" />
              P2H &amp; usulan
            </CardTitle>
            <CardDescription>
              Default aman: usulan P2H tidak otomatis menjadi WO.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="animate-spin" aria-hidden="true" />
                Memuat pengaturan...
              </div>
            ) : (
              <form onSubmit={save} className="space-y-5">
                <div className="flex items-start justify-between gap-4 rounded-lg border border-border p-4">
                  <div>
                    <p className="text-sm font-semibold">Auto-WO P2H</p>
                    <p className="mt-1 max-w-xl text-sm text-muted-foreground">
                      Jika aktif, kandidat baru dengan temuan langsung dibuatkan WO.
                      Default nonaktif; hanya kandidat baru yang terpengaruh.
                    </p>
                  </div>
                  <label className="flex cursor-pointer items-center gap-2">
                    <input
                      type="checkbox"
                      checked={autoWO}
                      onChange={(event) => setAutoWO(event.target.checked)}
                      className="size-4 rounded border-input accent-primary"
                    />
                    <span className="text-sm font-medium">{autoWO ? "Aktif" : "Nonaktif"}</span>
                  </label>
                </div>

                {autoWO ? (
                  <Alert className="border-amber-500/30 bg-amber-500/5">
                    <ShieldAlert aria-hidden="true" />
                    <AlertTitle>Perhatian</AlertTitle>
                    <AlertDescription>
                      Auto-WO akan membuat work order tanpa persetujuan planner. Aktifkan
                      hanya jika proses operasional sudah disepakati.
                    </AlertDescription>
                  </Alert>
                ) : null}

                <div className="grid gap-4 sm:grid-cols-2">
                  <Field data-invalid={Boolean(fieldErrors.candidate_sla_hours)}>
                    <FieldLabel htmlFor="sla-hours">SLA reminder (jam)</FieldLabel>
                    <Input
                      id="sla-hours"
                      type="number"
                      min={1}
                      max={168}
                      value={slaHours}
                      onChange={(event) => setSlaHours(event.target.value)}
                      aria-invalid={Boolean(fieldErrors.candidate_sla_hours)}
                    />
                    <p className="text-xs text-muted-foreground">
                      Reminder dikirim sekali setelah kandidat pending melewati SLA.
                    </p>
                    {fieldErrors.candidate_sla_hours ? (
                      <FieldError>{fieldErrors.candidate_sla_hours}</FieldError>
                    ) : null}
                  </Field>
                  <Field data-invalid={Boolean(fieldErrors.candidate_backfill_days)}>
                    <FieldLabel htmlFor="backfill-days">Backfill (hari)</FieldLabel>
                    <Input
                      id="backfill-days"
                      type="number"
                      min={0}
                      max={3650}
                      value={backfillDays}
                      onChange={(event) => setBackfillDays(event.target.value)}
                      aria-invalid={Boolean(fieldErrors.candidate_backfill_days)}
                    />
                    <p className="text-xs text-muted-foreground">
                      Laporan lebih lama dari batas ini masuk sebagai kandidat arsip.
                    </p>
                    {fieldErrors.candidate_backfill_days ? (
                      <FieldError>{fieldErrors.candidate_backfill_days}</FieldError>
                    ) : null}
                  </Field>
                </div>

                <Button type="submit" disabled={saving}>
                  {saving ? (
                    <Loader2 className="animate-spin" aria-hidden="true" />
                  ) : (
                    <Save aria-hidden="true" />
                  )}
                  Simpan pengaturan
                </Button>
              </form>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-5">
          <CardHeader>
            <CardTitle>Ringkasan nilai</CardTitle>
            <CardDescription>Nilai aktif saat ini.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <SummaryRow label="Auto-WO" value={settings?.p2h_auto_wo ? "Aktif" : "Nonaktif"} />
            <SummaryRow label="SLA reminder" value={`${settings?.candidate_sla_hours ?? "-"} jam`} />
            <SummaryRow label="Backfill" value={`${settings?.candidate_backfill_days ?? "-"} hari`} />
            <SummaryRow
              label="Terakhir diubah"
              value={settings ? formatDateTimeWita(settings.updated_at) : "-"}
            />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Riwayat perubahan</CardTitle>
          <CardDescription>50 perubahan terbaru dari app_settings_history.</CardDescription>
        </CardHeader>
        <CardContent>
          {settings && settings.history.length > 0 ? (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Waktu</TableHead>
                    <TableHead>Setting</TableHead>
                    <TableHead>Lama</TableHead>
                    <TableHead>Baru</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {settings.history.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        {formatDateTimeWita(item.changed_at)}
                      </TableCell>
                      <TableCell className="font-medium">
                        {settingLabels[item.key] || item.key}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {item.old_value || "-"}
                      </TableCell>
                      <TableCell>{item.new_value || "-"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Belum ada perubahan.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border pb-2 last:border-0 last:pb-0">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  );
}
