import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { UnitEditView } from "@/components/unit-edit-view";

type PageProps = {
  params: Promise<{ id: string }>;
};

export const metadata: Metadata = {
  title: "Edit Unit | GSI Work Order",
};

export default async function EditUnitPage({ params }: PageProps) {
  const { id } = await params;
  const unitId = Number(id);
  if (!Number.isInteger(unitId) || unitId <= 0) {
    notFound();
  }
  return <UnitEditView unitId={unitId} />;
}
