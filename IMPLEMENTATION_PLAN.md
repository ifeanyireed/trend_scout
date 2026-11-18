# TrendScout Studio — Implementation Plan

## Phase 0: Audit & Architectural Foundation
- Inspect repository state (greenfield workspace verified).
- Establish monorepo structure (`backend/`, `web/`, `extension/`).
- Document `ARCHITECTURE.md`, `DATABASE_SCHEMA.md`, `API_CONTRACTS.md`, `SOURCE_CONNECTORS.md`, `AI_WORKFLOWS.md`, `CHROME_EXTENSION.md`, `TESTING.md`, `DEPLOYMENT.md`, `KNOWN_LIMITATIONS.md`, and `PROGRESS.md`.
- Implement SQLite relational database with all 35 required tables, foreign keys, uniqueness constraints, and indexes.

## Phase 1: Trend Discovery & Underdog Intelligence
- Build multi-platform source connector framework (`googleNewsConnector`, `redditConnector`, `youtubeConnector`, `wikipediaConnector`, `googleTrendsConnector`).
- Implement `PersonResolutionAgent`, `DiscoveryAgent`, `TrendMonitoringAgent`, and `UnderdogAssessmentAgent`.
- Calculate 8 transparent, evidence-linked scores (`underdog_strength`, `transformation`, `achievement_significance`, `audience_momentum`, `documentary_potential`, `timeliness`, `evidence_confidence`, `saturation_risk`).
- Build Dashboard, Discover, Trending Now, Emerging Stories, Underdog Opportunities, Opportunity Detail, and Watchlist APIs and UI pages.

## Phase 2: Deep Research & Fact Verification
- Implement `ResearchAgent` and `FactVerificationAgent` fetching real biographical data from Wikipedia, Wikimedia, news feeds, and cross-referenced reports.
- Build full biography, chronological life timeline (`person_timelines`), evidence-linked claims (`research_claims` with `Verified`, `Reported`, `Attributed`, `Interpretation`, `Unverified`, `Unknown` labels), contradiction detection, and research gaps analysis.
- Connect Research Workspace and Research Library pages.

## Phase 3: Documentary Script Engine
- Implement `DocumentaryWriterAgent` generating long-form (15–25 min configurable, 20 min default) continuous storytelling documentary scripts grounded in verified research claims.
- Support paragraph regeneration, manual script editing, version history (`script_versions`), revision comparison, script approval, and clean narration text export.

## Phase 4: Visual Timeline, Google Flow & Audio Production
- Implement `VisualPromptAgent` generating character reference sheets (`character_references` across ages, clothing, style, and color palette) and 5–10 second visual segments (`visual_segments`) aligned to narration timing.
- Build Google Flow batch prompt exporter and adapter interface.
- Build Audio Studio supporting TTS synthesis (OpenAI/ElevenLabs/macOS `say` / WAV generator fallback), audio file upload, duration measurement, and `.srt` subtitle generation.

## Phase 5: Chrome Extension (Manifest V3)
- Build unpacked Chrome Extension in `/extension` and packaged `.zip` archive.
- Implement `manifest.json`, `background.js` service worker, `popup.html`/`popup.js`, `options.html`/`options.js`, session authentication, live discovery cards, badge counts, Chrome notifications, and deep links into TrendScout Studio.

## Phase 6: Cloud Operations, Persistent Queue & Notifications
- Implement persistent database-backed `JobQueue` with priorities, retries, exponential backoff, worker heartbeats, dead-letter handling, and idempotency.
- Implement `NotificationAgent` with email delivery (SMTP + local verifiable outbox) and configurable alert categories/digests.
- Build System Health, Notifications, Reports, and Settings pages.

## Phase 7: Production Package Export & FFmpeg Video Assembly
- Implement `ProductionAgent` and `ExportService` generating `documentary-script.docx`, `narration.txt`, `visual-timeline.csv`, `image-prompts.txt`, `subtitles.srt`, `production-manifest.json`, `research-sources.md`, `fact-check-report.md`, character/segment SVG/PNG visual assets, audio files, and packaged `.zip` archives.
- Implement modular FFmpeg 16:9 `.mp4` video assembly combining generated images, narration audio, and subtitles.
- Run end-to-end verification tests across all phases.
