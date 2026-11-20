const DEFAULT_API = 'https://trendscout-api.vercel.app';
const DEFAULT_DASH = 'https://trendscout-iota.vercel.app';

async function getConfig() {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'authToken', 'userEmail']);
  const clean = (v, fb) => String(v || fb).trim().replace(/\/+$/, '') || fb;
  return {
    apiBase: clean(stored.apiBaseUrl, DEFAULT_API),
    dashBase: clean(stored.dashboardBaseUrl, DEFAULT_DASH),
    token: stored.authToken || '',
    userEmail: stored.userEmail || 'producer@trendscout.studio',
  };
}

async function openDashboardRoute(routePath) {
  const { dashBase } = await getConfig();
  chrome.tabs.create({ url: `${dashBase}${routePath}` });
}

async function loadPopupData() {
  const { apiBase, token, userEmail } = await getConfig();
  document.getElementById('userLabel').textContent = userEmail;

  try {
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const resp = await fetch(`${apiBase}/api/extension/summary`, { headers });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();

    document.getElementById('connDot').className = 'dot';
    document.getElementById('connText').textContent = 'Cloud Active';
    document.getElementById('monState').textContent = data.monitoringActive ? 'ACTIVE' : 'PAUSED';
    document.getElementById('discCount').textContent = String(data.newDiscoveriesCount ?? 0);
    document.getElementById('alertCount').textContent = String(data.urgentAlertsCount ?? 0);

    if (data.lastScanAt) {
      const dt = new Date(data.lastScanAt);
      if (!isNaN(dt.getTime())) {
        document.getElementById('lastScanLabel').textContent = `Scanned ${dt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
      }
    }

    const container = document.getElementById('discoveriesList');
    container.innerHTML = '';

    const items = Array.isArray(data.latestDiscoveries) ? data.latestDiscoveries : [];
    if (items.length === 0) {
      container.innerHTML = '<div style="padding:16px;text-align:center;color:#64748b;">No discoveries yet. Click Scan Now.</div>';
      return;
    }

    for (const item of items) {
      const card = document.createElement('div');
      card.className = 'card';
      card.innerHTML = `
        <div class="card-top">
          <div class="person-name">${escapeHtml(item.personName)}</div>
          <div class="score-badge">${Number(item.overallScore || 90).toFixed(1)}</div>
        </div>
        <div class="achievement">${escapeHtml(item.achievement || item.headline)}</div>
        <div class="tags">
          <span class="tag stage">${escapeHtml(item.trendStage)}</span>
          <span class="tag">${escapeHtml(item.underdogClassification)}</span>
          <span class="tag">${escapeHtml(item.eventDate || '')}</span>
        </div>
      `;
      card.addEventListener('click', () => {
        openDashboardRoute(`/opportunities/${encodeURIComponent(item.id)}`);
      });
      container.appendChild(card);
    }
  } catch (err) {
    document.getElementById('connDot').className = 'dot offline';
    document.getElementById('connText').textContent = 'Offline';
    document.getElementById('discoveriesList').innerHTML =
      `<div style="padding:14px;text-align:center;color:#ef4444;">Cannot reach backend at ${escapeHtml(apiBase)}. Verify server is running.</div>`;
  }
}

function escapeHtml(str) {
  return String(str || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

document.addEventListener('DOMContentLoaded', () => {
  loadPopupData();

  document.getElementById('openStudioBtn').addEventListener('click', () => openDashboardRoute('/'));
  document.getElementById('openEmergingBtn').addEventListener('click', () => openDashboardRoute('/emerging'));
  document.getElementById('openViralBtn').addEventListener('click', () => openDashboardRoute('/underdogs'));
  document.getElementById('openWatchlistBtn').addEventListener('click', () => openDashboardRoute('/watchlist'));
  document.getElementById('openSettingsBtn').addEventListener('click', () => {
    if (chrome.runtime.openOptionsPage) {
      chrome.runtime.openOptionsPage();
    } else {
      openDashboardRoute('/settings');
    }
  });

  document.getElementById('toggleAuthBtn').addEventListener('click', () => {
    const box = document.getElementById('loginBox');
    box.style.display = box.style.display === 'block' ? 'none' : 'block';
  });

  document.getElementById('loginSubmitBtn').addEventListener('click', async () => {
    const { apiBase } = await getConfig();
    const email = document.getElementById('emailInput').value.trim();
    const password = document.getElementById('passInput').value;
    try {
      const resp = await fetch(`${apiBase}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      if (resp.ok) {
        const res = await resp.json();
        await chrome.storage.local.set({
          authToken: res.token,
          refreshToken: res.refreshToken,
          userEmail: res.user?.email || email,
        });
        document.getElementById('loginBox').style.display = 'none';
        loadPopupData();
      }
    } catch (_) {}
  });

  document.getElementById('logoutBtn').addEventListener('click', async () => {
    await chrome.storage.local.remove(['authToken', 'refreshToken']);
    document.getElementById('userLabel').textContent = 'Signed out (Guest Read)';
    document.getElementById('loginBox').style.display = 'none';
  });

  document.getElementById('scanNowBtn').addEventListener('click', async () => {
    const btn = document.getElementById('scanNowBtn');
    btn.textContent = 'Scanning...';
    const { apiBase, token } = await getConfig();
    try {
      const headers = { 'Content-Type': 'application/json' };
      if (token) headers['Authorization'] = `Bearer ${token}`;
      await fetch(`${apiBase}/api/discoveries/scan`, { method: 'POST', headers, body: '{}' });
      chrome.runtime.sendMessage({ type: 'POLL_NOW' }).catch(() => {});
      await loadPopupData();
    } finally {
      btn.textContent = 'Scan Now';
    }
  });
});
