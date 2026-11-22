"use client";

import {
  Building2,
  CircleAlert,
  LifeBuoy,
  Loader2,
  Wrench,
} from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FieldError } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError, login } from "@/lib/api-client";

const REMEMBERED_NIK_KEY = "gsi-login-nik";

export function LoginForm() {
  const router = useRouter();
  const nikRef = useRef<HTMLInputElement>(null);
  const [nik, setNik] = useState("");
  const [password, setPassword] = useState("");
  const [rememberNik, setRememberNik] = useState(false);
  const [showHelp, setShowHelp] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<{ nik?: string; password?: string }>({});
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    try {
      const remembered = window.localStorage.getItem(REMEMBERED_NIK_KEY);
      if (remembered) {
        setNik(remembered);
        setRememberNik(true);
      }
    } catch {
      // Remembered NIK is optional; ignore unavailable storage.
    }
  }, []);

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
      try {
        if (rememberNik) {
          window.localStorage.setItem(REMEMBERED_NIK_KEY, nik.trim());
        } else {
          window.localStorage.removeItem(REMEMBERED_NIK_KEY);
        }
      } catch {
        // Login succeeds even if browser storage is unavailable.
      }
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
    <section className="flex min-h-screen items-center justify-center overflow-hidden bg-background">
      <div className="flex min-h-screen w-full items-stretch">
        <div className="dark hidden lg:flex lg:w-[32rem]">
          <div className="relative flex w-full flex-col items-center justify-center bg-background p-16 text-center text-foreground">
            <div className="z-10 flex max-w-sm flex-col items-center gap-6">
              <Link href="/" aria-label="GSI Work Order" className="grid size-10 place-items-center rounded-xl border border-white/15 bg-white/10 text-white backdrop-blur-sm">
                <Wrench className="size-5" aria-hidden="true" />
              </Link>
              <p className="text-3xl font-medium">Selamat Datang Kembali</p>
            </div>
            <div className="absolute z-0 h-full w-full">
              <video
                className="h-full w-full object-cover"
                loop
                autoPlay
                muted
                playsInline
              >
                <source src="/videos/wave-video-loop.mp4" type="video/mp4" />
              </video>
            </div>
          </div>
        </div>

        <div className="flex min-w-0 flex-1 flex-col items-center justify-center p-8 md:p-12">
          <div className="flex w-full min-w-0 max-w-md flex-col items-center gap-6">
            <Link href="/" aria-label="GSI Work Order" className="grid size-10 place-items-center rounded-xl bg-primary text-primary-foreground">
              <Wrench className="size-5" aria-hidden="true" />
            </Link>

            <div className="flex flex-col gap-1 text-center">
              <h1 id="login-heading" className="text-2xl font-medium text-foreground">
                Masuk ke GSI Work Order
              </h1>
              <p id="login-help" className="text-sm font-normal text-muted-foreground">
                Gunakan akun Odoo yang sudah terdaftar
              </p>
            </div>

            <div className="flex w-full flex-col items-center justify-center gap-3 sm:flex-row">
              <Button
                type="button"
                variant="outline"
                className="h-9 w-full flex-1 gap-2 rounded-lg px-5 text-sm font-medium"
                onClick={() => {
                  setShowHelp(false);
                  nikRef.current?.focus();
                }}
              >
                <Building2 className="size-4" aria-hidden="true" />
                Akun Odoo
              </Button>
              <Button
                type="button"
                variant="outline"
                className="h-9 w-full flex-1 gap-2 rounded-lg px-5 text-sm font-medium"
                aria-expanded={showHelp}
                aria-controls="access-help"
                onClick={() => setShowHelp((current) => !current)}
              >
                <LifeBuoy className="size-4" aria-hidden="true" />
                Bantuan akses
              </Button>
            </div>

            <div className="flex w-full items-center gap-3 text-sm font-normal text-muted-foreground before:h-px before:flex-1 before:bg-border before:content-[''] after:h-px after:flex-1 after:bg-border after:content-['']">
              atau masuk dengan
            </div>

            {showHelp ? (
              <p id="access-help" className="w-full rounded-lg border border-border bg-muted/40 px-3 py-2 text-xs leading-5 text-muted-foreground">
                Akun dikelola melalui Odoo. Hubungi admin internal GSI jika NIK belum terdaftar atau password perlu direset.
              </p>
            ) : null}

            <form
              className="w-full min-w-0 space-y-6"
              onSubmit={(event) => void handleSubmit(event)}
              aria-labelledby="login-heading"
              aria-describedby="login-help"
              aria-busy={pending}
              noValidate
            >
              <div className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="nik" className="text-sm font-normal text-muted-foreground">
                    NIK / Login<span aria-hidden="true">*</span>
                  </Label>
                  <Input
                    ref={nikRef}
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
                    className="h-8 rounded-lg px-2.5 py-1 text-base shadow-none md:text-sm"
                  />
                  {fieldErrors.nik ? <FieldError id="nik-error">{fieldErrors.nik}</FieldError> : null}
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="password" className="text-sm font-normal text-muted-foreground">
                    Password<span aria-hidden="true">*</span>
                  </Label>
                  <Input
                    id="password"
                    name="password"
                    type="password"
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
                    placeholder="Masukkan password"
                    className="h-8 rounded-lg px-2.5 py-1 text-base shadow-none md:text-sm"
                  />
                  {fieldErrors.password ? <FieldError id="password-error">{fieldErrors.password}</FieldError> : null}
                </div>

                <div className="flex flex-wrap items-center justify-between gap-4 text-sm">
                  <label htmlFor="remember" className="flex cursor-pointer items-center gap-2 text-sm font-normal text-muted-foreground">
                    <input
                      id="remember"
                      name="remember"
                      type="checkbox"
                      checked={rememberNik}
                      onChange={(event) => setRememberNik(event.target.checked)}
                      className="size-4 cursor-pointer rounded-[4px] border border-input accent-primary"
                    />
                    Ingat NIK di perangkat ini
                  </label>
                  <button
                    type="button"
                    className="text-sm font-medium underline-offset-4 transition-all hover:underline"
                    onClick={() => setShowHelp(true)}
                  >
                    Lupa password?
                  </button>
                </div>
              </div>

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

              <Button type="submit" disabled={pending} className="h-9 w-full rounded-lg">
                {pending ? (
                  <>
                    <Loader2 className="animate-spin" aria-hidden="true" />
                    Memproses...
                  </>
                ) : (
                  "Masuk"
                )}
              </Button>

              <p className="text-center text-sm font-normal text-muted-foreground">
                Akses internal GSI. Hubungi admin jika akun belum terdaftar.
              </p>
            </form>
          </div>
        </div>
      </div>
    </section>
  );
}
