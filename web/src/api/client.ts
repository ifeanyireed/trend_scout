const API_BASE = (import.meta as any).env?.VITE_API_BASE_URL || '';

export function getAuthToken(): string | null {
  return localStorage.getItem('ts_token');
}

export function setAuthSession(token: string, refreshToken: string, user: any) {
  localStorage.setItem('ts_token', token);
  localStorage.setItem('ts_refresh_token', refreshToken);
  localStorage.setItem('ts_user', JSON.stringify(user));
}

export function clearAuthSession() {
  localStorage.removeItem('ts_token');
  localStorage.removeItem('ts_refresh_token');
  localStorage.removeItem('ts_user');
}

export async function apiFetch<T = any>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };
  if (!(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const resp = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  if (!resp.ok) {
    const errBody = await resp.json().catch(() => ({}));
    throw new Error(errBody.error || `Request failed (${resp.status})`);
  }

  return resp.json();
}
