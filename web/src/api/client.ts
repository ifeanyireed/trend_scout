const DEFAULT_FALLBACK_API = 'https://trendscout-iota.vercel.app';
const rawBase = ((import.meta as any).env?.VITE_API_BASE_URL || '').trim().replace(/\/+$/, '');

// On deployed HTTPS web domains (e.g. trendscout-iota.vercel.app), always route through
// same-origin Vercel edge proxy (/api/*) instead of calling trendscout-api.vercel.app directly,
// which avoids CORS preflights and ISP routing timeouts to secondary Vercel IP pools.
function resolveApiBase(): string {
  if (typeof window !== 'undefined') {
    const host = window.location.hostname;
    const isLocal = host === 'localhost' || host === '127.0.0.1';
    if (!isLocal && (!rawBase || rawBase.includes('trendscout-api.vercel.app'))) {
      return '';
    }
  }
  return rawBase;
}

const API_BASE = resolveApiBase();

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: string;
}

export function parseJwtPayload(token: string | null): Record<string, any> | null {
  if (!token) return null;
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
    const json = atob(padded);
    return JSON.parse(json);
  } catch {
    return null;
  }
}

export function getAuthToken(): string | null {
  return localStorage.getItem('ts_token');
}

export function getRefreshToken(): string | null {
  return localStorage.getItem('ts_refresh_token');
}

export function getAuthUser(): AuthUser | null {
  try {
    const raw = localStorage.getItem('ts_user');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function getTokenExpiryMs(token: string | null): number | null {
  const payload = parseJwtPayload(token);
  if (payload && typeof payload.exp === 'number') {
    return payload.exp * 1000;
  }
  const stored = localStorage.getItem('ts_expires_at');
  if (stored) {
    const ms = new Date(stored).getTime();
    if (!isNaN(ms)) return ms;
  }
  return null;
}

export function isTokenExpired(token: string | null, skewSeconds = 15): boolean {
  if (!token) return true;
  const expMs = getTokenExpiryMs(token);
  if (!expMs) return true;
  return Date.now() >= expMs - skewSeconds * 1000;
}

export function getSessionRemainingMs(): number {
  const token = getAuthToken();
  const expMs = getTokenExpiryMs(token);
  if (!expMs) return 0;
  return Math.max(0, expMs - Date.now());
}

export function hasActiveSession(): boolean {
  const token = getAuthToken();
  const refresh = getRefreshToken();
  if (token && !isTokenExpired(token, 5)) {
    return true;
  }
  if (refresh && !isTokenExpired(refresh, 5)) {
    return true;
  }
  return false;
}

export function setAuthSession(
  token: string,
  refreshToken: string,
  user: any,
  expiresAt?: string,
  refreshExpiresAt?: string
) {
  localStorage.setItem('ts_token', token);
  if (refreshToken) {
    localStorage.setItem('ts_refresh_token', refreshToken);
  }
  if (user) {
    localStorage.setItem('ts_user', JSON.stringify(user));
  }
  if (expiresAt) {
    localStorage.setItem('ts_expires_at', expiresAt);
  } else {
    const expMs = getTokenExpiryMs(token);
    if (expMs) localStorage.setItem('ts_expires_at', new Date(expMs).toISOString());
  }
  if (refreshExpiresAt) {
    localStorage.setItem('ts_refresh_expires_at', refreshExpiresAt);
  }
  window.dispatchEvent(new CustomEvent('ts-auth-updated'));
}

export function clearAuthSession() {
  localStorage.removeItem('ts_token');
  localStorage.removeItem('ts_refresh_token');
  localStorage.removeItem('ts_user');
  localStorage.removeItem('ts_expires_at');
  localStorage.removeItem('ts_refresh_expires_at');
  sessionStorage.removeItem('ts_auto_synced');
}

function notifySessionExpired(reason = 'Your session has expired. Please sign in again.') {
  clearAuthSession();
  window.dispatchEvent(new CustomEvent('ts-auth-expired', { detail: { reason } }));
}

async function fetchWithTimeout(url: string, options: RequestInit, timeoutMs: number): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, {
      ...options,
      signal: options.signal || controller.signal,
    });
  } finally {
    clearTimeout(timer);
  }
}

let inflightRefresh: Promise<boolean> | null = null;

export async function refreshAuthSession(): Promise<boolean> {
  if (inflightRefresh) return inflightRefresh;

  const refreshToken = getRefreshToken();
  if (!refreshToken || isTokenExpired(refreshToken, 5)) {
    notifySessionExpired('Your session has expired. Please sign in again.');
    return false;
  }

  inflightRefresh = (async () => {
    try {
      const url = API_BASE ? `${API_BASE}/api/auth/refresh` : '/api/auth/refresh';
      const resp = await fetchWithTimeout(
        url,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refreshToken }),
        },
        15000
      );
      if (!resp.ok) {
        notifySessionExpired('Your session has expired. Please sign in again.');
        return false;
      }
      const data = await resp.json();
      if (!data?.token) {
        notifySessionExpired('Your session has expired. Please sign in again.');
        return false;
      }
      setAuthSession(data.token, data.refreshToken || refreshToken, data.user, data.expiresAt, data.refreshExpiresAt);
      return true;
    } catch {
      return false;
    } finally {
      inflightRefresh = null;
    }
  })();

  return inflightRefresh;
}

export async function ensureValidAccessToken(): Promise<boolean> {
  const token = getAuthToken();
  if (token && !isTokenExpired(token, 90)) {
    return true;
  }
  const refresh = getRefreshToken();
  if (refresh && !isTokenExpired(refresh, 5)) {
    return refreshAuthSession();
  }
  return false;
}

export async function apiFetch<T = any>(path: string, options: RequestInit = {}, retryOnAuth = true): Promise<T> {
  const cleanPath = path.startsWith('/') ? path : `/${path}`;
  const isAuthEndpoint = cleanPath === '/api/auth/login' || cleanPath === '/api/auth/refresh' || cleanPath === '/api/auth/logout';

  // Proactively refresh near-expiry JWT access token before protected requests
  if (!isAuthEndpoint) {
    const curToken = getAuthToken();
    if (curToken && isTokenExpired(curToken, 90)) {
      await refreshAuthSession();
    }
  }

  const token = getAuthToken();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };
  if (!(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  if (token && !isAuthEndpoint) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const url = API_BASE ? `${API_BASE}${cleanPath}` : cleanPath;
  const isLongOp =
    cleanPath.includes('/scan') ||
    cleanPath.includes('/generate') ||
    cleanPath.includes('/assemble-video') ||
    cleanPath.includes('/export');
  const timeoutMs = isLongOp ? 55000 : 18000;

  try {
    const resp = await fetchWithTimeout(url, { ...options, headers }, timeoutMs);
    const contentType = resp.headers.get('content-type') || '';

    // Handle 401 Unauthorized / Expired JWT Session with automatic single retry after refresh
    if (resp.status === 401 && !isAuthEndpoint) {
      if (retryOnAuth && getRefreshToken() && !isTokenExpired(getRefreshToken(), 5)) {
        const refreshed = await refreshAuthSession();
        if (refreshed) {
          return apiFetch<T>(path, options, false);
        }
      }
      const errBody = await resp.json().catch(() => ({}));
      notifySessionExpired(errBody.error || 'Your session has expired. Please sign in again.');
      throw new Error(errBody.error || 'Session expired. Please sign in again.');
    }

    if (resp.ok && contentType.includes('application/json')) {
      return resp.json();
    }

    // If relative path returned HTML (e.g. local static preview without proxy), fallback to edge proxy domain
    if (!API_BASE && (!resp.ok || !contentType.includes('application/json'))) {
      const fallbackUrl = `${DEFAULT_FALLBACK_API}${cleanPath}`;
      if (typeof window === 'undefined' || !window.location.origin.startsWith(DEFAULT_FALLBACK_API)) {
        const fbResp = await fetchWithTimeout(fallbackUrl, { ...options, mode: 'cors', headers }, timeoutMs);
        if (fbResp.ok && (fbResp.headers.get('content-type') || '').includes('application/json')) {
          return fbResp.json();
        }
      }
    }

    const errBody = await resp.json().catch(() => ({}));
    throw new Error(errBody.error || `Request failed (${resp.status})`);
  } catch (err: any) {
    if (err?.name === 'AbortError') {
      throw new Error('Request timed out — please retry in a moment.');
    }
    throw err;
  }
}
