import type { Metadata } from "next";

import { DashboardView } from "@/components/dashboard-view";

export const metadata: Metadata = {
  title: "Dashboard | PAMS",
  description: "Ringkasan unit, maintenance, dan operasional gudang",
};

export default function DashboardPage() {
  return <DashboardView />;
}
