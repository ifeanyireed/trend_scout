const DEFAULT_API_BASE = 'https://trendscout.vercel.app';
const DEFAULT_DASHBOARD_BASE = 'https://trendscout.vercel.app';

chrome.runtime.onInstalled.addListener(async () => {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'notificationsEnabled', 'seenNotificationIds']);
  if (!stored.apiBaseUrl) {
    await chrome.storage.local.set({
      apiBaseUrl: DEFAULT_API_BASE,
      dashboardBaseUrl: DEFAULT_DASHBOARD_BASE,
      notificationsEnabled: true,
      seenNotificationIds: [],
    });
  }
  chrome.alarms.create('trendscout-poll', { periodInMinutes: 1 });
  pollBackendSummary();
});

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
  const stored = await chrome.storage.local.get(['dashboardBaseUrl', 'notificationRoutes']);
  const dashBase = stored.dashboardBaseUrl || DEFAULT_DASHBOARD_BASE;
  const routes = stored.notificationRoutes || {};
  const targetPath = routes[notificationId] || '/';
  chrome.tabs.create({ url: `${dashBase}${targetPath}` });
});

async function pollBackendSummary() {
  try {
    const stored = await chrome.storage.local.get([
      'apiBaseUrl',
      'authToken',
      'notificationsEnabled',
      'seenNotificationIds',
      'notificationRoutes',
    ]);
    const apiBase = stored.apiBaseUrl || DEFAULT_API_BASE;
    const headers = { 'Content-Type': 'application/json' };
    if (stored.authToken) {
      headers['Authorization'] = `Bearer ${stored.authToken}`;
    }

    const resp = await fetch(`${apiBase}/api/extension/summary`, { headers });
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
