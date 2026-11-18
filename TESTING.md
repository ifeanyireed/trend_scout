# TrendScout Studio — Testing & Verification Guide

## Automated Verification Suite

Run the full end-to-end test suite from the workspace root:
```bash
npm test
```

## Covered Workflows
1. **Discovery & Connector Tests**: Verifies source polling (`google_news`, `wikipedia`, `reddit`, `youtube`, `google_trends`), person deduplication, alias resolution, event creation, trend classification, and 8-dimension evidence-linked opportunity scoring.
2. **Deep Research & Fact Verification Tests**: Verifies biographical synthesis, chronological timeline ordering (`person_timelines`), claim-to-source linkage (`research_claims` -> `research_sources`), verification status labeling, contradiction detection, and research gaps reporting.
3. **Documentary Script Tests**: Verifies 20-minute long-form script generation, source reference mapping (`script_references`), paragraph regeneration, version history (`script_versions`), and script approval.
4. **Visual Timeline & Audio Production Tests**: Verifies character reference creation (`character_references`), 5–10 second visual segmentation (`visual_segments`), audio generation/duration alignment (`audio_assets`), and `.srt` subtitle creation (`subtitle_assets`).
5. **Production Export & FFmpeg Assembly Tests**: Verifies `.docx`, `.txt`, `.csv`, `.srt`, `production-manifest.json`, `.zip` bundle creation, and 16:9 `.mp4` video assembly via FFmpeg.
6. **Chrome Extension & Cloud Queue Tests**: Verifies `/api/extension/summary` authentication, manifest structure, background job idempotency, retry backoff, and notification deduplication.
