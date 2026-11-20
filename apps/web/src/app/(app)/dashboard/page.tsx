import type { Metadata } from "next";

import { DashboardView } from "@/components/dashboard-view";

export const metadata: Metadata = {
  title: "Dashboard | GSI Work Order",
  description: "Ringkasan unit dan pembacaan HM/KM",
};

export default function DashboardPage() {
  return <DashboardView />;
}
