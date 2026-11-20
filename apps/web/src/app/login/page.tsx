import type { Metadata } from "next";

import { LoginForm } from "@/components/login-form";

export const metadata: Metadata = {
  title: "Masuk | GSI Work Order",
  description: "Login aplikasi GSI Work Order",
};

export default function LoginPage() {
  return <LoginForm />;
}
