# TrendScout Studio — Chrome Extension (Manifest V3)

## 1. Architecture & Files

Located in `/Volumes/ReedBreedCC/trend_scout/extension`:
- `manifest.json` — Chrome Extension Manifest V3 configuration (`permissions`: `storage`, `notifications`, `alarms`, `tabs`; `host_permissions` for backend API).
- `background.js` — Service worker that polls `/api/extension/summary` via `chrome.alarms`, updates the extension toolbar badge count, and triggers native Chrome notifications (`chrome.notifications.create`) for new underdog discoveries, accelerating trends, completed research, scripts, and exports. Clicking any notification opens the exact target route in TrendScout Studio.
- `popup.html`, `popup.css`, `popup.js` — Compact studio entry point displaying:
  - Header with TrendScout logo, connection telemetry, and authenticated user badge.
  - Login view when unauthenticated (persisting JWT securely in `chrome.storage.local`).
  - Monitoring status bar (Active/Inactive, last scan timestamp, new discoveries count, urgent alerts count).
  - Live list of discovered opportunities (person name, recent achievement, underdog classification, trend stage badge, opportunity score, detection time) clicking directly into `/opportunities/:id`.
  - Quick action buttons: **Open TrendScout Studio**, **Emerging Stories**, **Major Opportunities**, **Watchlist**, **Trigger Scan / Refresh**, and **Settings**.
- `options.html`, `options.js` — Full extension settings interface to configure Backend API URL, Dashboard URL, notification category toggles, and polling frequency.
- `icons/` — `icon16.png`, `icon48.png`, `icon128.png`.

## 2. How to Load Unpacked Extension in Google Chrome

1. Open Google Chrome and navigate to `chrome://extensions/`.
2. Enable **Developer mode** using the toggle in the top-right corner.
3. Click **Load unpacked**.
4. Select the directory `/Volumes/ReedBreedCC/trend_scout/extension` (or extract `/Volumes/ReedBreedCC/trend_scout/storage/exports/trendscout-chrome-extension.zip`).
5. Pin **TrendScout Studio** in the Chrome toolbar and click the icon to open the popup.
