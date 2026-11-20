"use client";

import { useCallback, useEffect, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  EngineModel,
  ListResponse,
  UnitModel,
} from "@/lib/types";
import { formatDateTimeWita } from "@/lib/time";

const inputClass =
  "mt-1 min-h-11 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30";

export function MasterView() {
  const { user } = useSession();
  const [unitModels, setUnitModels] = useState<UnitModel[]>([]);
  const [engineModels, setEngineModels] = useState<EngineModel[]>([]);
  const [error, setError] = useState("");
  const canWrite = user.role === "admin" || user.role === "planner";

  const load = useCallback(async () => {
    setError("");
    try {
      const [units, engines] = await Promise.all([
        proxyGet<ListResponse<UnitModel>>("/unit-models?limit=100"),
        proxyGet<ListResponse<EngineModel>>("/engine-models?limit=100"),
      ]);
      setUnitModels(units.data);
      setEngineModels(engines.data);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Master model tidak dapat dimuat.");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Master unit</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Model unit dan engine yang dipakai pada master unit.
        </p>
      </div>

      {error ? (
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <section aria-labelledby="unit-model-heading" className="space-y-4">
        <div>
          <h2 id="unit-model-heading" className="text-lg font-semibold">Unit model</h2>
          <p className="mt-1 text-sm text-muted-foreground">Kombinasi manufacturer dan model harus unik.</p>
        </div>
        {canWrite ? <UnitModelForm onSaved={() => void load()} /> : null}
        <div className="overflow-hidden rounded-xl border border-border bg-background">
          <div className="responsive-table-wrapper" tabIndex={0} role="region" aria-labelledby="unit-model-heading">
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
                    <td data-label="Model">{model.model}</td>
                    <td data-label="Kategori">{model.category || "-"}</td>
                    <td data-label="Status"><ActiveBadge active={model.is_active} /></td>
                    <td data-label="Diperbarui">{formatDateTimeWita(model.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {unitModels.length === 0 ? <p className="px-4 py-8 text-center text-sm text-muted-foreground">Belum ada unit model.</p> : null}
        </div>
      </section>

      <section aria-labelledby="engine-model-heading" className="space-y-4">
        <div>
          <h2 id="engine-model-heading" className="text-lg font-semibold">Engine model</h2>
          <p className="mt-1 text-sm text-muted-foreground">Kombinasi manufacturer dan model harus unik.</p>
        </div>
        {canWrite ? <EngineModelForm onSaved={() => void load()} /> : null}
        <div className="overflow-hidden rounded-xl border border-border bg-background">
          <div className="responsive-table-wrapper" tabIndex={0} role="region" aria-labelledby="engine-model-heading">
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
                    <td data-label="Model">{model.model}</td>
                    <td data-label="Bahan bakar">{model.fuel_type || "-"}</td>
                    <td data-label="Displacement">{model.displacement_cc ? `${model.displacement_cc} cc` : "-"}</td>
                    <td data-label="Status"><ActiveBadge active={model.is_active} /></td>
                    <td data-label="Diperbarui">{formatDateTimeWita(model.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {engineModels.length === 0 ? <p className="px-4 py-8 text-center text-sm text-muted-foreground">Belum ada engine model.</p> : null}
        </div>
      </section>

      {!canWrite ? (
        <p className="text-sm text-muted-foreground">Role Anda hanya dapat membaca master unit.</p>
      ) : null}
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
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
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
      onSaved();
    } catch (submitError) {
      setError(submitError instanceof ApiError ? submitError.message : "Unit model tidak dapat disimpan.");
    } finally {
      setPending(false);
    }
  };

  return (
    <form onSubmit={(event) => void submit(event)} className="rounded-xl border border-border bg-background p-5" noValidate>
      <h3 className="text-base font-semibold">Tambah unit model</h3>
      <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <label htmlFor="um-manufacturer" className="text-sm font-medium">Manufacturer</label>
          <input id="um-manufacturer" required value={manufacturer} onChange={(event) => setManufacturer(event.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="um-model" className="text-sm font-medium">Model</label>
          <input id="um-model" required value={model} onChange={(event) => setModel(event.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="um-category" className="text-sm font-medium">Kategori</label>
          <input id="um-category" value={category} onChange={(event) => setCategory(event.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="um-notes" className="text-sm font-medium">Catatan</label>
          <input id="um-notes" value={notes} onChange={(event) => setNotes(event.target.value)} className={inputClass} />
        </div>
      </div>
      <label className="mt-4 flex min-h-11 items-center gap-3 text-sm">
        <input type="checkbox" checked={active} onChange={(event) => setActive(event.target.checked)} className="size-4 accent-primary" />
        Aktif
      </label>
      {error ? <p role="alert" className="mt-3 text-sm text-destructive">{error}</p> : null}
      <button type="submit" disabled={pending} className="mt-4 min-h-11 rounded-md bg-primary px-4 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-60">
        {pending ? "Menyimpan..." : "Tambah unit model"}
      </button>
    </form>
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
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
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
      onSaved();
    } catch (submitError) {
      setError(submitError instanceof ApiError ? submitError.message : "Engine model tidak dapat disimpan.");
    } finally {
      setPending(false);
    }
  };

  return (
    <form onSubmit={(event) => void submit(event)} className="rounded-xl border border-border bg-background p-5" noValidate>
      <h3 className="text-base font-semibold">Tambah engine model</h3>
      <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <label htmlFor="em-manufacturer" className="text-sm font-medium">Manufacturer</label>
          <input id="em-manufacturer" required value={manufacturer} onChange={(event) => setManufacturer(event.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="em-model" className="text-sm font-medium">Model</label>
          <input id="em-model" required value={model} onChange={(event) => setModel(event.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="em-fuel" className="text-sm font-medium">Bahan bakar</label>
          <input id="em-fuel" value={fuelType} onChange={(event) => setFuelType(event.target.value)} className={inputClass} />
        </div>
        <div>
          <label htmlFor="em-displacement" className="text-sm font-medium">Displacement (cc)</label>
          <input id="em-displacement" type="number" min="1" step="1" value={displacement} onChange={(event) => setDisplacement(event.target.value)} className={inputClass} />
        </div>
        <div className="sm:col-span-2 lg:col-span-4">
          <label htmlFor="em-notes" className="text-sm font-medium">Catatan</label>
          <input id="em-notes" value={notes} onChange={(event) => setNotes(event.target.value)} className={inputClass} />
        </div>
      </div>
      <label className="mt-4 flex min-h-11 items-center gap-3 text-sm">
        <input type="checkbox" checked={active} onChange={(event) => setActive(event.target.checked)} className="size-4 accent-primary" />
        Aktif
      </label>
      {error ? <p role="alert" className="mt-3 text-sm text-destructive">{error}</p> : null}
      <button type="submit" disabled={pending} className="mt-4 min-h-11 rounded-md bg-primary px-4 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-60">
        {pending ? "Menyimpan..." : "Tambah engine model"}
      </button>
    </form>
  );
}

function ActiveBadge({ active }: { active: boolean }) {
  return (
    <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${active ? "bg-emerald-100 text-emerald-800" : "bg-muted text-muted-foreground"}`}>
      {active ? "Aktif" : "Nonaktif"}
    </span>
  );
}
