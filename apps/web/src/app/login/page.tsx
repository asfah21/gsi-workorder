import type { Metadata } from "next";

import { LoginForm } from "@/components/login-form";

export const metadata: Metadata = {
  title: "Masuk | PAMS",
  description: "Login aplikasi PAMS",
};

export default function LoginPage() {
  return <LoginForm />;
}
