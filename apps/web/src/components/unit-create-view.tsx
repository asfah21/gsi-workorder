"use client";

import { ArrowLeft, ShieldCheck } from "lucide-react";
import Link from "next/link";

import { useSession } from "@/components/app-shell";
import { PageHeader } from "@/components/layout/page-header";
import { UnitForm } from "@/components/unit-form";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardContent } from "@/components/ui/card";

export function UnitCreateView() {
  const { user } = useSession();
  const canWrite = user.role === "admin" || user.role === "planner";

  if (!canWrite) {
    return (
      <div className="flex max-w-xl flex-col gap-4">
        <PageHeader title="Tambah unit" />
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
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Tambah unit"
        description="HM/KM awal diisi melalui pembacaan setelah unit dibuat."
      />
      <Card>
        <CardContent>
          <UnitForm />
        </CardContent>
      </Card>
    </div>
  );
}
