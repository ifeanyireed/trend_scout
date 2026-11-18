document.addEventListener('DOMContentLoaded', async () => {
  const stored = await chrome.storage.local.get(['apiBaseUrl', 'dashboardBaseUrl', 'notificationsEnabled']);
  document.getElementById('apiBaseUrl').value = stored.apiBaseUrl || 'https://trendscout-api.vercel.app';
  document.getElementById('dashboardBaseUrl').value = stored.dashboardBaseUrl || 'https://trendscout-iota.vercel.app';
  document.getElementById('notificationsEnabled').checked = stored.notificationsEnabled !== false;

  document.getElementById('saveBtn').addEventListener('click', async () => {
    await chrome.storage.local.set({
      apiBaseUrl: document.getElementById('apiBaseUrl').value.trim(),
      dashboardBaseUrl: document.getElementById('dashboardBaseUrl').value.trim(),
      notificationsEnabled: document.getElementById('notificationsEnabled').checked,
    });
    const st = document.getElementById('statusMsg');
    st.textContent = 'Preferences saved!';
    setTimeout(() => { st.textContent = ''; }, 2500);
  });
});
