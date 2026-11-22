import type { ApiErrorBody } from "@/lib/types";

export class ApiError extends Error {
  status: number;
  code: string;
  fields: Record<string, string>;

  constructor(status: number, body: ApiErrorBody | null) {
    super(body?.error?.message || "Permintaan gagal");
    this.name = "ApiError";
    this.status = status;
    this.code = body?.error?.code || "request_failed";
    this.fields = body?.error?.fields || {};
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = init?.body instanceof FormData;
  const response = await fetch(path, {
    ...init,
    credentials: "same-origin",
    cache: "no-store",
    headers: {
      Accept: "application/json",
      ...(init?.body && !isFormData ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  const body = (await response.json().catch(() => null)) as T | ApiErrorBody | null;
  if (!response.ok) {
    throw new ApiError(response.status, body as ApiErrorBody | null);
  }
  return body as T;
}

export async function login(nik: string, password: string): Promise<{ user: import("@/lib/types").User }> {
  return request("/api/session/login", {
    method: "POST",
    body: JSON.stringify({ nik, password }),
  });
}

export async function getSession(): Promise<import("@/lib/types").User> {
  return request("/api/session/me");
}

export async function logout(): Promise<void> {
  await fetch("/api/session/logout", {
    method: "POST",
    credentials: "same-origin",
    cache: "no-store",
  });
}

export function proxyGet<T>(path: string): Promise<T> {
  return request(`/api/proxy${path}`);
}

export function proxyPost<T>(path: string, body: unknown): Promise<T> {
  return request(`/api/proxy${path}`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function proxyPatch<T>(path: string, body: unknown): Promise<T> {
  return request(`/api/proxy${path}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export async function proxyDelete(path: string): Promise<void> {
  const response = await fetch(`/api/proxy${path}`, {
    method: "DELETE",
    credentials: "same-origin",
    cache: "no-store",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as ApiErrorBody | null;
    throw new ApiError(response.status, body);
  }
}

export function proxyUpload<T>(path: string, formData: FormData): Promise<T> {
  return request(`/api/proxy${path}`, {
    method: "POST",
    body: formData,
  });
}
