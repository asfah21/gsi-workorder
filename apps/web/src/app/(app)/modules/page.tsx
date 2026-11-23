import type { Metadata } from "next";

import { ModuleDetailView } from "@/components/module-detail-view";

export const metadata: Metadata = {
  title: "Modul | PAMS",
  description: "Detail modul Inventory dari template desain Sanggar ERP.",
};

export default function ModulesPage() {
  return <ModuleDetailView />;
}
