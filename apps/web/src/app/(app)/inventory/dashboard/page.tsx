import type { Metadata } from "next";

import { InventoryDashboardView } from "@/components/inventory-dashboard-view";

export const metadata: Metadata = {
  title: "Dashboard Inventory | PAMS",
  description: "Ringkasan stok dan pergerakan inventory GSI.",
};

export default function InventoryDashboardPage() {
  return <InventoryDashboardView />;
}
