# TrendScout Studio — Source Connectors

## Overview
TrendScout Studio uses a unified `SourceConnector` contract that supports live public feeds out of the box and upgrades seamlessly when optional API keys are configured in `user_settings` or environment variables.

## Implemented Connectors

1. **Google News & Global Publishers Connector (`google_news`)**
   - **Mechanism**: Queries Google News RSS endpoints (`https://news.google.com/rss/search?q=...`) for breakthrough achievements, underdog victories, world records, debuts, and against-all-odds human stories.
   - **Extracted Data**: Headline, publisher, publication date, URL, extracted person and event candidates.
   - **Quota / Cost**: Free public RSS; rate-limited respectfully with caching and deduplication.

2. **Wikipedia & Wikimedia Biographical & Current Events Connector (`wikipedia`)**
   - **Mechanism**: Queries the MediaWiki Action & REST APIs (`https://en.wikipedia.org/api/rest_v1/page/summary/...` and search/parse endpoints) with proper User-Agent headers.
   - **Extracted Data**: Verified biographical summaries, birth dates, career history, structured life sections (Early life, Career, Struggles, Achievements), high-resolution public-domain/CC portraits, and citation links.

3. **Reddit Breakout Communities Connector (`reddit`)**
   - **Mechanism**: Queries public Reddit JSON feeds (`r/sports`, `r/olympics`, `r/NextFuckingLevel`, `r/BeAmazed`, `r/UpliftingNews`, `r/tennis`, `r/chess`, `r/trackandfield`) or OAuth API if configured.
   - **Extracted Data**: Upvotes, comment velocity, share links, external news articles, and community sentiment around emerging people.

4. **YouTube Velocity & Creator Coverage Connector (`youtube`)**
   - **Mechanism**: Uses YouTube Data API v3 when `YOUTUBE_API_KEY` is provided in Settings, and falls back to public YouTube RSS/Atom channel & search feeds (`https://www.youtube.com/feeds/videos.xml`) and oEmbed/page metadata extraction when running without an API key.
   - **Metrics Computed**: Total views, views per hour (`views / hours_since_publish`), views per day, channel-relative performance multiplier (`video_views / channel_baseline_views`), independent creator coverage count, and engagement ratio.

5. **Google Trends Breakout Signal Connector (`google_trends`)**
   - **Mechanism**: Queries Google Trends Daily Search Trends RSS (`https://trends.google.com/trending/rss?geo=US`) to measure search traffic volume and associated news coverage for rising people.

6. **Social Momentum Connectors (`x_twitter`, `tiktok`, `instagram`)**
   - **Mechanism**: Modular connector adapters that ingest configured API/webhook credentials (X API v2, TikTok Research API, Instagram Graph API) or cross-platform social signal citations discovered via news/Reddit aggregation. Clearly reports connector authentication state and quota usage in `connected_sources`.
