import type { Metadata } from "next";

import { UnitsView } from "@/components/units-view";

export const metadata: Metadata = {
  title: "Unit | GSI Work Order",
  description: "Master unit dan HM/KM",
};

export default function UnitsPage() {
  return <UnitsView />;
}
