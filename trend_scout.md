# TRENDScout STUDIO
# COMPLETE BUILD INSTRUCTION FOR THE CODING AGENT
## Chrome Extension + Web Dashboard + Cloud AI Discovery and Documentary Production System

### 1. YOUR ASSIGNMENT

Build the complete TrendScout Studio application, including:

1. A fully functional Chrome Extension.
2. A complete web dashboard.
3. A cloud-hosted backend.
4. Autonomous background discovery agents.
5. An underdog-story intelligence engine.
6. Multi-platform trend monitoring.
7. AI-powered research and fact verification.
8. Documentary script generation.
9. Visual timeline and image prompt generation.
10. Character reference management.
11. Audio and narration preparation.
12. Production package generation and export.
13. Email notifications and monitoring.
14. Persistent database and file storage.

**The Chrome extension is a mandatory first-class product component. Do not leave it as a future feature.**

The application must work as one connected system.

The user must be able to install the Chrome extension, click it, open TrendScout Studio, discover stories, investigate opportunities, generate documentaries and manage production from the browser.

---

# 2. REQUIRED PRODUCT EXPERIENCE

The intended user experience is:

1. Install TrendScout from a locally loadable Chrome extension package during development.
2. Click the TrendScout icon in Chrome.
3. See a compact popup with recent discoveries, alerts and monitoring status.
4. Click "Open TrendScout Studio".
5. The full dashboard opens in a new browser tab.
6. Browse trending people and underdog opportunities.
7. Open an individual opportunity report.
8. Review the evidence, trend performance and life-story research.
9. Approve the opportunity for documentary development.
10. Generate a complete documentary script.
11. Open its production workspace.
12. Review the narration and timestamped visual timeline.
13. Manage character references, images and audio.
14. Export the production package.

The cloud backend continues operating independently of the browser.

---

# 3. CHROME EXTENSION — BUILD THIS ACTUALLY

Use Chrome Manifest V3.

Create a complete extension project with its own manifest, background service worker, popup interface, options/settings interface, icons, authentication handling and communication with the TrendScout backend.

### 3.1 Extension structure

Implement:

- `manifest.json`
- Background service worker.
- Popup UI.
- Extension settings page.
- Authentication/session integration.
- API client.
- Notification handling.
- Extension-to-dashboard navigation.
- Error and connection handling.
- Build and packaging scripts.

Use a modular React-based implementation if appropriate.

Do not put long-running discovery jobs inside the Chrome extension. Chrome may be closed or suspended. The cloud backend owns continuous monitoring.

### 3.2 Extension popup

Design a polished, lightweight popup.

Include:

**Header**
- TrendScout logo.
- Application name.
- Connection status.
- User/account indicator.

**Monitoring status**
- Monitoring active/inactive.
- Last successful scan.
- Number of new discoveries.
- Number of urgent alerts.

**Latest discoveries**
Show a short list of recently detected opportunities.

Each item should display:
- Person's name.
- Recent achievement.
- Underdog classification.
- Trend stage.
- Opportunity score.
- Detection time.

**Quick actions**
- Open TrendScout Studio.
- View Emerging Stories.
- View Major Opportunities.
- Open Watchlist.
- Refresh latest information.
- Settings.

The popup must retrieve actual backend data.

Do not use hardcoded discovery cards.

### 3.3 Chrome notifications

Implement Chrome notifications for important events where the user has granted notification permission.

Examples:
- A new strong underdog opportunity.
- A story entering accelerating status.
- A major viral opportunity.
- A completed deep research report.
- A completed documentary script.
- A production package ready for export.

Allow notification preferences to be controlled by the user.

Avoid duplicate notifications.

Clicking a notification should open the relevant TrendScout dashboard page.

### 3.4 Extension authentication

Implement secure authentication between the extension and web dashboard.

Requirements:
- Login and logout.
- Session persistence using an appropriate secure design.
- Token refresh where supported.
- Expired-session handling.
- Unauthorized request handling.
- Secure API communication over HTTPS in production.

Do not expose AI API keys or other backend secrets in extension JavaScript.

The extension must never contain privileged API credentials.

### 3.5 Extension-to-dashboard navigation

Implement direct links to:

- Dashboard overview.
- Trending Now.
- Emerging Stories.
- Opportunity reports.
- Watchlist.
- Research library.
- My Documentaries.
- Production Studio.
- Settings.

Opening an opportunity from the extension should take the user directly to that opportunity's detail page.

### 3.6 Extension installation and testing

Provide:
- Development build.
- Production build.
- Chrome-compatible package.
- Installation instructions for loading the unpacked extension.
- Extension permissions documentation.
- Automated extension tests where practical.

Do not claim the extension is published on the Chrome Web Store unless it has actually been submitted and approved.

---

# 4. TRENDScout WEB DASHBOARD

Build a complete responsive web application.

The extension popup is only the entry point. The web dashboard is the main working environment.

### Main navigation

1. Overview.
2. Discover.
3. Trending Now.
4. Emerging Stories.
5. Underdog Opportunities.
6. Research Library.
7. Watchlist.
8. My Documentaries.
9. Production Studio.
10. Notifications.
11. Reports.
12. Settings.

### Dashboard overview

Display real operational information:

- Total discovered people.
- Newly discovered opportunities.
- Emerging stories.
- Accelerating stories.
- Major viral opportunities.
- Strong underdog stories.
- Stories awaiting research.
- Scripts generated.
- Projects in production.
- Completed packages.
- Background monitoring status.
- Connector health.
- Latest notifications.

Every metric must come from real application data.

---

# 5. TREND DISCOVERY — FIRST IMPLEMENTATION PRIORITY

Build this first as the first complete, testable product capability.

It must find people and events, not simply keywords.

Monitor supported sources including:

- YouTube.
- X/Twitter.
- TikTok.
- Instagram.
- Reddit.
- Google Trends.
- News publishers and search services.

Use official APIs or permitted providers. Verify access, cost, quotas and restrictions before committing to integrations.

For every discovery, identify:

- Person.
- Event.
- Event date.
- Achievement.
- Relevant source.
- First detected time.
- Current attention signals.
- Independent coverage.
- Research status.

### Trend classifications

- Early Whisper.
- Emerging.
- Accelerating.
- Major Viral Opportunity.
- Saturated.
- Declining.
- Uncertain.

Use historical snapshots to measure growth. Never invent historical view counts or viral metrics.

### YouTube intelligence

Collect available video and channel metrics.

Calculate:
- Views per hour.
- Views per day.
- Channel-relative performance.
- Independent creator coverage.
- Coverage growth.
- Engagement indicators.

A high view count alone does not prove virality.

### Underdog analysis

For every promising person, investigate:

- Origins and childhood.
- Background.
- Early ambitions.
- Disadvantages.
- Rejections.
- Failures.
- Obstacles.
- Turning points.
- Career progression.
- Achievements.
- Recent victory or breakthrough.

Classify the person as:
- Strong underdog.
- Emerging underdog.
- Comeback story.
- Breakthrough story.
- Unexpected champion.
- Extraordinary achievement.
- Insufficient evidence.

Require supporting sources for factual claims.

### Scoring

Provide separate scores for:

- Underdog strength.
- Transformation.
- Achievement significance.
- Audience momentum.
- Documentary potential.
- Timeliness.
- Evidence confidence.
- Saturation risk.

Show why each score was assigned.

Do not manufacture a dramatic life story to increase a score.

---

# 6. RESEARCH AND FACT VERIFICATION

For every qualified opportunity, conduct deeper research.

Build:
- Full biography.
- Chronological life timeline.
- Evidence-linked research report.
- Recent event verification.
- Independent news coverage.
- Audience-interest report.
- Research gaps report.
- Contradiction report.

Each fact must be labelled appropriately:

- Verified.
- Reported.
- Attributed.
- Interpretation.
- Unverified.
- Unknown.

Store source URLs, publishers, publication dates, retrieval dates and relevant claims.

The AI must not invent sources, quotations, dates, achievements or personal hardships.

---

# 7. DOCUMENTARY SCRIPT GENERATION

Generate professional long-form YouTube documentaries.

Default target: approximately 20 minutes.

Allow configurable durations, including 15–25 minutes.

The narrative must cover the subject's life from their beginnings through their struggles, turning points, achievements and recent breakthrough.

Use:
- A strong opening hook.
- Emotional storytelling.
- Suspense.
- Narrative progression.
- Natural transitions.
- Carefully timed reveals.
- A satisfying conclusion.

The narration must be continuous storytelling, not conventional screenplay scenes.

Every factual section must have supporting research references.

Allow:
- Generate script.
- Regenerate selected paragraphs.
- Edit script.
- Approve script.
- Save versions.
- Compare revisions.
- Export narration text.

---

# 8. VISUAL TIMELINE AND GOOGLE FLOW WORKFLOW

This is a critical production requirement.

The user wants still images changing approximately every 5–10 seconds while the narration continues.

For a 20-minute documentary, this means approximately 120–240 visual segments, depending on narration pacing.

For every segment generate:

- Segment ID.
- Start timestamp.
- End timestamp.
- Duration.
- Exact narration passage.
- Detailed image prompt.
- Character reference.
- Location reference.
- Image generation status.
- Image asset reference.

Do not create conventional scene breaks that interrupt continuous narration.

### Character consistency

Before generating the full image timeline, create a character reference package:

- Main subject at different ages.
- Appearance.
- Hair.
- Clothing.
- Body proportions.
- Supporting characters where relevant.
- Important locations.
- Environmental references.
- Visual style.
- Colour palette.

Maintain visual consistency across the documentary.

### Google Flow integration

Investigate the current supported Google Flow workflow and available integrations.

Do not assume a public bulk-generation API exists.

Build an export format that lets the user conveniently copy or retrieve image prompts and reference assets for manual Google Flow use.

If a supported integration becomes available, implement it behind a separate adapter.

Never make the rest of TrendScout dependent on an unverified Google Flow integration.

---

# 9. AUDIO AND NARRATION

Provide:

- Clean narration text.
- Audio generation integration through a configurable provider.
- Audio file upload.
- Audio playback.
- Segment-level audio management.
- Narration duration measurement.
- Subtitle generation.
- Audio asset management.

Prefer generating or importing narration audio before finalising precise visual timing.

Use actual audio duration or speech alignment where available.

Do not assume every paragraph takes the same amount of time to narrate.

---

# 10. PRODUCTION STUDIO

Create a complete production workspace for each documentary.

Tabs:

1. Overview.
2. Script.
3. Research.
4. Visual Timeline.
5. Character References.
6. Images.
7. Audio.
8. Subtitles.
9. Export.

The user must be able to move through the production workflow without manually searching for assets.

### Export package

Produce:

- `documentary-script.docx` or an equivalent editable document.
- `narration.txt`.
- `visual-timeline.csv`.
- `image-prompts.txt`.
- `subtitles.srt`.
- `production-manifest.json`.
- `research-sources.md`.
- `fact-check-report.md`.
- Character reference images.
- Generated image assets.
- Narration audio.
- Thumbnail concept.
- YouTube title suggestions.
- YouTube description.
- Music and sound design suggestions.

Package files in an organised ZIP archive.

### Optional video assembly

Provide a modular FFmpeg-based assembly engine.

Where all required assets are available, it should be capable of combining:

- Still images.
- Narration audio.
- Background music.
- Transitions.
- Subtitles.
- Intro/outro assets.
- 16:9 video output.

Do not mark a project complete until the required production assets have actually been generated or imported.

---

# 11. CLOUD BACKEND AND AUTONOMOUS AGENTS

The platform must continue operating while Chrome is closed and the user's computer is off.

Implement cloud-based background services.

### Required agents and workers

**Discovery Agent**
Continuously discovers new people and events.

**Trend Monitoring Agent**
Rechecks qualified subjects and measures changes in public attention.

**Person Resolution Agent**
Deduplicates identities and connects aliases.

**Underdog Assessment Agent**
Investigates narrative potential and evaluates supporting evidence.

**Research Agent**
Conducts deeper biographical research.

**Fact Verification Agent**
Checks claims, dates, contradictions and source quality.

**Documentary Writer Agent**
Generates source-grounded narration.

**Visual Prompt Agent**
Creates character references and timestamped image prompts.

**Production Agent**
Organises assets and creates export packages.

**Notification Agent**
Sends email and extension alerts.

These agents should be orchestrated through a reliable job system, not implemented as independent uncontrolled loops.

### Background job requirements

Implement:
- Persistent queue.
- Scheduled execution.
- Retries.
- Exponential backoff where appropriate.
- Failure tracking.
- Job priorities.
- Idempotency.
- Duplicate prevention.
- Recovery of unfinished jobs.
- Worker heartbeat.
- Execution logs.
- Job timeouts.
- Dead-letter handling.

Use independently running workers where necessary.

Do not rely on a browser tab being open or a long-running serverless request.

---

# 12. EMAIL AND NOTIFICATIONS

Support:

- Early whisper alerts.
- Emerging opportunities.
- Accelerating stories.
- Major viral opportunities.
- Daily digest.
- Weekly top 10 documentary opportunities.
- Research completion.
- Script completion.
- Production package completion.
- System failure notifications.

Each alert should include the subject, event, date, underdog assessment, momentum evidence, opportunity score, saturation status and link to the full report.

Prevent duplicate alerts.

Make notification preferences configurable.

---

# 13. DATABASE AND DATA MODEL

Design a relational database with appropriate indexes, relationships and migrations.

At minimum, include:

- users
- user_settings
- connected_sources
- source_runs
- source_metrics
- people
- person_aliases
- events
- person_events
- discoveries
- trend_observations
- videos
- social_posts
- news_articles
- research_sources
- research_claims
- person_timelines
- underdog_assessments
- opportunity_scores
- watchlists
- notifications
- research_jobs
- documentary_projects
- documentary_scripts
- script_versions
- script_references
- visual_segments
- character_references
- image_assets
- audio_assets
- subtitle_assets
- production_exports
- background_jobs
- audit_logs

Include:
- Primary keys.
- Foreign keys.
- Created and updated timestamps.
- Appropriate uniqueness constraints.
- Indexes for discovery and trend queries.
- Status fields.
- Source provenance.

Maintain the relationship between each documentary claim and its supporting research sources.

Avoid storing duplicated source content unnecessarily.

Use persistent file/object storage for large media assets rather than placing all files in the relational database.

---

# 14. SECURITY AND RELIABILITY

Implement:

- Secure authentication.
- Secure session handling.
- Server-side API keys.
- HTTPS.
- Input validation.
- Rate limiting.
- Access control.
- Audit logs.
- Safe file handling.
- Secure extension communication.
- Protection against duplicate jobs.
- Backup and recovery procedures.

Never put privileged API keys in:
- Chrome extension source.
- Browser local storage.
- Public frontend bundles.
- Client-side environment variables.

The extension should access the backend through authenticated endpoints.

---

# 15. SETTINGS AND CONFIGURATION

Build a comprehensive settings page.

Include:

### Discovery preferences
- Categories.
- Countries and regions where relevant.
- Discovery period.
- Minimum opportunity score.
- Underdog priority.
- Minimum evidence confidence.
- Saturation tolerance.

### Monitoring preferences
- Connector enable/disable.
- Monitoring frequency.
- Source-specific limits.
- API quota visibility.
- Last successful scan.

### AI preferences
- Provider.
- Model.
- Research depth.
- Script duration.
- Writing style.
- Language.
- AI usage limits.

### Notification preferences
- Email address.
- Alert categories.
- Alert thresholds.
- Daily digest time.
- Weekly report schedule.

### Production preferences
- Image frequency.
- Visual style.
- Aspect ratio.
- Narration provider.
- Subtitle format.
- Export settings.

All settings must actually influence system behaviour.

---

# 16. REQUIRED USER INTERFACE PAGES

Build the following pages and connect them to real backend operations.

| Page | Purpose |
|---|---|
| Login | Secure access |
| Dashboard | Overall intelligence overview |
| Discover | New discoveries |
| Trending Now | Current high-momentum stories |
| Emerging Stories | Early opportunities |
| Underdog Opportunities | Ranked narrative opportunities |
| Opportunity Detail | Full evidence and assessment |
| Watchlist | Followed people and events |
| Research Workspace | Detailed biography and fact-checking |
| Research Library | Source and evidence management |
| My Documentaries | Documentary projects |
| Script Editor | Edit and approve narration |
| Production Studio | Manage documentary production |
| Visual Timeline | Narration and image prompts |
| Character References | Visual consistency assets |
| Audio Studio | Narration and audio files |
| Export Centre | Download production packages |
| Notifications | Alerts and history |
| Reports | Daily and weekly intelligence reports |
| Settings | Configuration |
| System Health | Connectors, workers and job monitoring |

Use a clean, modern, professional interface.

Prefer a light, readable design with clear hierarchy, strong typography, useful data visualisations and responsive layouts.

Do not overcomplicate the interface.

---

# 17. IMPLEMENTATION PHASES

Build in phases, but do not mistake phase separation for permission to omit features.

## PHASE 0 — AUDIT AND ARCHITECTURE

- Inspect repository.
- Identify existing functionality.
- Define architecture.
- Define database schema.
- Define API contracts.
- Define connector contracts.
- Define job workflow.
- Define extension/backend communication.
- Prepare implementation plan.

Deliverable: working architectural foundation and implementation report.

## PHASE 1 — TREND DISCOVERY AND UNDERDOG INTELLIGENCE

This is the first major working milestone.

Build:
- Source connector framework.
- Initial source integrations.
- Discovery engine.
- Person and event resolution.
- YouTube performance tracking.
- Trend observation history.
- Cross-platform signal model.
- Underdog Story Finder.
- Evidence-linked scoring.
- Opportunity ranking.
- Discovery dashboard.
- Trending Now page.
- Emerging Stories page.
- Opportunity detail page.
- Watchlist.
- Email alerts.
- Scheduled cloud monitoring.

Prioritise reliable source integrations rather than pretending all platforms are connected.

At minimum, build a real working discovery flow using the most accessible reliable sources, then add the remaining connectors through the same architecture.

**Phase 1 acceptance:** The system can discover a real person and event, gather evidence, assess the underdog story, measure available attention signals, rank the opportunity and display it in the dashboard.

## PHASE 2 — DEEP RESEARCH

Build:
- Biography research.
- Chronological timeline.
- Source library.
- Claim extraction.
- Fact verification.
- Contradiction detection.
- Research gaps.
- Opportunity reports.

**Acceptance:** A qualified discovery can be transformed into a source-linked research report with an accurate chronological life story and explicit uncertainties.

## PHASE 3 — DOCUMENTARY SCRIPT ENGINE

Build:
- Long-form script generation.
- Narrative structure.
- Source-linked script passages.
- Script editing.
- Script versioning.
- Approval workflow.
- Narration export.

**Acceptance:** The system generates a complete, editable documentary narration from an approved research report.

## PHASE 4 — VISUAL AND AUDIO PRODUCTION

Build:
- Character reference sheets.
- Visual timeline.
- Five-to-ten-second image segmentation.
- Detailed image prompts.
- Image asset management.
- Audio generation/import.
- Speech timing alignment.
- Subtitle generation.
- Production manifest.

**Acceptance:** A documentary script becomes a complete, organised visual and audio production package.

## PHASE 5 — CHROME EXTENSION

Build and test the actual Chrome extension, including:
- Manifest V3.
- Popup.
- Background service worker.
- Authentication.
- Latest discoveries.
- Alerts.
- Dashboard navigation.
- Chrome notifications.
- Settings.
- Installation package.

The extension is a required product component and should be developed alongside the dashboard foundation rather than treated as an optional afterthought.

**Acceptance:** The user can install the extension, authenticate, view real discoveries, open individual reports and access the full dashboard.

## PHASE 6 — CLOUD OPERATIONS AND AUTOMATION

Complete:
- Persistent background workers.
- Scheduled monitoring.
- Job retries.
- Recovery.
- Email delivery.
- System health.
- Usage tracking.
- Error alerts.
- Backup strategy.

**Acceptance:** Monitoring and notifications continue when the browser is closed and the user's computer is switched off.

## PHASE 7 — EXPORT AND VIDEO ASSEMBLY

Build:
- Complete production package export.
- ZIP generation.
- FFmpeg assembly where assets are available.
- Final video validation.
- Download and production history.

**Acceptance:** The user can export a complete production package and, where all necessary assets are available, assemble a finished video.

---

# 18. TESTING AND QUALITY ASSURANCE

Do not consider a phase complete because its pages render.

Test actual workflows.

### Discovery tests
- Discover a person through a source.
- Identify the event.
- Resolve aliases.
- Deduplicate repeated records.
- Capture source metrics.
- Generate a trend classification.
- Assess underdog potential.
- Verify claim references.
- Rank the opportunity.

### Research tests
- Generate a biography.
- Build a chronological timeline.
- Link claims to sources.
- Identify conflicting claims.
- Identify missing information.

### Documentary tests
- Generate narration.
- Check script completeness.
- Validate source references.
- Edit and save a version.
- Regenerate selected content.

### Production tests
- Generate visual segments.
- Verify timestamps.
- Check segment durations.
- Export prompts.
- Generate subtitles.
- Create a production manifest.
- Package files.

### Extension tests
- Install extension.
- Authenticate.
- Retrieve discoveries.
- Open dashboard.
- Open opportunity detail.
- Receive notifications.
- Handle expired sessions.
- Handle backend errors.

### Cloud tests
- Close Chrome.
- Stop the local machine.
- Verify scheduled jobs continue in the cloud.
- Verify alerts are sent.
- Simulate connector failure.
- Verify retry and recovery.
- Confirm duplicate jobs and alerts are prevented.

---

# 19. DEFINITION OF DONE

TrendScout is not complete when a dashboard is built.

It is complete when the following end-to-end workflow works:

1. The cloud discovery engine finds a real person and event.
2. The system collects real source evidence.
3. It detects and records public attention.
4. The AI evaluates the person's underdog story.
5. The system produces transparent scores and confidence.
6. The user receives an alert.
7. The user opens the opportunity from the Chrome extension.
8. The dashboard presents the complete opportunity report.
9. The user initiates deeper research.
10. The system creates a source-linked biography.
11. The user generates a documentary script.
12. The script is converted into a visual timeline.
13. Character references and image prompts are prepared.
14. Audio and subtitles can be generated or imported.
15. The production package is exported.
16. The final production assets can be assembled into a video where supported.

No fabricated research, fake integrations, dead buttons or placeholder metrics may be presented as working functionality.

---

# 20. ENGINEERING RULES FOR THE AGENT

1. Work directly in the existing repository.
2. Audit before modifying.
3. Preserve working code.
4. Build modular, maintainable components.
5. Use database migrations.
6. Use typed API contracts.
7. Keep secrets server-side.
8. Implement real integrations.
9. Document API limitations.
10. Implement error handling.
11. Test every completed phase.
12. Keep the application deployable throughout development.
13. Maintain a clear implementation progress report.
14. Do not silently skip requirements.
15. Do not replace functionality with mock data.
16. Do not build unnecessary abstractions.
17. Do not wait for the Chrome extension to be built last; establish its interface and develop it alongside the dashboard.
18. Do not make the system dependent on the user's computer being online.
19. Do not generate unsupported biographical claims.
20. Do not declare a phase complete without passing its acceptance tests.

Maintain these project documents:

- `ARCHITECTURE.md`
- `IMPLEMENTATION_PLAN.md`
- `DATABASE_SCHEMA.md`
- `API_CONTRACTS.md`
- `SOURCE_CONNECTORS.md`
- `AI_WORKFLOWS.md`
- `CHROME_EXTENSION.md`
- `TESTING.md`
- `DEPLOYMENT.md`
- `PROGRESS.md`
- `KNOWN_LIMITATIONS.md`

Update `PROGRESS.md` after every meaningful implementation milestone.

---

# 21. FIRST EXECUTION INSTRUCTION

Begin immediately with the repository audit and architecture assessment.

Then establish the shared backend contracts, database foundation and dashboard/extension communication.

Prioritise the first working Trend Discovery and Underdog Intelligence workflow.

Build the Chrome extension as a real installable component, not a conceptual screen.

Do not stop after the audit, mockup or architectural plan. Continue implementing the system phase by phase.

At the end of each phase, report:

- What was implemented.
- Files and modules created or modified.
- Integrations connected.
- Database changes.
- Tests executed.
- Test results.
- Deployment status.
- Remaining work.
- Known limitations.

**Final instruction: Build TrendScout Studio as a complete working product. The objective is not to show the user what the application could look like. The objective is to deliver an operational AI documentary discovery and production assistant that the user can install, open, use and rely on.**