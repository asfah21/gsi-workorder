"use client";

import {
  ArrowRightLeft,
  CircleAlert,
  CircleCheck,
  Cog,
  History,
  Loader2,
  Plus,
  Repeat2,
  TriangleAlert,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

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
import type {
  ComponentInstallation,
  ComponentLifetime,
  ComponentOption,
  DataResponse,
  ListResponse,
  Unit,
} from "@/lib/types";
import {
  formatDateTimeWita,
  formatNumber,
  toDatetimeLocalWita,
  witaLocalToISOString,
} from "@/lib/time";

type LifetimeState = ComponentLifetime["lifetime_state"];

const stateLabels: Record<LifetimeState, string> = {
  inactive: "Nonaktif",
  unset: "Tanpa lifetime",
  safe: "Aman",
  upcoming: "Akan jatuh tempo",
  due: "Jatuh tempo",
  overdue: "Overdue",
};

const statusLabels: Record<ComponentInstallation["status"], string> = {
  active: "Terpasang",
  removed: "Dilepas",
  transferred: "Dipindah",
  scrapped: "Scrap",
};

function LifetimeStateBadge({ state }: { state: LifetimeState }) {
  const variant =
    state === "overdue"
      ? "destructive"
      : state === "due"
        ? "default"
        : state === "upcoming"
          ? "secondary"
          : "outline";
  return (
    <Badge variant={variant} className="h-5 px-2">
      {stateLabels[state]}
    </Badge>
  );
}

function InstallationStatusBadge({ status }: { status: ComponentInstallation["status"] }) {
  return (
    <Badge variant={status === "active" ? "secondary" : "outline"} className="h-5 px-2">
      {statusLabels[status]}
    </Badge>
  );
}

export function UnitInstallationPanel({ unit }: { unit: Unit }) {
  const { user } = useSession();
  const [items, setItems] = useState<ComponentLifetime[]>([]);
  const [components, setComponents] = useState<ComponentOption[]>([]);
  const [units, setUnits] = useState<Unit[]>([]);
  const [loading, setLoading] = useState(true);
  const [unitsLoading, setUnitsLoading] = useState(false);
  const [error, setError] = useState("");
  const [unitsError, setUnitsError] = useState("");
  const [message, setMessage] = useState("");
  const [installOpen, setInstallOpen] = useState(false);
  const [replaceTarget, setReplaceTarget] = useState<ComponentLifetime | null>(null);
  const [transferTarget, setTransferTarget] = useState<ComponentLifetime | null>(null);

  const canWrite = user.role === "admin" || user.role === "planner";

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [lifetimeResult, componentResult] = await Promise.all([
        proxyGet<DataResponse<ComponentLifetime[]>>(`/units/${unit.id}/component-lifetime`),
        proxyGet<DataResponse<ComponentOption[]>>("/components?active_only=true"),
      ]);
      setItems(lifetimeResult.data);
      setComponents(componentResult.data);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Data komponen unit tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [unit.id]);

  const loadUnits = useCallback(async () => {
    setUnitsLoading(true);
    setUnitsError("");
    try {
      const limit = 100;
      let page = 1;
      let total = 0;
      const collected: Unit[] = [];
      do {
        const result = await proxyGet<ListResponse<Unit>>(
          `/units?is_active=true&page=${page}&limit=${limit}`,
        );
        collected.push(...result.data);
        total = result.meta.total;
        if (result.data.length === 0) break;
        page += 1;
      } while (collected.length < total && page <= 10);
      setUnits(collected);
    } catch (loadError) {
      setUnitsError(
        loadError instanceof ApiError
          ? loadError.message
          : "Daftar unit tujuan tidak dapat dimuat.",
      );
    } finally {
      setUnitsLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (canWrite) {
      void loadUnits();
    }
  }, [canWrite, loadUnits]);

  const activeItems = useMemo(
    () => items.filter((item) => item.status === "active"),
    [items],
  );
  const historyItems = useMemo(
    () => items.filter((item) => item.status !== "active"),
    [items],
  );
  const attentionItems = useMemo(
    () =>
      activeItems.filter(
        (item) => item.lifetime_state === "due" || item.lifetime_state === "overdue",
      ),
    [activeItems],
  );

  const handleSaved = (nextMessage: string) => {
    setMessage(nextMessage);
    setInstallOpen(false);
    setReplaceTarget(null);
    setTransferTarget(null);
    void load();
  };

  return (
    <div className="flex flex-col gap-4">
      {message ? (
        <Alert role="status" className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300">
          <CircleCheck aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription className="text-emerald-700 dark:text-emerald-300">
            {message}
          </AlertDescription>
        </Alert>
      ) : null}

      {attentionItems.length > 0 ? (
        <Alert>
          <TriangleAlert aria-hidden="true" />
          <AlertTitle>Komponen perlu perhatian</AlertTitle>
          <AlertDescription>
            {attentionItems
              .map(
                (item) =>
                  `${item.component_name} (${item.serial_number}) ${stateLabels[item.lifetime_state].toLowerCase()}`,
              )
              .join(", ")}
            .
          </AlertDescription>
        </Alert>
      ) : null}

      <Card className="gap-0 overflow-hidden">
        <CardHeader className="border-b">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Cog className="size-4 text-muted-foreground" aria-hidden="true" />
                Komponen terpasang
              </CardTitle>
              <CardDescription>
                Serial, pemakaian, dan sisa lifetime komponen aktif.
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="secondary" className="h-6 px-2.5">
                {activeItems.length} aktif
              </Badge>
              {canWrite ? (
                <Button type="button" size="sm" onClick={() => setInstallOpen(true)}>
                  <Plus aria-hidden="true" /> Pasang komponen
                </Button>
              ) : null}
            </div>
          </div>
        </CardHeader>

        {error ? (
          <CardContent className="py-6">
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Komponen unit tidak dapat dimuat</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          </CardContent>
        ) : loading ? (
          <CardContent className="flex flex-col gap-2 py-6">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-14 w-full" />
            ))}
          </CardContent>
        ) : activeItems.length === 0 ? (
          <CardContent className="py-10">
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Cog aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada komponen terpasang.</EmptyTitle>
                <EmptyDescription>
                  {canWrite
                    ? "Pasang komponen dari master untuk mulai melacak serial dan lifetime."
                    : "Komponen yang dipasang planner/admin akan muncul di sini."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          </CardContent>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableCaption className="sr-only">
                Komponen aktif unit {unit.code}
              </TableCaption>
              <TableHeader>
                <TableRow>
                  <TableHead>Komponen</TableHead>
                  <TableHead>Serial</TableHead>
                  <TableHead>Terpasang</TableHead>
                  <TableHead>Pemakaian</TableHead>
                  <TableHead>Lifetime</TableHead>
                  <TableHead>Status</TableHead>
                  {canWrite ? <TableHead className="text-right">Aksi</TableHead> : null}
                </TableRow>
              </TableHeader>
              <TableBody>
                {activeItems.map((item) => (
                  <TableRow key={item.installation_id}>
                    <TableCell className="min-w-44">
                      <div className="flex flex-wrap items-center gap-1">
                        <p className="font-medium">{item.component_name}</p>
                        {item.component_class ? (
                          <Badge variant="outline" className="h-5 px-2 capitalize">
                            {item.component_class}
                          </Badge>
                        ) : null}
                      </div>
                      <p className="text-xs text-muted-foreground">{item.component_code}</p>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{item.serial_number}</TableCell>
                    <TableCell className="min-w-40 whitespace-nowrap text-sm">
                      <p>{formatDateTimeWita(item.installed_at)}</p>
                      <p className="text-xs text-muted-foreground">
                        HM {formatNumber(item.installed_hm)} · KM {formatNumber(item.installed_km)}
                      </p>
                    </TableCell>
                    <TableCell className="min-w-40 whitespace-nowrap text-sm">
                      <p>HM {formatNumber(item.used_hm)} · KM {formatNumber(item.used_km)}</p>
                      <p className="text-xs text-muted-foreground">
                        {formatNumber(item.used_days)} hari
                      </p>
                    </TableCell>
                    <TableCell className="min-w-52 whitespace-nowrap text-sm">
                      <p>
                        HM {item.lifetime_hm !== undefined ? formatNumber(item.lifetime_hm) : "-"}
                        {" · "}sisa {item.remaining_hm !== undefined ? formatNumber(item.remaining_hm) : "-"}
                      </p>
                      <p>
                        KM {item.lifetime_km !== undefined ? formatNumber(item.lifetime_km) : "-"}
                        {" · "}sisa {item.remaining_km !== undefined ? formatNumber(item.remaining_km) : "-"}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {item.lifetime_days !== undefined ? `${formatNumber(item.lifetime_days)} hari` : "Hari -"}
                        {" · "}sisa {item.remaining_days !== undefined ? formatNumber(item.remaining_days) : "-"}
                      </p>
                    </TableCell>
                    <TableCell>
                      <LifetimeStateBadge state={item.lifetime_state} />
                    </TableCell>
                    {canWrite ? (
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            type="button"
                            size="xs"
                            variant="outline"
                            onClick={() => setReplaceTarget(item)}
                          >
                            <Repeat2 aria-hidden="true" /> Ganti
                          </Button>
                          <Button
                            type="button"
                            size="xs"
                            variant="outline"
                            onClick={() => setTransferTarget(item)}
                          >
                            <ArrowRightLeft aria-hidden="true" /> Transfer
                          </Button>
                        </div>
                      </TableCell>
                    ) : null}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </Card>

      <Card className="gap-0 overflow-hidden">
        <CardHeader className="border-b">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <CardTitle className="flex items-center gap-2">
                <History className="size-4 text-muted-foreground" aria-hidden="true" />
                Riwayat instalasi
              </CardTitle>
              <CardDescription>
                Komponen yang sudah dilepas, dipindah, atau di-scrap.
              </CardDescription>
            </div>
            <Badge variant="secondary" className="h-6 px-2.5">
              {historyItems.length} riwayat
            </Badge>
          </div>
        </CardHeader>

        {loading ? (
          <CardContent className="flex flex-col gap-2 py-6">
            {Array.from({ length: 3 }).map((_, index) => (
              <Skeleton key={index} className="h-12 w-full" />
            ))}
          </CardContent>
        ) : historyItems.length === 0 ? (
          <CardContent className="py-8">
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <History aria-hidden="true" />
                </EmptyMedia>
                <EmptyTitle>Belum ada riwayat instalasi.</EmptyTitle>
                <EmptyDescription>
                  Replace dan transfer akan menyimpan histori di sini.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          </CardContent>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableCaption className="sr-only">
                Riwayat instalasi komponen unit {unit.code}
              </TableCaption>
              <TableHeader>
                <TableRow>
                  <TableHead>Komponen</TableHead>
                  <TableHead>Serial</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Terpasang</TableHead>
                  <TableHead>Dilepas</TableHead>
                  <TableHead>Pemakaian</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {historyItems.map((item) => (
                  <TableRow key={item.installation_id}>
                    <TableCell className="min-w-44">
                      <p className="font-medium">{item.component_name}</p>
                      <p className="text-xs text-muted-foreground">{item.component_code}</p>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{item.serial_number}</TableCell>
                    <TableCell>
                      <InstallationStatusBadge status={item.status} />
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-sm">
                      {formatDateTimeWita(item.installed_at)}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-sm">
                      {item.removed_at ? formatDateTimeWita(item.removed_at) : "-"}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-sm">
                      HM {formatNumber(item.used_hm)} · KM {formatNumber(item.used_km)}
                      <p className="text-xs text-muted-foreground">
                        {formatNumber(item.used_days)} hari
                      </p>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </Card>

      {installOpen ? (
        <InstallComponentSheet
          unit={unit}
          components={components}
          onOpenChange={setInstallOpen}
          onSaved={handleSaved}
        />
      ) : null}
      {replaceTarget ? (
        <ReplaceComponentSheet
          key={replaceTarget.installation_id}
          unit={unit}
          installation={replaceTarget}
          components={components}
          onOpenChange={(open) => {
            if (!open) setReplaceTarget(null);
          }}
          onSaved={handleSaved}
        />
      ) : null}
      {transferTarget ? (
        <TransferComponentSheet
          key={transferTarget.installation_id}
          unit={unit}
          installation={transferTarget}
          units={units}
          unitsLoading={unitsLoading}
          unitsError={unitsError}
          onOpenChange={(open) => {
            if (!open) setTransferTarget(null);
          }}
          onSaved={handleSaved}
        />
      ) : null}
    </div>
  );
}

function InstallComponentSheet({
  unit,
  components,
  onOpenChange,
  onSaved,
}: {
  unit: Unit;
  components: ComponentOption[];
  onOpenChange: (open: boolean) => void;
  onSaved: (message: string) => void;
}) {
  const [componentId, setComponentId] = useState("");
  const [serialNumber, setSerialNumber] = useState("");
  const [installedAt, setInstalledAt] = useState(() => toDatetimeLocalWita());
  const [installedHM, setInstalledHM] = useState("");
  const [installedKM, setInstalledKM] = useState("");
  const [lifetimeHM, setLifetimeHM] = useState("");
  const [lifetimeDays, setLifetimeDays] = useState("");
  const [lifetimeKM, setLifetimeKM] = useState("");
  const [notes, setNotes] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const selectComponent = (value: string) => {
    setComponentId(value);
    const component = components.find((item) => String(item.id) === value);
    setLifetimeHM(component?.default_lifetime_hm?.toString() ?? "");
    setLifetimeDays(component?.default_lifetime_days?.toString() ?? "");
    setLifetimeKM(component?.default_lifetime_km?.toString() ?? "");
    setFieldErrors((current) => {
      if (!current.component_id) return current;
      const next = { ...current };
      delete next.component_id;
      return next;
    });
  };

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    const nextErrors: Record<string, string> = {};
    if (!componentId) nextErrors.component_id = "Pilih komponen";
    if (!serialNumber.trim()) nextErrors.serial_number = "Serial number wajib diisi";
    if (!installedAt) nextErrors.installed_at = "Tanggal pemasangan wajib diisi";
    const hm = parseOptionalNonNegative(installedHM, "installed_hm", nextErrors);
    const km = parseOptionalNonNegative(installedKM, "installed_km", nextErrors);
    const lifeHM = parseOptionalNonNegative(lifetimeHM, "lifetime_hm", nextErrors);
    const lifeKM = parseOptionalNonNegative(lifetimeKM, "lifetime_km", nextErrors);
    const lifeDays = parseOptionalNonNegativeInt(lifetimeDays, "lifetime_days", nextErrors);
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setPending(true);
    try {
      await proxyPost<DataResponse<ComponentInstallation>>(
        `/units/${unit.id}/component-installations`,
        {
          component_id: Number(componentId),
          serial_number: serialNumber.trim(),
          installed_at: installedAt ? witaLocalToISOString(installedAt) : undefined,
          installed_hm: hm,
          installed_km: km,
          lifetime_hm: lifeHM,
          lifetime_days: lifeDays,
          lifetime_km: lifeKM,
          notes: notes.trim(),
        },
      );
      onSaved("Komponen berhasil dipasang.");
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
      } else {
        setError("Komponen tidak dapat dipasang.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Sheet open onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Pasang komponen</SheetTitle>
          <SheetDescription>
            Daftarkan komponen fisik pada unit {unit.code}. HM/KM kosong akan memakai reading unit
            saat ini.
          </SheetDescription>
        </SheetHeader>

        {components.length === 0 ? (
          <div className="px-4">
            <Alert>
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Master komponen belum tersedia</AlertTitle>
              <AlertDescription>
                Tambahkan master komponen terlebih dahulu agar komponen dapat dipasang.
              </AlertDescription>
            </Alert>
          </div>
        ) : null}

        <form className="grid gap-4 px-4" onSubmit={(event) => void submit(event)} noValidate>
          <Field data-invalid={Boolean(fieldErrors.component_id)}>
            <FieldLabel htmlFor="installation-component">
              Komponen <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <NativeSelect
              id="installation-component"
              value={componentId}
              onChange={(event) => selectComponent(event.target.value)}
            >
              <NativeSelectOption value="">Pilih komponen</NativeSelectOption>
              {components.map((component) => (
                <NativeSelectOption key={component.id} value={String(component.id)}>
                  {component.code} — {component.name}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {fieldErrors.component_id ? (
              <FieldError>{fieldErrors.component_id}</FieldError>
            ) : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.serial_number)}>
            <FieldLabel htmlFor="installation-serial">
              Serial number <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="installation-serial"
              value={serialNumber}
              onChange={(event) => setSerialNumber(event.target.value)}
              placeholder="Serial fisik komponen"
            />
            {fieldErrors.serial_number ? (
              <FieldError>{fieldErrors.serial_number}</FieldError>
            ) : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.installed_at)}>
            <FieldLabel htmlFor="installation-date">
              Tanggal pasang (WITA) <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="installation-date"
              type="datetime-local"
              value={installedAt}
              onChange={(event) => setInstalledAt(event.target.value)}
            />
            {fieldErrors.installed_at ? <FieldError>{fieldErrors.installed_at}</FieldError> : null}
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.installed_hm)}>
              <FieldLabel htmlFor="installation-hm">HM pasang</FieldLabel>
              <Input
                id="installation-hm"
                type="number"
                min="0"
                step="0.1"
                value={installedHM}
                onChange={(event) => setInstalledHM(event.target.value)}
                placeholder={formatNumber(unit.current_hm)}
              />
              {fieldErrors.installed_hm ? <FieldError>{fieldErrors.installed_hm}</FieldError> : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.installed_km)}>
              <FieldLabel htmlFor="installation-km">KM pasang</FieldLabel>
              <Input
                id="installation-km"
                type="number"
                min="0"
                step="0.1"
                value={installedKM}
                onChange={(event) => setInstalledKM(event.target.value)}
                placeholder={formatNumber(unit.current_km)}
              />
              {fieldErrors.installed_km ? <FieldError>{fieldErrors.installed_km}</FieldError> : null}
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.lifetime_hm)}>
              <FieldLabel htmlFor="installation-lifetime-hm">Lifetime HM</FieldLabel>
              <Input
                id="installation-lifetime-hm"
                type="number"
                min="0"
                step="0.1"
                value={lifetimeHM}
                onChange={(event) => setLifetimeHM(event.target.value)}
              />
              {fieldErrors.lifetime_hm ? <FieldError>{fieldErrors.lifetime_hm}</FieldError> : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.lifetime_days)}>
              <FieldLabel htmlFor="installation-lifetime-days">Lifetime hari</FieldLabel>
              <Input
                id="installation-lifetime-days"
                type="number"
                min="0"
                step="1"
                value={lifetimeDays}
                onChange={(event) => setLifetimeDays(event.target.value)}
              />
              {fieldErrors.lifetime_days ? (
                <FieldError>{fieldErrors.lifetime_days}</FieldError>
              ) : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.lifetime_km)}>
              <FieldLabel htmlFor="installation-lifetime-km">Lifetime KM</FieldLabel>
              <Input
                id="installation-lifetime-km"
                type="number"
                min="0"
                step="0.1"
                value={lifetimeKM}
                onChange={(event) => setLifetimeKM(event.target.value)}
              />
              {fieldErrors.lifetime_km ? <FieldError>{fieldErrors.lifetime_km}</FieldError> : null}
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="installation-notes">Catatan</FieldLabel>
            <Textarea
              id="installation-notes"
              rows={3}
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
            />
            <FieldDescription>
              Lifetime kosong akan memakai default master komponen.
            </FieldDescription>
          </Field>

          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Komponen tidak dapat dipasang</AlertTitle>
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
            <Button type="submit" size="sm" disabled={pending || components.length === 0}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" /> Menyimpan...
                </>
              ) : (
                "Pasang komponen"
              )}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}

function ReplaceComponentSheet({
  unit,
  installation,
  components,
  onOpenChange,
  onSaved,
}: {
  unit: Unit;
  installation: ComponentLifetime;
  components: ComponentOption[];
  onOpenChange: (open: boolean) => void;
  onSaved: (message: string) => void;
}) {
  const component = components.find((item) => item.id === installation.component_id);
  const [serialNumber, setSerialNumber] = useState("");
  const [installedAt, setInstalledAt] = useState(() => toDatetimeLocalWita());
  const [installedHM, setInstalledHM] = useState("");
  const [installedKM, setInstalledKM] = useState("");
  const [lifetimeHM, setLifetimeHM] = useState(
    component?.default_lifetime_hm?.toString() ?? "",
  );
  const [lifetimeDays, setLifetimeDays] = useState(
    component?.default_lifetime_days?.toString() ?? "",
  );
  const [lifetimeKM, setLifetimeKM] = useState(
    component?.default_lifetime_km?.toString() ?? "",
  );
  const [notes, setNotes] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    const nextErrors: Record<string, string> = {};
    if (!serialNumber.trim()) nextErrors.serial_number = "Serial number wajib diisi";
    if (!installedAt) nextErrors.installed_at = "Tanggal pemasangan wajib diisi";
    const hm = parseOptionalNonNegative(installedHM, "installed_hm", nextErrors);
    const km = parseOptionalNonNegative(installedKM, "installed_km", nextErrors);
    const lifeHM = parseOptionalNonNegative(lifetimeHM, "lifetime_hm", nextErrors);
    const lifeKM = parseOptionalNonNegative(lifetimeKM, "lifetime_km", nextErrors);
    const lifeDays = parseOptionalNonNegativeInt(lifetimeDays, "lifetime_days", nextErrors);
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setPending(true);
    try {
      await proxyPost<DataResponse<ComponentInstallation>>(
        `/units/${unit.id}/component-installations/${installation.installation_id}/replace`,
        {
          serial_number: serialNumber.trim(),
          installed_at: installedAt ? witaLocalToISOString(installedAt) : undefined,
          installed_hm: hm,
          installed_km: km,
          lifetime_hm: lifeHM,
          lifetime_days: lifeDays,
          lifetime_km: lifeKM,
          notes: notes.trim(),
        },
      );
      onSaved("Komponen berhasil diganti.");
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
      } else {
        setError("Komponen tidak dapat diganti.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Sheet open onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Ganti komponen</SheetTitle>
          <SheetDescription>
            {installation.component_name} ({installation.serial_number}) akan ditutup dan diganti
            pada unit {unit.code}.
          </SheetDescription>
        </SheetHeader>

        <form className="grid gap-4 px-4" onSubmit={(event) => void submit(event)} noValidate>
          <Field data-invalid={Boolean(fieldErrors.serial_number)}>
            <FieldLabel htmlFor="replace-serial">
              Serial number baru <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="replace-serial"
              value={serialNumber}
              onChange={(event) => setSerialNumber(event.target.value)}
              placeholder="Serial fisik komponen pengganti"
            />
            {fieldErrors.serial_number ? (
              <FieldError>{fieldErrors.serial_number}</FieldError>
            ) : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.installed_at)}>
            <FieldLabel htmlFor="replace-date">
              Tanggal pasang (WITA) <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="replace-date"
              type="datetime-local"
              value={installedAt}
              onChange={(event) => setInstalledAt(event.target.value)}
            />
            {fieldErrors.installed_at ? <FieldError>{fieldErrors.installed_at}</FieldError> : null}
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.installed_hm)}>
              <FieldLabel htmlFor="replace-hm">HM pasang</FieldLabel>
              <Input
                id="replace-hm"
                type="number"
                min="0"
                step="0.1"
                value={installedHM}
                onChange={(event) => setInstalledHM(event.target.value)}
                placeholder={formatNumber(unit.current_hm)}
              />
              {fieldErrors.installed_hm ? <FieldError>{fieldErrors.installed_hm}</FieldError> : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.installed_km)}>
              <FieldLabel htmlFor="replace-km">KM pasang</FieldLabel>
              <Input
                id="replace-km"
                type="number"
                min="0"
                step="0.1"
                value={installedKM}
                onChange={(event) => setInstalledKM(event.target.value)}
                placeholder={formatNumber(unit.current_km)}
              />
              {fieldErrors.installed_km ? <FieldError>{fieldErrors.installed_km}</FieldError> : null}
            </Field>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.lifetime_hm)}>
              <FieldLabel htmlFor="replace-lifetime-hm">Lifetime HM</FieldLabel>
              <Input
                id="replace-lifetime-hm"
                type="number"
                min="0"
                step="0.1"
                value={lifetimeHM}
                onChange={(event) => setLifetimeHM(event.target.value)}
              />
              {fieldErrors.lifetime_hm ? <FieldError>{fieldErrors.lifetime_hm}</FieldError> : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.lifetime_days)}>
              <FieldLabel htmlFor="replace-lifetime-days">Lifetime hari</FieldLabel>
              <Input
                id="replace-lifetime-days"
                type="number"
                min="0"
                step="1"
                value={lifetimeDays}
                onChange={(event) => setLifetimeDays(event.target.value)}
              />
              {fieldErrors.lifetime_days ? (
                <FieldError>{fieldErrors.lifetime_days}</FieldError>
              ) : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.lifetime_km)}>
              <FieldLabel htmlFor="replace-lifetime-km">Lifetime KM</FieldLabel>
              <Input
                id="replace-lifetime-km"
                type="number"
                min="0"
                step="0.1"
                value={lifetimeKM}
                onChange={(event) => setLifetimeKM(event.target.value)}
              />
              {fieldErrors.lifetime_km ? <FieldError>{fieldErrors.lifetime_km}</FieldError> : null}
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="replace-notes">Catatan</FieldLabel>
            <Textarea
              id="replace-notes"
              rows={3}
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
            />
            <FieldDescription>
              Pelepasan komponen lama memakai reading unit saat ini; isi HM/KM pasang bila berbeda.
            </FieldDescription>
          </Field>

          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Komponen tidak dapat diganti</AlertTitle>
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
            <Button type="submit" size="sm" disabled={pending}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" /> Menyimpan...
                </>
              ) : (
                "Ganti komponen"
              )}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}

function TransferComponentSheet({
  unit,
  installation,
  units,
  unitsLoading,
  unitsError,
  onOpenChange,
  onSaved,
}: {
  unit: Unit;
  installation: ComponentLifetime;
  units: Unit[];
  unitsLoading: boolean;
  unitsError: string;
  onOpenChange: (open: boolean) => void;
  onSaved: (message: string) => void;
}) {
  const [toUnitId, setToUnitId] = useState("");
  const [installedAt, setInstalledAt] = useState(() => toDatetimeLocalWita());
  const [installedHM, setInstalledHM] = useState("");
  const [installedKM, setInstalledKM] = useState("");
  const [notes, setNotes] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const targetUnits = units.filter((item) => item.id !== unit.id);
  const target = targetUnits.find((item) => String(item.id) === toUnitId);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    const nextErrors: Record<string, string> = {};
    if (!toUnitId) nextErrors.to_unit_id = "Pilih unit tujuan";
    if (!installedAt) nextErrors.installed_at = "Tanggal pemasangan wajib diisi";
    const hm = parseOptionalNonNegative(installedHM, "installed_hm", nextErrors);
    const km = parseOptionalNonNegative(installedKM, "installed_km", nextErrors);
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    setPending(true);
    try {
      await proxyPost<DataResponse<ComponentInstallation>>(
        `/units/${unit.id}/component-installations/${installation.installation_id}/transfer`,
        {
          to_unit_id: Number(toUnitId),
          installed_at: installedAt ? witaLocalToISOString(installedAt) : undefined,
          installed_hm: hm,
          installed_km: km,
          notes: notes.trim(),
        },
      );
      onSaved("Komponen berhasil ditransfer.");
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFieldErrors(submitError.fields);
        setError(submitError.message);
      } else {
        setError("Komponen tidak dapat ditransfer.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Sheet open onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Transfer komponen</SheetTitle>
          <SheetDescription>
            Pindahkan {installation.component_name} ({installation.serial_number}) dari unit{" "}
            {unit.code}. Serial dan lifetime instalasi akan dipertahankan.
          </SheetDescription>
        </SheetHeader>

        <form className="grid gap-4 px-4" onSubmit={(event) => void submit(event)} noValidate>
          <Field data-invalid={Boolean(fieldErrors.to_unit_id)}>
            <FieldLabel htmlFor="transfer-unit">
              Unit tujuan <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <NativeSelect
              id="transfer-unit"
              value={toUnitId}
              onChange={(event) => setToUnitId(event.target.value)}
              disabled={unitsLoading}
            >
              <NativeSelectOption value="">
                {unitsLoading ? "Memuat unit..." : "Pilih unit tujuan"}
              </NativeSelectOption>
              {targetUnits.map((item) => (
                <NativeSelectOption key={item.id} value={String(item.id)}>
                  {item.code} — {item.unit_type || "unit"}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {fieldErrors.to_unit_id ? <FieldError>{fieldErrors.to_unit_id}</FieldError> : null}
            {unitsError ? <FieldError>{unitsError}</FieldError> : null}
            {target ? (
              <FieldDescription>
                HM target {formatNumber(target.current_hm)} · KM target {formatNumber(target.current_km)}
              </FieldDescription>
            ) : null}
          </Field>

          <Field data-invalid={Boolean(fieldErrors.installed_at)}>
            <FieldLabel htmlFor="transfer-date">
              Tanggal pasang di unit tujuan (WITA){" "}
              <span aria-hidden="true" className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="transfer-date"
              type="datetime-local"
              value={installedAt}
              onChange={(event) => setInstalledAt(event.target.value)}
            />
            {fieldErrors.installed_at ? <FieldError>{fieldErrors.installed_at}</FieldError> : null}
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field data-invalid={Boolean(fieldErrors.installed_hm)}>
              <FieldLabel htmlFor="transfer-hm">HM pasang</FieldLabel>
              <Input
                id="transfer-hm"
                type="number"
                min="0"
                step="0.1"
                value={installedHM}
                onChange={(event) => setInstalledHM(event.target.value)}
                placeholder={target ? formatNumber(target.current_hm) : "HM unit tujuan"}
              />
              {fieldErrors.installed_hm ? <FieldError>{fieldErrors.installed_hm}</FieldError> : null}
            </Field>
            <Field data-invalid={Boolean(fieldErrors.installed_km)}>
              <FieldLabel htmlFor="transfer-km">KM pasang</FieldLabel>
              <Input
                id="transfer-km"
                type="number"
                min="0"
                step="0.1"
                value={installedKM}
                onChange={(event) => setInstalledKM(event.target.value)}
                placeholder={target ? formatNumber(target.current_km) : "KM unit tujuan"}
              />
              {fieldErrors.installed_km ? <FieldError>{fieldErrors.installed_km}</FieldError> : null}
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="transfer-notes">Catatan</FieldLabel>
            <Textarea
              id="transfer-notes"
              rows={3}
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
            />
            <FieldDescription>
              HM/KM kosong akan memakai reading unit tujuan saat ini.
            </FieldDescription>
          </Field>

          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Komponen tidak dapat ditransfer</AlertTitle>
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
            <Button type="submit" size="sm" disabled={pending || unitsLoading || targetUnits.length === 0}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" /> Menyimpan...
                </>
              ) : (
                "Transfer komponen"
              )}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}

function parseOptionalNonNegative(
  value: string,
  field: string,
  errors: Record<string, string>,
): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || parsed < 0) {
    errors[field] = "Nilai harus angka dan tidak negatif";
    return undefined;
  }
  return parsed;
}

function parseOptionalNonNegativeInt(
  value: string,
  field: string,
  errors: Record<string, string>,
): number | undefined {
  const parsed = parseOptionalNonNegative(value, field, errors);
  if (parsed === undefined) return undefined;
  if (!Number.isInteger(parsed)) {
    errors[field] = "Nilai harus bilangan bulat";
    return undefined;
  }
  return parsed;
}
