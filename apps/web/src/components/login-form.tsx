"use client";

import {
  CircleAlert,
  Eye,
  EyeOff,
  Gauge,
  Loader2,
  ShieldCheck,
  Truck,
  Wrench,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { ThemeToggle } from "@/components/theme-toggle";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ApiError, login } from "@/lib/api-client";

const highlights = [
  {
    icon: ShieldCheck,
    title: "Akses sesuai role",
    description: "Admin, planner, mechanic, storekeeper, dan viewer.",
  },
  {
    icon: Truck,
    title: "Master unit terpusat",
    description: "Identitas unit, model, dan engine dalam satu tempat.",
  },
  {
    icon: Gauge,
    title: "HM/KM tercatat",
    description: "Reading manual dan riwayat terbaru selalu tersedia.",
  },
];

export function LoginForm() {
  const router = useRouter();
  const [nik, setNik] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<{ nik?: string; password?: string }>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const focusFirstError = (errors: { nik?: string; password?: string }) => {
    const first = (["nik", "password"] as const).find((key) => errors[key]);
    if (first) {
      requestAnimationFrame(() => document.getElementById(first)?.focus());
    }
  };

  const clearFieldError = (key: "nik" | "password") => {
    setFieldErrors((current) => {
      if (!current[key]) return current;
      const next = { ...current };
      delete next[key];
      return next;
    });
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setFieldErrors({});

    const nextErrors: { nik?: string; password?: string } = {};
    if (!nik.trim()) nextErrors.nik = "NIK atau login wajib diisi.";
    if (!password) nextErrors.password = "Password wajib diisi.";
    if (nextErrors.nik || nextErrors.password) {
      setFieldErrors(nextErrors);
      focusFirstError(nextErrors);
      return;
    }

    setPending(true);
    try {
      await login(nik.trim(), password);
      router.replace("/dashboard");
      router.refresh();
    } catch (loginError) {
      if (loginError instanceof ApiError) {
        const mapped: { nik?: string; password?: string } = {
          nik: loginError.fields.nik,
          password: loginError.fields.password,
        };
        if (mapped.nik || mapped.password) {
          setFieldErrors(mapped);
          focusFirstError(mapped);
        } else {
          requestAnimationFrame(() => document.getElementById("login-error")?.focus());
        }
        setError(loginError.message);
      } else {
        setError("Login gagal. Periksa koneksi lalu coba lagi.");
        requestAnimationFrame(() => document.getElementById("login-error")?.focus());
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <main className="relative grid min-h-dvh min-w-0 bg-muted/30 lg:grid-cols-[1.05fr_0.95fr]">
      <div className="absolute right-4 top-4 z-10 sm:right-6 sm:top-6">
        <ThemeToggle />
      </div>
      <section className="relative hidden overflow-hidden bg-slate-950 lg:flex lg:flex-col lg:justify-between lg:p-10 xl:p-14">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage:
              "radial-gradient(circle at 15% 5%, rgba(45, 212, 191, 0.24), transparent 42%), radial-gradient(circle at 88% 92%, rgba(59, 130, 246, 0.18), transparent 40%)",
          }}
        />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 opacity-[0.07]"
          style={{
            backgroundImage:
              "linear-gradient(to right, white 1px, transparent 1px), linear-gradient(to bottom, white 1px, transparent 1px)",
            backgroundSize: "32px 32px",
          }}
        />

        <div className="relative flex items-center gap-3">
          <span className="grid size-11 place-items-center rounded-xl bg-white/10 text-teal-200 ring-1 ring-white/15 backdrop-blur">
            <Wrench className="size-5" aria-hidden="true" />
          </span>
          <div>
            <p className="text-sm font-semibold tracking-tight text-white">GSI Work Order</p>
            <p className="text-xs text-slate-400">Maintenance &amp; Unit</p>
          </div>
        </div>

        <div className="relative max-w-xl">
          <p className="text-xs font-semibold uppercase tracking-[0.28em] text-teal-300">
            PT GSI · Internal
          </p>
          <p className="mt-4 text-4xl font-semibold leading-tight tracking-tight text-white xl:text-5xl">
            Maintenance terpusat, keputusan lebih cepat.
          </p>
          <p className="mt-4 max-w-lg text-base leading-7 text-slate-300">
            Kelola master unit, model, dan pembacaan HM/KM dalam satu alur kerja yang
            konsisten.
          </p>
          <ul className="mt-8 space-y-4">
            {highlights.map((item) => (
              <li key={item.title} className="flex gap-3">
                <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-lg bg-white/10 text-teal-200 ring-1 ring-white/10">
                  <item.icon className="size-4" aria-hidden="true" />
                </span>
                <span>
                  <span className="block text-sm font-medium text-slate-100">{item.title}</span>
                  <span className="mt-0.5 block text-sm leading-6 text-slate-400">
                    {item.description}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-slate-500">GSI Work Order · Milestone 1</p>
      </section>

      <section className="flex min-h-dvh min-w-0 items-center justify-center px-4 py-10 sm:px-6 lg:px-10">
        <div className="w-full min-w-0 max-w-md">
          <div className="mb-8 flex items-center gap-3 lg:hidden">
            <span className="grid size-11 place-items-center rounded-xl bg-primary text-primary-foreground shadow-sm">
              <Wrench className="size-5" aria-hidden="true" />
            </span>
            <div>
              <p className="text-sm font-semibold tracking-tight">GSI Work Order</p>
              <p className="text-xs text-muted-foreground">Maintenance &amp; Unit</p>
            </div>
          </div>

          <div className="rounded-2xl border border-border bg-background p-6 shadow-sm sm:p-8">
            <div className="mb-7">
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                PT GSI
              </p>
              <h1 id="login-heading" className="mt-2 text-2xl font-semibold tracking-tight">
                Masuk ke Work Order
              </h1>
              <p id="login-help" className="mt-2 text-sm leading-6 text-muted-foreground">
                Gunakan akun Odoo yang sudah terdaftar di aplikasi.
              </p>
            </div>

            <form
              className="space-y-5"
              onSubmit={(event) => void handleSubmit(event)}
              aria-labelledby="login-heading"
              aria-describedby="login-help"
              aria-busy={pending}
              noValidate
            >
              <Field data-invalid={Boolean(fieldErrors.nik)}>
                <FieldLabel htmlFor="nik">
                  NIK / Login <span aria-hidden="true" className="text-destructive">*</span>
                </FieldLabel>
                <Input
                  id="nik"
                  name="nik"
                  type="text"
                  autoComplete="username"
                  autoCapitalize="none"
                  autoCorrect="off"
                  spellCheck={false}
                  enterKeyHint="next"
                  required
                  value={nik}
                  onChange={(event) => {
                    setNik(event.target.value);
                    clearFieldError("nik");
                  }}
                  aria-invalid={Boolean(fieldErrors.nik)}
                  aria-describedby={fieldErrors.nik ? "nik-error" : undefined}
                  placeholder="Masukkan NIK atau login Odoo"
                  className="h-11"
                />
                {fieldErrors.nik ? (
                  <FieldError id="nik-error">{fieldErrors.nik}</FieldError>
                ) : null}
              </Field>

              <Field data-invalid={Boolean(fieldErrors.password)}>
                <FieldLabel htmlFor="password">
                  Password <span aria-hidden="true" className="text-destructive">*</span>
                </FieldLabel>
                <div className="relative">
                  <Input
                    id="password"
                    name="password"
                    type={showPassword ? "text" : "password"}
                    autoComplete="current-password"
                    enterKeyHint="done"
                    required
                    value={password}
                    onChange={(event) => {
                      setPassword(event.target.value);
                      clearFieldError("password");
                    }}
                    aria-invalid={Boolean(fieldErrors.password)}
                    aria-describedby={fieldErrors.password ? "password-error" : undefined}
                    className="h-11 pr-12"
                  />
                  <button
                    type="button"
                    aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                    aria-pressed={showPassword}
                    aria-controls="password"
                    onClick={() => setShowPassword((current) => !current)}
                    className="absolute inset-y-0 right-0 grid w-11 place-items-center rounded-r-lg text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    {showPassword ? (
                      <EyeOff className="size-4" aria-hidden="true" />
                    ) : (
                      <Eye className="size-4" aria-hidden="true" />
                    )}
                  </button>
                </div>
                {fieldErrors.password ? (
                  <FieldError id="password-error">{fieldErrors.password}</FieldError>
                ) : null}
              </Field>

              {error ? (
                <Alert
                  id="login-error"
                  tabIndex={-1}
                  variant="destructive"
                  className="border-destructive/30 bg-destructive/5"
                >
                  <CircleAlert aria-hidden="true" />
                  <AlertTitle>Login gagal</AlertTitle>
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              ) : null}

              <Button
                type="submit"
                disabled={pending}
                className="h-11 w-full px-4 text-sm font-semibold"
              >
                {pending ? (
                  <>
                    <Loader2 className="animate-spin" aria-hidden="true" />
                    Memproses...
                  </>
                ) : (
                  "Masuk"
                )}
              </Button>
            </form>

            <p className="mt-6 text-center text-xs leading-5 text-muted-foreground">
              Akses internal PT GSI. Hubungi admin bila akun belum terdaftar.
            </p>
          </div>

          <p className="mt-6 text-center text-xs text-muted-foreground lg:hidden">
            GSI Work Order · Internal
          </p>
        </div>
      </section>
    </main>
  );
}
