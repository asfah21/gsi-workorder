import type { Metadata } from "next";

import { ReportsView } from "@/components/reports-view";

export const metadata: Metadata = {
  title: "Laporan | PAMS",
  description: "Laporan PM, shortage, sparepart, inventory, dan komponen",
};

export default function ReportsPage() {
  return <ReportsView />;
}
