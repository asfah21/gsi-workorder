"use client";

import { CircleAlert, Eye, EyeOff, Loader2, Wrench } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { ThemeToggle } from "@/components/theme-toggle";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ApiError, login } from "@/lib/api-client";

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
    <main className="relative flex min-h-dvh min-w-0 items-center justify-center bg-accent px-4 py-10">
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>

      <div className="w-full min-w-0 max-w-md">
        <div className="mb-6 flex items-center justify-center gap-3">
          <span className="grid size-9 place-items-center rounded-md bg-primary text-primary-foreground">
            <Wrench className="size-4" aria-hidden="true" />
          </span>
          <div>
            <p className="text-sm font-semibold">GSI Work Order</p>
            <p className="text-xs text-muted-foreground">Maintenance &amp; Unit</p>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-background p-6 shadow-lg sm:p-8">
          <div className="mb-6">
            <h1 id="login-heading" className="text-xl font-semibold">
              Masuk ke Work Order
            </h1>
            <p id="login-help" className="mt-2 text-sm text-muted-foreground">
              Gunakan akun Odoo yang sudah terdaftar di aplikasi.
            </p>
          </div>

          <form
            className="space-y-4"
            onSubmit={(event) => void handleSubmit(event)}
            aria-labelledby="login-heading"
            aria-describedby="login-help"
            aria-busy={pending}
            noValidate
          >
            <Field data-invalid={Boolean(fieldErrors.nik)}>
              <FieldLabel htmlFor="nik" className="text-muted-foreground">
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
              />
              {fieldErrors.nik ? (
                <FieldError id="nik-error">{fieldErrors.nik}</FieldError>
              ) : null}
            </Field>

            <Field data-invalid={Boolean(fieldErrors.password)}>
              <FieldLabel htmlFor="password" className="text-muted-foreground">
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
                  className="pr-10"
                />
                <button
                  type="button"
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                  aria-pressed={showPassword}
                  aria-controls="password"
                  onClick={() => setShowPassword((current) => !current)}
                  className="absolute inset-y-0 right-0 grid w-9 place-items-center rounded-r-md text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
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

            <Button type="submit" disabled={pending} className="w-full">
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

          <p className="mt-6 text-center text-xs text-muted-foreground">
            Akses internal PT GSI. Hubungi admin bila akun belum terdaftar.
          </p>
        </div>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          GSI Work Order · Internal
        </p>
      </div>
    </main>
  );
}
