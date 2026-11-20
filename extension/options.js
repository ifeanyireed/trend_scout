document.addEventListener('DOMContentLoaded', async () => {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'notificationsEnabled']);
  document.getElementById('apiBaseUrl').value = stored.apiBaseUrl || 'https://trendscout-api.vercel.app';
  document.getElementById('dashboardBaseUrl').value = stored.dashboardBaseUrl || 'https://trendscout-iota.vercel.app';
  document.getElementById('notificationsEnabled').checked = stored.notificationsEnabled !== false;

  document.getElementById('saveBtn').addEventListener('click', async () => {
    const clean = (v, fb) => String(v || fb).trim().replace(/\/+$/, '') || fb;
    await chrome.storage.local.set({
      apiBaseUrl: clean(document.getElementById('apiBaseUrl').value, 'https://trendscout-api.vercel.app'),
      dashboardBaseUrl: clean(document.getElementById('dashboardBaseUrl').value, 'https://trendscout-iota.vercel.app'),
      notificationsEnabled: document.getElementById('notificationsEnabled').checked,
    });
    chrome.runtime.sendMessage({ type: 'POLL_NOW' }).catch(() => {});
    const st = document.getElementById('statusMsg');
    st.textContent = 'Preferences saved!';
    setTimeout(() => { st.textContent = ''; }, 2500);
  });
});
