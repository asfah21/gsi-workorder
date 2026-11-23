import type { Metadata } from "next";
import { redirect } from "next/navigation";

export const metadata: Metadata = {
  title: "Tambah Unit | PAMS",
};

export default function NewUnitPage() {
  redirect("/units?create=1");
}
