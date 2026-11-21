"use client";

import { ArrowLeft, ShieldCheck } from "lucide-react";
import Link from "next/link";

import { useSession } from "@/components/app-shell";
import { UnitForm } from "@/components/unit-form";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardContent } from "@/components/ui/card";

export function UnitCreateView() {
  const { user } = useSession();
  const canWrite = user.role === "admin" || user.role === "planner";

  if (!canWrite) {
    return (
      <div className="max-w-xl space-y-4">
        <Card>
          <CardContent>
            <h1 className="text-xl font-semibold">Tambah unit</h1>
          </CardContent>
        </Card>
        <Alert className="border-sky-500/30 bg-sky-500/5">
          <ShieldCheck aria-hidden="true" />
          <AlertTitle>Akses baca saja</AlertTitle>
          <AlertDescription>Role Anda hanya dapat membaca master unit.</AlertDescription>
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

  return (
    <div className="space-y-4">
      <Card>
        <CardContent>
          <Link
            href="/units"
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            <ArrowLeft className="size-4" aria-hidden="true" />
            Unit
          </Link>
          <h1 className="mt-2 text-xl font-semibold">Tambah unit</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            HM/KM awal diisi melalui pembacaan setelah unit dibuat.
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent>
          <UnitForm />
        </CardContent>
      </Card>
    </div>
  );
}
