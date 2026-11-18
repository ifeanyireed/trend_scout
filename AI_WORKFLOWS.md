# TrendScout Studio — AI & Autonomous Agent Workflows

## 1. Ten Autonomous Agents

1. **`DiscoveryAgent`**: Polls enabled connectors (`google_news`, `reddit`, `youtube`, `google_trends`, `wikipedia`), extracts candidate people and events, and persists raw source evidence (`news_articles`, `videos`, `social_posts`).
2. **`PersonResolutionAgent`**: Normalizes candidate names, resolves aliases (`person_aliases`), links `people` to `events` via `person_events`, and prevents duplicate person records.
3. **`TrendMonitoringAgent`**: Captures time-series snapshots in `trend_observations`, computing `views_per_hour`, `views_per_day`, `channel_relative_perf`, `independent_coverage`, and classifying stories into:
   - `Early Whisper`
   - `Emerging`
   - `Accelerating`
   - `Major Viral Opportunity`
   - `Saturated`
   - `Declining`
   - `Uncertain`
4. **`UnderdogAssessmentAgent`**: Evaluates documented biographical evidence across origins, early ambitions, disadvantages, rejections, obstacles, turning points, and recent breakthroughs. Assigns classification (`Strong underdog`, `Emerging underdog`, `Comeback story`, `Breakthrough story`, `Unexpected champion`, `Extraordinary achievement`, `Insufficient evidence`) and calculates the 8-dimension `opportunity_scores` with explicit JSON rationales.
5. **`ResearchAgent`**: Queries Wikipedia/Wikimedia sections, news archives, and cited references to construct a complete biography, chronological `person_timelines`, and granular `research_claims` linked to `research_sources`.
6. **`FactVerificationAgent`**: Audits every extracted claim against source reliability and cross-source corroboration, labeling each claim as `Verified`, `Reported`, `Attributed`, `Interpretation`, `Unverified`, or `Unknown`, and generating the `contradictions_report` and `research_gaps_report`.
7. **`DocumentaryWriterAgent`**: Generates continuous 15–25 minute narrative documentary scripts (`documentary_scripts` & `script_versions`) structured into 6 narrative acts (The Opening Hook, Humble Origins & Early Spark, The Crucible of Rejection & Setbacks, The Turning Point, The Against-All-Odds Breakthrough, Legacy & Meaning). Every paragraph links to `script_references` (`claim_id` and `source_id`).
8. **`VisualPromptAgent`**:
   - Creates `character_references` sheets (subject at childhood, youth, and breakthrough age; supporting figures; key locations; visual style and color palette).
   - Splits narration into 5–10 second continuous segments (`visual_segments`, ~120–240 segments for a 20-minute script) with detailed Google Flow image prompts and character/location links.
9. **`ProductionAgent`**: Generates visual frame assets, synthesizes/aligns narration audio (`audio_assets`), generates `.srt` subtitles (`subtitle_assets`), builds the ZIP export package (`production_exports`), and runs the modular FFmpeg 16:9 video assembler.
10. **`NotificationAgent`**: Dispatches deduplicated Chrome extension alerts, in-app notifications, and HTML/text emails (`Early whisper`, `Emerging`, `Accelerating`, `Major viral`, `Daily digest`, `Weekly Top 10`, `Research completion`, `Script completion`, `Production package ready`).

## 2. Pluggable AI Provider Architecture
- Supports OpenAI (`OPENAI_API_KEY`), Anthropic (`ANTHROPIC_API_KEY`), or Gemini (`GEMINI_API_KEY`) configured server-side via Settings or `.env`.
- Includes a deterministic, source-grounded synthesis engine that strictly uses real retrieved `research_sources` and `research_claims` so the entire discovery, research, scriptwriting, and visual timeline pipeline operates with zero fabricated facts even before external LLM keys are added.
