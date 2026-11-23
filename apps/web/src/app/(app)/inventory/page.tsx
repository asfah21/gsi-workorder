import type { Metadata } from "next";

import { InventoryView } from "@/components/inventory-view";

export const metadata: Metadata = {
  title: "Inventory | PAMS",
  description: "Pantau stok, transaksi, dan pergerakan inventory GSI.",
};

export default function InventoryPage() {
  return <InventoryView />;
}
