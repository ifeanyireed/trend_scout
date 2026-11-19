const DEFAULT_FALLBACK_API = 'https://trendscout-api.vercel.app';
const rawBase = ((import.meta as any).env?.VITE_API_BASE_URL || '').trim();
// Strip trailing slashes so URL joining never produces "//api"
const API_BASE = rawBase.replace(/\/+$/, '');

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

  const cleanPath = path.startsWith('/') ? path : `/${path}`;
  const url = API_BASE ? `${API_BASE}${cleanPath}` : cleanPath;

  try {
    const resp = await fetch(url, {
      ...options,
      headers,
    });

    const contentType = resp.headers.get('content-type') || '';
    if (resp.ok && contentType.includes('application/json')) {
      return resp.json();
    }

    // If request returned non-JSON (e.g. HTML from SPA rewrite) or 404 on relative path, fallback to direct backend
    if (!API_BASE && (!resp.ok || !contentType.includes('application/json'))) {
      const fallbackUrl = `${DEFAULT_FALLBACK_API}${cleanPath}`;
      const fbResp = await fetch(fallbackUrl, {
        ...options,
        mode: 'cors',
        headers,
      });
      if (fbResp.ok) {
        return fbResp.json();
      }
    }

    const errBody = await resp.json().catch(() => ({}));
    throw new Error(errBody.error || `Request failed (${resp.status})`);
  } catch (err: any) {
    // If request failed and we did not already call clean DEFAULT_FALLBACK_API, attempt clean fallback
    if (url !== `${DEFAULT_FALLBACK_API}${cleanPath}`) {
      try {
        const fallbackUrl = `${DEFAULT_FALLBACK_API}${cleanPath}`;
        const fbResp = await fetch(fallbackUrl, {
          ...options,
          mode: 'cors',
          headers,
        });
        if (fbResp.ok) {
          return fbResp.json();
        }
      } catch (_) {
        // Fall through to original error
      }
    }
    throw err;
  }
}
