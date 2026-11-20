import type { Metadata } from "next";

import { UnitCreateView } from "@/components/unit-create-view";

export const metadata: Metadata = {
  title: "Tambah Unit | GSI Work Order",
};

export default function NewUnitPage() {
  return <UnitCreateView />;
}
