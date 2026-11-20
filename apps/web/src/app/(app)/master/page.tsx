import type { Metadata } from "next";

import { MasterView } from "@/components/master-view";

export const metadata: Metadata = {
  title: "Master Unit | GSI Work Order",
  description: "Master unit model dan engine model",
};

export default function MasterPage() {
  return <MasterView />;
}
