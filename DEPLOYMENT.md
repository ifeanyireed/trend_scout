# TrendScout Studio — Deployment & Operations Guide

## 1. Local & Cloud Startup

```bash
# Install dependencies across root, backend, and web
npm install

# Build backend, web dashboard, and package the Chrome extension
npm run build

# Start the cloud backend server + autonomous background worker + web studio
npm start
```

- **Backend API & Production Dashboard Server**: `http://localhost:4000`
- **Vite Dev Server (Development Mode)**: `http://localhost:5173` (proxies `/api` and `/storage` to `http://localhost:4000`)
- **Unpacked Chrome Extension Path**: `/Volumes/ReedBreedCC/trend_scout/extension`
- **Packaged Chrome Extension Archive**: `/Volumes/ReedBreedCC/trend_scout/storage/exports/trendscout-chrome-extension.zip`

## 2. Environment Variables (`.env`)

All secrets are stored strictly server-side:
- `PORT=4000`
- `DATABASE_URL=postgresql://...` (Neon Cloud PostgreSQL 18)
- `JWT_SECRET=<server-secret>`
- `STORAGE_DIR=./storage`
- `DASHBOARD_URL=https://your-trendscout.vercel.app`
- `GEMINI_API_KEY=<gemini-api-key>`
- `GEMINI_MODEL=gemini-3.8-flash`
- `YOUTUBE_API_KEY=` (Optional: upgrades YouTube RSS/public feed connector to full Data API v3 quota)
- `SMTP_HOST=`, `SMTP_PORT=`, `SMTP_USER=`, `SMTP_PASS=` (Optional: dispatches live SMTP emails in addition to persistent local email outbox)

## 3. Vercel Deployment

1. **Deploying the Web Studio to Vercel**:
   - Push or import this repository to Vercel.
   - Vercel automatically detects `vercel.json` and builds the Vite frontend from `web/` to `web/dist`.
   - In your Vercel Project Settings -> Environment Variables, add:
     - `VITE_API_BASE_URL`: The URL of your cloud backend API (e.g. `https://your-trendscout-api.up.railway.app` or your cloud server).
2. **Chrome Extension**:
   - In Chrome, open `chrome://extensions` and click **Options** on TrendScout Studio.
   - Enter your Vercel deployment URL (e.g. `https://your-trendscout.vercel.app`) as the **Web Studio Dashboard URL** and **Cloud Backend API URL**.
