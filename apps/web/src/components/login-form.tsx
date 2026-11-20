"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { ApiError, login } from "@/lib/api-client";

export function LoginForm() {
  const router = useRouter();
  const [nik, setNik] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setPending(true);
    try {
      await login(nik.trim(), password);
      router.replace("/dashboard");
      router.refresh();
    } catch (loginError) {
      if (loginError instanceof ApiError) {
        setError(loginError.message);
      } else {
        setError("Login gagal. Periksa koneksi lalu coba lagi.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <main className="grid min-h-dvh place-items-center bg-muted/40 px-4 py-10">
      <div className="w-full max-w-md rounded-xl border border-border bg-background p-6 shadow-sm sm:p-8">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">
          PT GSI
        </p>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Masuk ke Work Order</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Gunakan akun Odoo yang sudah terdaftar di aplikasi.
        </p>

        <form
          className="mt-6 space-y-5"
          onSubmit={(event) => void handleSubmit(event)}
          aria-labelledby="login-heading"
          noValidate
        >
          <h2 id="login-heading" className="sr-only">
            Form login
          </h2>
          <div>
            <label htmlFor="nik" className="text-sm font-medium">
              NIK / Login
            </label>
            <input
              id="nik"
              name="nik"
              type="text"
              autoComplete="username"
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              required
              value={nik}
              onChange={(event) => setNik(event.target.value)}
              className="mt-1 min-h-12 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
            />
          </div>

          <div>
            <label htmlFor="password" className="text-sm font-medium">
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="mt-1 min-h-12 w-full rounded-md border border-input bg-background px-3 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
            />
          </div>

          {error ? (
            <p role="alert" className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {error}
            </p>
          ) : null}

          <button
            type="submit"
            disabled={pending}
            className="min-h-12 w-full rounded-md bg-primary px-4 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            {pending ? "Memproses..." : "Masuk"}
          </button>
        </form>
      </div>
    </main>
  );
}
