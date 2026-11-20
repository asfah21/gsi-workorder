"use client";

import Link from "next/link";

import { useSession } from "@/components/app-shell";
import { UnitForm } from "@/components/unit-form";

export function UnitCreateView() {
  const { user } = useSession();
  const canWrite = user.role === "admin" || user.role === "planner";

  if (!canWrite) {
    return (
      <div className="max-w-xl space-y-3">
        <h1 className="text-2xl font-semibold tracking-tight">Tambah unit</h1>
        <p role="alert" className="rounded-lg border border-border bg-background px-4 py-3 text-sm">
          Role Anda hanya dapat membaca master unit.
        </p>
        <Link href="/units" className="text-sm font-medium underline underline-offset-4">
          Kembali ke daftar unit
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Tambah unit</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          HM/KM awal diisi melalui pembacaan setelah unit dibuat.
        </p>
      </div>
      <div className="rounded-xl border border-border bg-background p-5 sm:p-6">
        <UnitForm />
      </div>
    </div>
  );
}
