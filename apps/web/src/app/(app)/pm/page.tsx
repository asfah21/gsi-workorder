import type { Metadata } from "next";

import { PMView } from "@/components/pm-view";

export const metadata: Metadata = {
  title: "Maintenance | PAMS",
  description: "Kelola jadwal dan kepatuhan preventive maintenance GSI.",
};

export default function PMPage() {
  return <PMView />;
}
