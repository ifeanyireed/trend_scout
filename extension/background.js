const DEFAULT_API_BASE = 'https://trendscout-iota.vercel.app';
const DEFAULT_DASHBOARD_BASE = 'https://trendscout-iota.vercel.app';

function cleanBaseUrl(u, fallback) {
  const cleaned = String(u || fallback).trim().replace(/\/+$/, '');
  if (!cleaned || cleaned.includes('localhost') || cleaned.includes('trendscout-api.vercel.app')) {
    return fallback;
  }
  return cleaned;
}

async function ensurePollingAlarm() {
  const existing = await chrome.alarms.get('trendscout-poll');
  if (!existing) {
    chrome.alarms.create('trendscout-poll', { periodInMinutes: 1 });
  }
}

async function migrateLegacyStorage() {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'notificationsEnabled', 'seenNotificationIds']);
  const needsMigration =
    !stored.apiBaseUrl ||
    stored.apiBaseUrl.includes('localhost') ||
    stored.apiBaseUrl.includes('trendscout-api.vercel.app');
  if (needsMigration) {
    await chrome.storage.local.set({
      apiBaseUrl: DEFAULT_API_BASE,
      dashboardBaseUrl: stored.dashboardBaseUrl && !stored.dashboardBaseUrl.includes('localhost') ? stored.dashboardBaseUrl : DEFAULT_DASHBOARD_BASE,
      notificationsEnabled: stored.notificationsEnabled !== false,
      seenNotificationIds: stored.seenNotificationIds || [],
    });
  }
}

chrome.runtime.onInstalled.addListener(async () => {
  await migrateLegacyStorage();
  await ensurePollingAlarm();
  pollBackendSummary();
});

chrome.runtime.onStartup.addListener(async () => {
  await migrateLegacyStorage();
  await ensurePollingAlarm();
  pollBackendSummary();
});

migrateLegacyStorage().then(() => ensurePollingAlarm());

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'trendscout-poll') {
    pollBackendSummary();
  }
});

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg && msg.type === 'POLL_NOW') {
    pollBackendSummary().then((data) => sendResponse({ ok: true, data })).catch((err) => sendResponse({ ok: false, error: String(err) }));
    return true;
  }
});

chrome.notifications.onClicked.addListener(async (notificationId) => {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'authToken', 'dashboardBaseUrl', 'notificationRoutes']);
  const apiBase = cleanBaseUrl(stored.apiBaseUrl, DEFAULT_API_BASE);
  const dashBase = cleanBaseUrl(stored.dashboardBaseUrl, DEFAULT_DASHBOARD_BASE);
  const routes = stored.notificationRoutes || {};
  const targetPath = routes[notificationId] || '/';
  // Mark notification read on cloud backend
  try {
    const headers = { 'Content-Type': 'application/json' };
    if (stored.authToken) headers['Authorization'] = `Bearer ${stored.authToken}`;
    await fetch(`${apiBase}/api/notifications/${encodeURIComponent(notificationId)}/read`, { method: 'POST', headers });
  } catch (_) {}
  chrome.tabs.create({ url: `${dashBase}${targetPath}` });
});

async function refreshExtensionToken(apiBase, refreshToken) {
  if (!refreshToken) return null;
  try {
    const resp = await fetch(`${apiBase}/api/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    });
    if (!resp.ok) {
      await chrome.storage.local.remove(['authToken', 'refreshToken', 'tokenExpiresAt']);
      return null;
    }
    const data = await resp.json();
    if (data?.token) {
      await chrome.storage.local.set({
        authToken: data.token,
        refreshToken: data.refreshToken || refreshToken,
        tokenExpiresAt: data.expiresAt || '',
      });
      return data.token;
    }
  } catch (_) {}
  return null;
}

async function pollBackendSummary() {
  try {
    const stored = await chrome.storage.local.get([
      'apiBaseUrl',
      'authToken',
      'refreshToken',
      'notificationsEnabled',
      'seenNotificationIds',
      'notificationRoutes',
    ]);
    const apiBase = cleanBaseUrl(stored.apiBaseUrl, DEFAULT_API_BASE);
    let token = stored.authToken || '';
    const buildHeaders = (tok) => {
      const h = { 'Content-Type': 'application/json' };
      if (tok) h['Authorization'] = `Bearer ${tok}`;
      return h;
    };

    let resp = await fetch(`${apiBase}/api/extension/summary`, { headers: buildHeaders(token) });
    if (resp.status === 401 && stored.refreshToken) {
      const refreshed = await refreshExtensionToken(apiBase, stored.refreshToken);
      if (refreshed) {
        token = refreshed;
        resp = await fetch(`${apiBase}/api/extension/summary`, { headers: buildHeaders(token) });
      }
    }
    if (resp.status === 401) {
      chrome.action.setBadgeText({ text: '🔑' });
      chrome.action.setBadgeBackgroundColor({ color: '#f59e0b' });
      await chrome.storage.local.set({ connectionStatus: 'auth_required' });
      return null;
    }
    if (!resp.ok) {
      chrome.action.setBadgeText({ text: '!' });
      chrome.action.setBadgeBackgroundColor({ color: '#ef4444' });
      return null;
    }

    const data = await resp.json();
    const count = Number(data.newDiscoveriesCount || 0);
    chrome.action.setBadgeText({ text: count > 0 ? String(count) : '' });
    chrome.action.setBadgeBackgroundColor({ color: '#E02424' });

    await chrome.storage.local.set({
      latestSummary: data,
      lastSyncedAt: new Date().toISOString(),
      connectionStatus: 'connected',
    });

    if (stored.notificationsEnabled !== false && Array.isArray(data.unreadNotifications)) {
      const seen = new Set(stored.seenNotificationIds || []);
      const routes = stored.notificationRoutes || {};
      for (const n of data.unreadNotifications) {
        if (!seen.has(n.id)) {
          seen.add(n.id);
          routes[n.id] = n.actionUrl || '/';
          chrome.notifications.create(n.id, {
            type: 'basic',
            iconUrl: 'icons/icon128.png',
            title: n.title || 'TrendScout Studio Alert',
            message: n.message || 'New underdog opportunity update.',
            priority: 2,
          });
        }
      }
      await chrome.storage.local.set({
        seenNotificationIds: Array.from(seen).slice(-100),
        notificationRoutes: routes,
      });
    }

    return data;
  } catch (err) {
    await chrome.storage.local.set({ connectionStatus: 'disconnected' });
    return null;
  }
}
