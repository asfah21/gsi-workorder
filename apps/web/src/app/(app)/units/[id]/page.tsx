import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { UnitDetailView } from "@/components/unit-detail-view";

type PageProps = {
  params: Promise<{ id: string }>;
};

export const metadata: Metadata = {
  title: "Detail Unit | GSI Work Order",
};

export default async function UnitDetailPage({ params }: PageProps) {
  const { id } = await params;
  const unitId = Number(id);
  if (!Number.isInteger(unitId) || unitId <= 0) {
    notFound();
  }
  return <UnitDetailView unitId={unitId} />;
}
