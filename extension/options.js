document.addEventListener('DOMContentLoaded', async () => {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'notificationsEnabled']);
  const savedApi = stored.apiBaseUrl && !stored.apiBaseUrl.includes('trendscout-api.vercel.app')
    ? stored.apiBaseUrl
    : 'https://trendscout-iota.vercel.app';
  document.getElementById('apiBaseUrl').value = savedApi;
  document.getElementById('dashboardBaseUrl').value = stored.dashboardBaseUrl || 'https://trendscout-iota.vercel.app';
  document.getElementById('notificationsEnabled').checked = stored.notificationsEnabled !== false;

  document.getElementById('saveBtn').addEventListener('click', async () => {
    const clean = (v, fb) => String(v || fb).trim().replace(/\/+$/, '') || fb;
    await chrome.storage.local.set({
      apiBaseUrl: clean(document.getElementById('apiBaseUrl').value, 'https://trendscout-iota.vercel.app'),
      dashboardBaseUrl: clean(document.getElementById('dashboardBaseUrl').value, 'https://trendscout-iota.vercel.app'),
      notificationsEnabled: document.getElementById('notificationsEnabled').checked,
    });
    chrome.runtime.sendMessage({ type: 'POLL_NOW' }).catch(() => {});
    const st = document.getElementById('statusMsg');
    st.textContent = 'Preferences saved!';
    setTimeout(() => { st.textContent = ''; }, 2500);
  });
});
