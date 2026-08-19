// web/src/lib/api/client.ts
export class APIError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = 'APIError';
  }
}

export type RequestOptions = Omit<RequestInit, 'body'> & { body?: unknown };

function cookie(name: string): string | undefined {
  const value = document.cookie.split('; ').find((entry) => entry.startsWith(`${name}=`));
  return value?.slice(name.length + 1);
}

export async function request<T = unknown>(path: string, options: RequestOptions = {}): Promise<T | null> {
  const method = (options.method ?? 'GET').toUpperCase();
  // Plain object keeps canonical header casing (Headers lowercases names).
  const headers: Record<string, string> = {};
  if (options.headers) {
    const incoming = new Headers(options.headers);
    incoming.forEach((value, key) => {
      headers[key] = value;
    });
  }
  headers.Accept = 'application/json';
  let body: BodyInit | undefined;
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
  }
  if (!['GET', 'HEAD'].includes(method)) {
    const csrf = cookie('pa_csrf');
    if (csrf) headers['X-CSRF-Token'] = decodeURIComponent(csrf);
  }
  const response = await fetch(path, { ...options, method, headers, body });
  const text = response.status === 204 ? '' : await response.text();
  if (!response.ok) {
    let message = text.trim();
    try {
      const data = JSON.parse(text) as { message?: string; code?: string; error?: string };
      message = data.message ?? data.code ?? data.error?.replaceAll('_', ' ') ?? message;
    } catch { /* retain plain-text response */ }
    throw new APIError(response.status, message || `Request failed (${response.status})`);
  }
  return text.trim() ? JSON.parse(text) as T : null;
}

export const get = <T>(path: string) => request<T>(path);
export const mutate = <T>(path: string, method: string, body: unknown) => request<T>(path, { method, body });
export const post = <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body });

/** Typed client used by route pages (paths include /api/v1 prefix). */
export const api = {
  get: <T>(path: string) => request<T>(path) as Promise<T>,
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }) as Promise<T>,
};
