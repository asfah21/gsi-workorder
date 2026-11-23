import type { Metadata } from "next";

import { ModuleFormView } from "@/components/module-form-view";

export const metadata: Metadata = {
  title: "Tambah Modul | PAMS",
  description: "Form unggah modul dari template desain Sanggar ERP.",
};

export default function ModuleNewPage() {
  return <ModuleFormView />;
}
