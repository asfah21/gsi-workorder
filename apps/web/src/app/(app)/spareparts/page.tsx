import type { Metadata } from "next";

import { SparepartsView } from "@/components/spareparts-view";

export const metadata: Metadata = {
  title: "Sparepart | PAMS",
  description: "Kelola master sparepart dan komponen GSI.",
};

export default function SparepartsPage() {
  return <SparepartsView />;
}
