# TrendScout Studio — API Contracts

All API routes are prefixed with `/api` and return JSON. Authenticated routes accept a Bearer JWT token (`Authorization: Bearer <token>`) or session cookie.

## 1. Authentication (`/api/auth`)
- `POST /api/auth/login` — Authenticate with `{ email, password }`; returns `{ token, refreshToken, user }`.
- `POST /api/auth/refresh` — Exchange `{ refreshToken }` for a fresh `{ token, user }`.
- `POST /api/auth/logout` — Revoke current session.
- `GET /api/auth/me` — Retrieve authenticated user profile and monitoring state.

## 2. Dashboard & Extension Summary (`/api/dashboard`)
- `GET /api/dashboard/overview` — Returns real-time counts (`totalPeople`, `newDiscoveries`, `emergingStories`, `acceleratingStories`, `majorViralOpportunities`, `strongUnderdogs`, `storiesAwaitingResearch`, `scriptsGenerated`, `projectsInProduction`, `completedPackages`), `monitoringStatus`, `connectors`, and `recentNotifications`.
- `GET /api/extension/summary` — Dedicated compact payload for the Chrome Extension popup (`monitoringStatus`, `lastScanAt`, `newDiscoveriesCount`, `urgentAlertsCount`, `latestDiscoveries`, `unreadNotifications`).

## 3. Discovery & Opportunities (`/api/discoveries`)
- `GET /api/discoveries` — Query parameters: `stage`, `underdogClass`, `minScore`, `category`, `search`, `sort`. Returns list of discoveries joined with person, event, latest trend observation, underdog assessment, and 8-dimension opportunity score.
- `GET /api/discoveries/:id` — Full opportunity report including person biography, aliases, event details, trend observation history, videos, social posts, news articles, research claims, timeline, underdog assessment, and opportunity score rationale.
- `POST /api/discoveries/scan` — Enqueue and/or immediately trigger a live multi-source discovery run across enabled connectors.
- `POST /api/discoveries/:id/approve` — Approve an opportunity for documentary development and create/open its `documentary_projects` record.

## 4. Watchlist (`/api/watchlist`)
- `GET /api/watchlist` — List watched people/discoveries with latest momentum delta.
- `POST /api/watchlist` — Add `{ discoveryId, personId, notes, alertOnStageChange }`.
- `DELETE /api/watchlist/:id` — Remove item from watchlist.

## 5. Deep Research & Verification (`/api/research`)
- `GET /api/research/library` — List all `research_sources`, `research_claims`, verification stats, and research jobs.
- `GET /api/research/person/:personId` — Get full biography, `person_timelines`, `research_sources`, `research_claims`, contradictions report, and research gaps report.
- `POST /api/research/person/:personId/run` — Trigger `ResearchAgent` & `FactVerificationAgent` on a person/discovery.
- `POST /api/research/claims/:claimId/verify` — Update verification label (`Verified`, `Reported`, `Attributed`, `Interpretation`, `Unverified`, `Unknown`) or notes.

## 6. Documentaries, Scripts & Production Studio (`/api/projects`)
- `GET /api/projects` — List all documentary projects with script, visual timeline, audio, and export status.
- `POST /api/projects` — Create a documentary project from a discovery/person.
- `GET /api/projects/:id` — Retrieve complete project workspace (Overview, Script, Versions, References, Research, Visual Timeline, Character References, Images, Audio, Subtitles, Exports).
- `POST /api/projects/:id/script/generate` — Run `DocumentaryWriterAgent` to generate a source-linked long-form narration script (`targetDurationMinutes`, `style`).
- `PUT /api/projects/:id/script` — Save edited script narration and create a new `script_versions` revision.
- `POST /api/projects/:id/script/regenerate-paragraph` — Regenerate a specific section/paragraph grounded in research claims.
- `POST /api/projects/:id/script/approve` — Mark script as approved for visual & audio production.
- `POST /api/projects/:id/characters/generate` — Run `VisualPromptAgent` to generate character consistency sheets across age ranges and locations.
- `PUT /api/projects/:id/characters/:charId` — Edit character reference sheet or regenerate reference artwork.
- `POST /api/projects/:id/timeline/generate` — Segment narration into 5–10 second visual beats (120–240 segments for 20 min) with Google Flow prompts and character/location bindings.
- `PUT /api/projects/:id/timeline/:segmentId` — Update segment prompt, duration, or regenerate still image asset.
- `POST /api/projects/:id/images/generate-batch` — Generate visual frames for timeline segments.
- `POST /api/projects/:id/audio/generate` — Generate master or segment narration audio, measure exact duration, align visual segment timestamps, and generate `.srt` subtitles.
- `POST /api/projects/:id/audio/upload` — Upload custom narration audio file.
- `POST /api/projects/:id/export` — Build complete production ZIP package (`documentary-script.docx`, `narration.txt`, `visual-timeline.csv`, `image-prompts.txt`, `subtitles.srt`, `production-manifest.json`, `research-sources.md`, `fact-check-report.md`, images, audio).
- `POST /api/projects/:id/assemble-video` — Run modular FFmpeg 16:9 video assembler combining images, narration audio, and subtitles into an `.mp4` file.

## 7. Notifications, Reports, Settings & System Health
- `GET /api/notifications` — List notifications and email delivery logs.
- `POST /api/notifications/:id/read` — Mark notification read.
- `POST /api/notifications/read-all` — Mark all notifications read.
- `GET /api/reports` — Retrieve daily digest and weekly Top 10 underdog documentary opportunity reports.
- `POST /api/reports/generate` — Generate fresh intelligence digest and dispatch configured email alerts.
- `GET /api/settings` & `PUT /api/settings` — Read/update Discovery, Monitoring, AI, Notification, and Production preferences.
- `GET /api/system/health` — Connector health, worker heartbeat, persistent job queue telemetry (`queued`, `running`, `completed`, `failed`, `dead_letter`), and audit logs.
- `POST /api/system/jobs/:id/retry` — Retry failed or dead-letter background job.
