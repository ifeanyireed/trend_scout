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

  const cleanPath = path.startsWith('/') ? path : `/${path}`;
  const url = API_BASE ? `${API_BASE}${cleanPath}` : cleanPath;
  const isLongOp = cleanPath.includes('/scan') || cleanPath.includes('/generate') || cleanPath.includes('/assemble-video') || cleanPath.includes('/export');
  const timeoutMs = isLongOp ? 55000 : 18000;

  try {
    const resp = await fetchWithTimeout(url, { ...options, headers }, timeoutMs);
    const contentType = resp.headers.get('content-type') || '';

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
