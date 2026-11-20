"use client";

import {
  Boxes,
  CircleAlert,
  CircleCheck,
  Loader2,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ActiveBadge } from "@/components/status-badges";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  EngineModel,
  ListResponse,
  UnitModel,
} from "@/lib/types";
import { formatDateTimeWita } from "@/lib/time";

export function MasterView() {
  const { user } = useSession();
  const [unitModels, setUnitModels] = useState<UnitModel[]>([]);
  const [engineModels, setEngineModels] = useState<EngineModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const canWrite = user.role === "admin" || user.role === "planner";

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [units, engines] = await Promise.all([
        proxyGet<ListResponse<UnitModel>>("/unit-models?limit=100"),
        proxyGet<ListResponse<EngineModel>>("/engine-models?limit=100"),
      ]);
      setUnitModels(units.data);
      setEngineModels(engines.data);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Master model tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">Master unit</h1>
        <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
          Model unit dan engine yang dipakai pada master unit.
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          <Badge variant="secondary" className="h-6 gap-1.5 px-2.5">
            <Boxes className="size-3.5" aria-hidden="true" />
            {unitModels.length} unit model
          </Badge>
          <Badge variant="secondary" className="h-6 gap-1.5 px-2.5">
            <Settings2 className="size-3.5" aria-hidden="true" />
            {engineModels.length} engine model
          </Badge>
        </div>
      </div>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Master model tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {!canWrite ? (
        <Alert className="border-sky-500/30 bg-sky-500/5">
          <ShieldCheck aria-hidden="true" />
          <AlertTitle>Akses baca saja</AlertTitle>
          <AlertDescription>
            Role Anda dapat melihat master unit, tetapi tidak dapat menambah model.
          </AlertDescription>
        </Alert>
      ) : null}

      <section aria-labelledby="unit-model-heading" className="space-y-4">
        <div>
          <h2 id="unit-model-heading" className="text-lg font-semibold tracking-tight">
            Unit model
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Kombinasi manufacturer dan model harus unik.
          </p>
        </div>
        <div className="grid gap-4">
          {canWrite ? <UnitModelForm onSaved={() => void load()} /> : null}
          <Card className="overflow-hidden shadow-sm">
            <CardHeader className="border-b">
              <CardTitle>Daftar unit model</CardTitle>
              <CardDescription>Referensi model untuk master unit.</CardDescription>
            </CardHeader>
            {loading ? (
              <TableSkeleton />
            ) : unitModels.length === 0 ? (
              <EmptyModels label="unit model" />
            ) : (
              <div
                className="responsive-table-wrapper"
                tabIndex={0}
                role="region"
                aria-labelledby="unit-model-heading"
              >
                <table className="responsive-table">
                  <caption className="sr-only">Daftar unit model</caption>
                  <thead>
                    <tr>
                      <th scope="col">Manufacturer</th>
                      <th scope="col">Model</th>
                      <th scope="col">Kategori</th>
                      <th scope="col">Status</th>
                      <th scope="col">Diperbarui</th>
                    </tr>
                  </thead>
                  <tbody>
                    {unitModels.map((model) => (
                      <tr key={model.id}>
                        <td data-label="Manufacturer">{model.manufacturer}</td>
                        <td data-label="Model" className="font-medium">{model.model}</td>
                        <td data-label="Kategori">{model.category || "-"}</td>
                        <td data-label="Status">
                          <ActiveBadge active={model.is_active} />
                        </td>
                        <td data-label="Diperbarui" className="text-muted-foreground">
                          {formatDateTimeWita(model.updated_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </div>
      </section>

      <section aria-labelledby="engine-model-heading" className="space-y-4">
        <div>
          <h2 id="engine-model-heading" className="text-lg font-semibold tracking-tight">
            Engine model
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Kombinasi manufacturer dan model harus unik.
          </p>
        </div>
        <div className="grid gap-4">
          {canWrite ? <EngineModelForm onSaved={() => void load()} /> : null}
          <Card className="overflow-hidden shadow-sm">
            <CardHeader className="border-b">
              <CardTitle>Daftar engine model</CardTitle>
              <CardDescription>Referensi engine untuk master unit.</CardDescription>
            </CardHeader>
            {loading ? (
              <TableSkeleton />
            ) : engineModels.length === 0 ? (
              <EmptyModels label="engine model" />
            ) : (
              <div
                className="responsive-table-wrapper"
                tabIndex={0}
                role="region"
                aria-labelledby="engine-model-heading"
              >
                <table className="responsive-table">
                  <caption className="sr-only">Daftar engine model</caption>
                  <thead>
                    <tr>
                      <th scope="col">Manufacturer</th>
                      <th scope="col">Model</th>
                      <th scope="col">Bahan bakar</th>
                      <th scope="col">Displacement</th>
                      <th scope="col">Status</th>
                      <th scope="col">Diperbarui</th>
                    </tr>
                  </thead>
                  <tbody>
                    {engineModels.map((model) => (
                      <tr key={model.id}>
                        <td data-label="Manufacturer">{model.manufacturer}</td>
                        <td data-label="Model" className="font-medium">{model.model}</td>
                        <td data-label="Bahan bakar">{model.fuel_type || "-"}</td>
                        <td data-label="Displacement">
                          {model.displacement_cc ? `${model.displacement_cc} cc` : "-"}
                        </td>
                        <td data-label="Status">
                          <ActiveBadge active={model.is_active} />
                        </td>
                        <td data-label="Diperbarui" className="text-muted-foreground">
                          {formatDateTimeWita(model.updated_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </div>
      </section>
    </div>
  );
}

function UnitModelForm({ onSaved }: { onSaved: () => void }) {
  const [manufacturer, setManufacturer] = useState("");
  const [model, setModel] = useState("");
  const [category, setCategory] = useState("");
  const [notes, setNotes] = useState("");
  const [active, setActive] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setPending(true);
    try {
      await proxyPost<DataResponse<UnitModel>>("/unit-models", {
        manufacturer: manufacturer.trim(),
        model: model.trim(),
        category: category.trim(),
        notes: notes.trim(),
        is_active: active,
      });
      setManufacturer("");
      setModel("");
      setCategory("");
      setNotes("");
      setActive(true);
      setMessage("Unit model berhasil ditambahkan.");
      onSaved();
    } catch (submitError) {
      setError(
        submitError instanceof ApiError
          ? submitError.message
          : "Unit model tidak dapat disimpan.",
      );
    } finally {
      setPending(false);
    }
  };

  return (
    <Card className="shadow-sm">
      <CardHeader className="border-b">
        <CardTitle>Tambah unit model</CardTitle>
        <CardDescription>Tambahkan referensi model unit baru.</CardDescription>
      </CardHeader>
      <CardContent className="pt-4">
        <form onSubmit={(event) => void submit(event)} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="space-y-2">
              <Label htmlFor="um-manufacturer">
                Manufacturer <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <Input
                id="um-manufacturer"
                name="manufacturer"
                required
                autoComplete="off"
                value={manufacturer}
                onChange={(event) => setManufacturer(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="um-model">
                Model <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <Input
                id="um-model"
                name="model"
                required
                autoComplete="off"
                value={model}
                onChange={(event) => setModel(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="um-category">Kategori</Label>
              <Input
                id="um-category"
                name="category"
                value={category}
                onChange={(event) => setCategory(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="um-notes">Catatan</Label>
              <Input
                id="um-notes"
                name="notes"
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
              />
            </div>
          </div>
          <label className="flex min-h-11 items-center gap-3 text-sm">
            <input
              type="checkbox"
              name="is_active"
              checked={active}
              onChange={(event) => setActive(event.target.checked)}
              className="size-4 accent-primary"
            />
            Aktif
          </label>
          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Gagal menyimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
          {message ? (
            <Alert className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300">
              <CircleCheck aria-hidden="true" />
              <AlertTitle>Berhasil</AlertTitle>
              <AlertDescription className="text-emerald-700 dark:text-emerald-300">
                {message}
              </AlertDescription>
            </Alert>
          ) : null}
          <Button type="submit" disabled={pending} className="h-11 px-4">
            {pending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                Menyimpan...
              </>
            ) : (
              "Tambah unit model"
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

function EngineModelForm({ onSaved }: { onSaved: () => void }) {
  const [manufacturer, setManufacturer] = useState("");
  const [model, setModel] = useState("");
  const [fuelType, setFuelType] = useState("");
  const [displacement, setDisplacement] = useState("");
  const [notes, setNotes] = useState("");
  const [active, setActive] = useState(true);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setPending(true);
    try {
      await proxyPost<DataResponse<EngineModel>>("/engine-models", {
        manufacturer: manufacturer.trim(),
        model: model.trim(),
        fuel_type: fuelType.trim(),
        displacement_cc: displacement ? Number(displacement) : null,
        notes: notes.trim(),
        is_active: active,
      });
      setManufacturer("");
      setModel("");
      setFuelType("");
      setDisplacement("");
      setNotes("");
      setActive(true);
      setMessage("Engine model berhasil ditambahkan.");
      onSaved();
    } catch (submitError) {
      setError(
        submitError instanceof ApiError
          ? submitError.message
          : "Engine model tidak dapat disimpan.",
      );
    } finally {
      setPending(false);
    }
  };

  return (
    <Card className="shadow-sm">
      <CardHeader className="border-b">
        <CardTitle>Tambah engine model</CardTitle>
        <CardDescription>Tambahkan referensi engine baru.</CardDescription>
      </CardHeader>
      <CardContent className="pt-4">
        <form onSubmit={(event) => void submit(event)} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="space-y-2">
              <Label htmlFor="em-manufacturer">
                Manufacturer <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <Input
                id="em-manufacturer"
                name="manufacturer"
                required
                autoComplete="off"
                value={manufacturer}
                onChange={(event) => setManufacturer(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="em-model">
                Model <span aria-hidden="true" className="text-destructive">*</span>
              </Label>
              <Input
                id="em-model"
                name="model"
                required
                autoComplete="off"
                value={model}
                onChange={(event) => setModel(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="em-fuel">Bahan bakar</Label>
              <Input
                id="em-fuel"
                name="fuel_type"
                value={fuelType}
                onChange={(event) => setFuelType(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="em-displacement">Displacement (cc)</Label>
              <Input
                id="em-displacement"
                name="displacement_cc"
                type="number"
                inputMode="numeric"
                min="1"
                step="1"
                value={displacement}
                onChange={(event) => setDisplacement(event.target.value)}
              />
            </div>
            <div className="space-y-2 sm:col-span-2 lg:col-span-4">
              <Label htmlFor="em-notes">Catatan</Label>
              <Input
                id="em-notes"
                name="notes"
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
              />
            </div>
          </div>
          <label className="flex min-h-11 items-center gap-3 text-sm">
            <input
              type="checkbox"
              name="is_active"
              checked={active}
              onChange={(event) => setActive(event.target.checked)}
              className="size-4 accent-primary"
            />
            Aktif
          </label>
          {error ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Gagal menyimpan</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
          {message ? (
            <Alert className="border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300">
              <CircleCheck aria-hidden="true" />
              <AlertTitle>Berhasil</AlertTitle>
              <AlertDescription className="text-emerald-700 dark:text-emerald-300">
                {message}
              </AlertDescription>
            </Alert>
          ) : null}
          <Button type="submit" disabled={pending} className="h-11 px-4">
            {pending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden="true" />
                Menyimpan...
              </>
            ) : (
              "Tambah engine model"
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}

function TableSkeleton() {
  return (
    <div className="space-y-2 p-4" role="status" aria-label="Memuat master model">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-11 w-full" />
      ))}
    </div>
  );
}

function EmptyModels({ label }: { label: string }) {
  return (
    <CardContent className="py-12 text-center">
      <span className="mx-auto grid size-12 place-items-center rounded-2xl bg-muted text-muted-foreground">
        <Boxes className="size-6" aria-hidden="true" />
      </span>
      <p className="mt-4 font-medium">Belum ada {label}.</p>
      <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
        Tambahkan referensi model agar dapat dipilih saat membuat unit.
      </p>
    </CardContent>
  );
}
