const DEFAULT_API = 'https://trendscout-iota.vercel.app';
const DEFAULT_DASH = 'https://trendscout-iota.vercel.app';

async function getConfig() {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'authToken', 'refreshToken', 'userEmail']);
  const clean = (v, fb) => {
    const c = String(v || fb).trim().replace(/\/+$/, '');
    if (!c || c.includes('trendscout-api.vercel.app')) return fb;
    return c;
  };
  return {
    apiBase: clean(stored.apiBaseUrl, DEFAULT_API),
    dashBase: clean(stored.dashboardBaseUrl, DEFAULT_DASH),
    token: stored.authToken || '',
    refreshToken: stored.refreshToken || '',
    userEmail: stored.userEmail || 'Not signed in',
  };
}

async function authFetchPopup(path, options = {}) {
  const cfg = await getConfig();
  let token = cfg.token;
  const doReq = (tok) => {
    const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
    if (tok) headers['Authorization'] = `Bearer ${tok}`;
    return fetch(`${cfg.apiBase}${path}`, { ...options, headers });
  };
  let resp = await doReq(token);
  if (resp.status === 401 && cfg.refreshToken) {
    try {
      const refResp = await fetch(`${cfg.apiBase}/api/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken: cfg.refreshToken }),
      });
      if (refResp.ok) {
        const refData = await refResp.json();
        if (refData?.token) {
          token = refData.token;
          await chrome.storage.local.set({
            authToken: refData.token,
            refreshToken: refData.refreshToken || cfg.refreshToken,
            tokenExpiresAt: refData.expiresAt || '',
          });
          resp = await doReq(token);
        }
      }
    } catch (_) {}
  }
  return resp;
}

async function openDashboardRoute(routePath) {
  const { dashBase } = await getConfig();
  chrome.tabs.create({ url: `${dashBase}${routePath}` });
}

async function loadPopupData() {
  const { apiBase, userEmail } = await getConfig();
  document.getElementById('userLabel').textContent = userEmail;

  try {
    const resp = await authFetchPopup('/api/extension/summary');
    if (resp.status === 401) {
      document.getElementById('connDot').className = 'dot offline';
      document.getElementById('connText').textContent = 'Sign In Required';
      document.getElementById('loginBox').style.display = 'block';
      document.getElementById('discoveriesList').innerHTML =
        '<div style="padding:14px;text-align:center;color:#64748b;">Session expired or not signed in. Sign in above to view live opportunities.</div>';
      return;
    }
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
    const errEl = document.getElementById('loginError');
    if (errEl) errEl.style.display = 'none';
    try {
      const resp = await fetch(`${apiBase}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, rememberMe: true }),
      });
      const res = await resp.json().catch(() => ({}));
      if (resp.ok && res.token) {
        await chrome.storage.local.set({
          authToken: res.token,
          refreshToken: res.refreshToken,
          tokenExpiresAt: res.expiresAt || '',
          userEmail: res.user?.email || email,
        });
        document.getElementById('loginBox').style.display = 'none';
        chrome.runtime.sendMessage({ type: 'POLL_NOW' }).catch(() => {});
        loadPopupData();
      } else if (errEl) {
        errEl.textContent = res.error || 'Invalid credentials';
        errEl.style.display = 'block';
      }
    } catch (_) {
      if (errEl) {
        errEl.textContent = 'Cannot reach studio backend';
        errEl.style.display = 'block';
      }
    }
  });

  document.getElementById('logoutBtn').addEventListener('click', async () => {
    await chrome.storage.local.remove(['authToken', 'refreshToken', 'tokenExpiresAt', 'userEmail']);
    document.getElementById('userLabel').textContent = 'Not signed in';
    document.getElementById('loginBox').style.display = 'block';
    loadPopupData();
  });

  document.getElementById('scanNowBtn').addEventListener('click', async () => {
    const btn = document.getElementById('scanNowBtn');
    btn.textContent = 'Scanning...';
    try {
      await authFetchPopup('/api/discoveries/scan', { method: 'POST', body: '{}' });
      chrome.runtime.sendMessage({ type: 'POLL_NOW' }).catch(() => {});
      await loadPopupData();
    } finally {
      btn.textContent = 'Scan Now';
    }
  });
});
