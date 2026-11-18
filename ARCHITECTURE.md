# TrendScout Studio — System Architecture (Go + PostgreSQL + React + Chrome MV3)

## 1. System Overview

TrendScout Studio is an integrated AI documentary discovery, research, and production platform composed of three tightly coupled systems:
1. **Chrome Extension (Manifest V3)**: A lightweight, installable browser entry point providing real-time discovery alerts, status telemetry, quick opportunity scanning, and direct deep-linking into the studio dashboard.
2. **TrendScout Web Dashboard (React + TypeScript + Vite + Tailwind)**: The primary operational studio interface with 21 functional views covering discovery, trend velocity, underdog intelligence, deep biographical research, documentary script authoring, visual timeline segmentation (5–10s intervals for Google Flow), character reference consistency, audio/narration timing, and production package assembly.
3. **Cloud Backend & Autonomous Agent Engine (Go 1.26 + PostgreSQL 18 via `pgx/v5` + Persistent Job Queue + FFmpeg)**: Continuously monitors public sources (Google News RSS, Reddit public JSON, Wikipedia/Wikimedia APIs, YouTube public RSS/data feeds, Google Trends RSS, and optional API-key enhanced connectors), resolves people and events, computes evidence-linked underdog and momentum scores, runs multi-stage research and script pipelines, and packages complete documentary exports (including `.docx`, `.csv`, `.srt`, `.zip`, and FFmpeg `.mp4` video assembly).

---

## 2. Monorepo Structure

```
/Volumes/ReedBreedCC/trend_scout/
├── .env                          # Server-side configuration (PostgreSQL DATABASE_URL, JWT_SECRET, PORT)
├── package.json                  # Workspace orchestration & root scripts
├── backend/                      # Go 1.26 Cloud Backend, API Server & Autonomous Workers
│   ├── go.mod & go.sum
│   ├── cmd/
│   │   └── server/
│   │       └── main.go           # HTTP API Server + Background Worker & Scheduler entrypoint
│   └── internal/
│       ├── db/
│       │   ├── postgres.go       # PostgreSQL connection pool (pgx/v5 stdlib), migrations (34 tables) & seed
│       ├── connectors/
│       │   └── connectors.go     # Google News, Wikipedia, YouTube, Reddit & Google Trends live connectors
│       ├── agents/
│       │   └── agents.go         # All 10 Autonomous Agents (Discovery, Trend, Person, Underdog, Research, FactCheck, Writer, VisualPrompt, Production, Notification)
│       ├── queue/
│       │   └── queue.go          # Persistent PostgreSQL-backed Job Queue (`FOR UPDATE SKIP LOCKED`), heartbeats, retries & scheduler
│       ├── services/
│       │   └── export.go         # OpenXML .docx generator, CSV, SRT, ZIP packager, WAV/TTS audio & FFmpeg 16:9 MP4 builder
│       └── api/
│           ├── handlers.go       # Typed REST API endpoints for Web & Chrome Extension
│           └── e2e_test.go       # End-to-end automated integration test suite against PostgreSQL
├── web/                          # React + TypeScript + Tailwind Web Studio Dashboard
│   └── src/                      # All 21 required studio pages & components
├── extension/                    # Chrome Extension (Manifest V3)
│   ├── manifest.json
│   ├── background.js             # Service worker polling, badge count & Chrome notifications
│   ├── popup.html & popup.js     # Live status, discoveries, alerts & deep links
│   ├── options.html & options.js # Extension API URL, polling & notification preferences
│   └── icons/                    # Extension icons (16, 48, 128)
└── storage/                      # Persistent media, exports, audio, images, and email outbox
```

---

## 3. Core Architectural Principles

1. **Zero Browser Dependency for Monitoring**: The Chrome extension never runs heavy discovery scrapes. All scheduled discovery, trend velocity snapshots, and research pipelines execute in the Go backend worker queue (`background_jobs` table in PostgreSQL).
2. **Evidence-Grounded AI & Scoring**: Every underdog score, biographical timeline entry, and documentary script paragraph links explicitly to `research_sources` and `research_claims` in PostgreSQL.
3. **5–10 Second Visual Timeline Engine**: Documentary scripts are segmented by actual or measured speech duration into 5–10 second visual beats (120–240 segments for a 20-minute documentary), complete with subject age-matched character references and Google Flow-ready prompts.
4. **Server-Side Security**: All AI, YouTube, News, Database, and SMTP credentials remain strictly on the Go server (`user_settings` / `.env`) and are never exposed to the Chrome extension or browser bundle.
