# TrendScout Studio — Known Limitations & Platform Notes

1. **Social Platform API Tiers (X/Twitter, TikTok, Instagram)**:
   - Official X API v2, TikTok Research API, and Instagram Graph API require paid or approved developer credentials. When those credentials are not entered in Settings, TrendScout Studio relies on Google News RSS, Reddit public JSON, Wikipedia/Wikimedia APIs, YouTube public Atom/oEmbed feeds, and Google Trends RSS, while clearly reporting the connector authentication status in System Health and Settings.
2. **Google Flow Integration**:
   - Because Google Flow does not expose a public unauthenticated bulk-generation REST API, TrendScout Studio implements a structured **Google Flow Prompt & Reference Pack Exporter** (`google_flow_pack` / `image-prompts.txt` / `visual-timeline.csv` / one-click copy per 5–10s segment) behind a modular adapter, plus built-in local SVG/PNG storyboard frame generation so FFmpeg video assembly works out of the box.
3. **Chrome Web Store Publication**:
   - The Chrome Extension is built, validated, and packaged as both an unpacked Manifest V3 folder (`/extension`) and a distributable `.zip` (`/storage/exports/trendscout-chrome-extension.zip`) ready for `chrome://extensions` Developer Mode loading. It has not been submitted to the public Chrome Web Store.
