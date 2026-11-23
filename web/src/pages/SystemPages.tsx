import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CheckCircle2, Eye, EyeOff, RefreshCw, Save, ShieldAlert, Zap } from 'lucide-react';
import { apiFetch } from '../api/client';
import { StageBadge } from './DiscoveryPages';

// NOTIFICATIONS PAGE
export const NotificationsPage: React.FC = () => {
  const navigate = useNavigate();
  const [notifs, setNotifs] = useState<any[]>([]);

  const load = () => {
    apiFetch('/api/notifications').then((r) => setNotifs(r.notifications || []));
  };

  useEffect(() => {
    load();
  }, []);

  const markAllRead = async () => {
    await apiFetch('/api/notifications/read-all', { method: 'POST', body: '{}' });
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-display text-nexa-text-primary">
            Notifications & Email Alert Log
          </h1>
          <p className="text-xs text-nexa-text-secondary mt-1">
            Deduplicated alerts delivered to the Chrome Extension and producer email outbox.
          </p>
        </div>
        <button
          onClick={markAllRead}
          className="px-4 py-2 rounded-full bg-nexa-brand text-white text-xs font-extrabold shadow-md shadow-nexa-brand/25 cursor-pointer"
        >
          Mark All Read
        </button>
      </div>

      <div className="space-y-3">
        {notifs.map((n) => (
          <div
            key={n.id}
            onClick={() => navigate(n.actionUrl || '/')}
            className={`p-5 rounded-3xl border cursor-pointer transition flex items-start justify-between gap-4 ${
              n.isRead
                ? 'bg-nexa-bg-surface border-nexa-border'
                : 'bg-nexa-brand/5 border-nexa-brand/30'
            }`}
          >
            <div className="space-y-1.5">
              <div className="flex items-center gap-2">
                <span className="px-2.5 py-0.5 rounded-full bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20 text-[10px] font-extrabold uppercase">
                  {n.type}
                </span>
                <span className="text-sm font-black text-display text-nexa-text-primary">{n.title}</span>
              </div>
              <p className="text-xs text-nexa-text-secondary">{n.message}</p>
              <div className="text-[11px] text-nexa-text-muted">
                Created: {new Date(n.createdAt).toLocaleString()} • Email Outbox Logged:{' '}
                {n.emailedAt ? new Date(n.emailedAt).toLocaleTimeString() : 'Delivered'}
              </div>
            </div>
            <span className="text-xs font-extrabold text-nexa-brand shrink-0">Open →</span>
          </div>
        ))}
      </div>
    </div>
  );
};

// REPORTS PAGE (Daily Digest & Weekly Top 10 Underdog Opportunities)
export const ReportsPage: React.FC = () => {
  const navigate = useNavigate();
  const [report, setReport] = useState<any>(null);

  useEffect(() => {
    apiFetch('/api/reports').then(setReport);
  }, []);

  if (!report) return <div className="p-8 text-sm text-nexa-text-muted">Loading intelligence reports...</div>;

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-black text-display text-nexa-text-primary">
          Daily Digest & Weekly Top 10 Documentary Opportunities
        </h1>
        <p className="text-xs text-nexa-text-secondary mt-1">
          Executive intelligence summaries ranked by Underdog Strength, Momentum, Evidence Confidence, and Low Saturation Risk.
        </p>
      </div>

      <div className="nexa-card rounded-3xl p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-black text-display text-nexa-text-primary">
            {report.dailyDigest?.title}
          </h2>
          <span className="text-xs text-nexa-text-muted">
            Generated {new Date(report.generatedAt).toLocaleString()}
          </span>
        </div>

        <div className="space-y-3">
          {(report.weeklyTop10 || []).map((item: any, idx: number) => (
            <div
              key={item.id}
              onClick={() => navigate(`/opportunities/${item.id}`)}
              className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border hover:border-nexa-brand cursor-pointer flex items-center justify-between gap-4 transition"
            >
              <div className="flex items-center gap-4">
                <div className="w-9 h-9 rounded-2xl bg-nexa-brand text-white font-black flex items-center justify-center text-xs shrink-0 shadow-sm shadow-nexa-brand/25">
                  #{idx + 1}
                </div>
                <div>
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-black text-display text-nexa-text-primary">
                      {item.personName}
                    </span>
                    <StageBadge stage={item.trendStage} />
                    <span className="text-xs font-semibold text-nexa-text-muted">
                      {item.underdogClassification}
                    </span>
                  </div>
                  <p className="text-xs text-nexa-text-secondary mt-0.5">{item.headline}</p>
                </div>
              </div>
              <div className="text-right shrink-0">
                <div className="text-lg font-black text-display text-nexa-brand">
                  {Number(item.overallScore).toFixed(1)}
                </div>
                <div className="text-[10px] text-nexa-text-muted font-bold">
                  Sat. Risk: {item.saturationRisk}%
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// SETTINGS PAGE
export const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<any>(null);
  const [saved, setSaved] = useState(false);
  const [showGeminiKey, setShowGeminiKey] = useState(false);
  const [showYoutubeKey, setShowYoutubeKey] = useState(false);
  const [testingAI, setTestingAI] = useState(false);
  const [aiTestResult, setAiTestResult] = useState<any>(null);
  const [liveGroqModels, setLiveGroqModels] = useState<string[]>([]);

  useEffect(() => {
    apiFetch('/api/settings').then(setSettings);
  }, []);

  if (!settings) return <div className="p-8 text-sm text-nexa-text-muted">Loading system settings...</div>;

  const updateField = (section: string, key: string, value: any) => {
    setSettings((prev: any) => ({
      ...prev,
      [section]: {
        ...(prev[section] || {}),
        [key]: value,
      },
    }));
  };

  const handleSave = async () => {
    await apiFetch('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    });
    setSaved(true);
    setTimeout(() => setSaved(false), 2500);
  };

  const handleTestAI = async () => {
    setTestingAI(true);
    setAiTestResult(null);
    try {
      const res = await apiFetch('/api/settings/test-ai', {
        method: 'POST',
        body: JSON.stringify({
          provider: settings.ai?.provider || 'Groq',
          geminiApiKey: settings.ai?.geminiApiKey || '',
          model: settings.ai?.model || 'llama-3.3-70b-versatile',
        }),
      });
      setAiTestResult(res);
      if (Array.isArray(res?.availableModels) && res.availableModels.length > 0) {
        setLiveGroqModels(res.availableModels);
      }
      if (res?.connected && res?.modelUsed) {
        updateField('ai', 'model', res.modelUsed);
      }
    } catch (err: any) {
      setAiTestResult({ connected: false, error: err?.message || 'Test request failed' });
    } finally {
      setTestingAI(false);
    }
  };

  const handleApiKeyChange = (val: string) => {
    const trimmed = val.trim();
    updateField('ai', 'geminiApiKey', val);
    if (trimmed.startsWith('gsk_')) {
      updateField('ai', 'provider', 'Groq');
      updateField('ai', 'model', 'qwen/qwen3.8-27b');
    } else if (trimmed.startsWith('sk-or-')) {
      updateField('ai', 'provider', 'OpenRouter');
      updateField('ai', 'model', 'meta-llama/llama-3.3-70b-instruct:free');
    } else if (trimmed.startsWith('AIza') || trimmed.startsWith('AQ.')) {
      updateField('ai', 'provider', 'Google Gemini');
      updateField('ai', 'model', 'gemini-2.5-flash');
    } else if (trimmed.startsWith('sk-')) {
      updateField('ai', 'provider', 'OpenAI');
      updateField('ai', 'model', 'gpt-4o-mini');
    }
  };

  const activeProvider = settings.ai?.provider || 'Groq';
  const defaultGroqModels = [
    { value: 'qwen/qwen3.8-27b', label: 'qwen/qwen3.8-27b (Recommended Free Tier)' },
    { value: 'openai/gpt-oss-120b', label: 'openai/gpt-oss-120b (High-Capacity Reasoning)' },
    { value: 'openai/gpt-oss-20b', label: 'openai/gpt-oss-20b (Fast Reasoning)' },
  ];
  const groqModelList =
    liveGroqModels.length > 0
      ? liveGroqModels.map((m) => ({ value: m, label: `${m} (Live on your Groq key)` }))
      : defaultGroqModels;
  const modelOptions: Record<string, { value: string; label: string }[]> = {
    Groq: groqModelList,
    OpenRouter: [
      { value: 'meta-llama/llama-3.3-70b-instruct:free', label: 'llama-3.3-70b-instruct:free' },
      { value: 'google/gemma-3-27b-it:free', label: 'gemma-3-27b-it:free' },
    ],
    'Google Gemini': [
      { value: 'gemini-2.5-flash', label: 'gemini-2.5-flash (Recommended)' },
      { value: 'gemini-2.5-pro', label: 'gemini-2.5-pro (Deepest Reasoning)' },
      { value: 'gemini-2.0-flash', label: 'gemini-2.0-flash' },
    ],
    OpenAI: [
      { value: 'gpt-4o-mini', label: 'gpt-4o-mini (Fast & Low Cost)' },
      { value: 'gpt-4o', label: 'gpt-4o (Flagship)' },
    ],
  };
  const currentModels = modelOptions[activeProvider] || modelOptions.Groq;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-display text-nexa-text-primary">
            Studio Settings & Configuration
          </h1>
          <p className="text-xs text-nexa-text-secondary mt-1">
            Configure Discovery thresholds, Monitoring intervals, AI models, Email/Chrome notifications, and Production defaults.
          </p>
        </div>
        <button
          onClick={handleSave}
          className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white font-extrabold text-xs flex items-center gap-2 shadow-lg shadow-nexa-brand/25 cursor-pointer"
        >
          <Save className="w-4 h-4" />
          {saved ? 'Settings Saved!' : 'Save All Settings'}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="nexa-card rounded-3xl p-6 space-y-4">
          <h2 className="text-base font-black text-display text-nexa-text-primary">
            1. Discovery Preferences
          </h2>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Minimum Opportunity Score
            </label>
            <input
              type="number"
              value={settings.discovery?.minOpportunityScore ?? 75}
              onChange={(e) => updateField('discovery', 'minOpportunityScore', Number(e.target.value))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Minimum Evidence Confidence (%)
            </label>
            <input
              type="number"
              value={settings.discovery?.minEvidenceConfidence ?? 80}
              onChange={(e) => updateField('discovery', 'minEvidenceConfidence', Number(e.target.value))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Maximum Saturation Tolerance (%)
            </label>
            <input
              type="number"
              value={settings.discovery?.saturationTolerance ?? 35}
              onChange={(e) => updateField('discovery', 'saturationTolerance', Number(e.target.value))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
        </div>

        <div className="nexa-card rounded-3xl p-6 space-y-4">
          <h2 className="text-base font-black text-display text-nexa-text-primary">
            2. Cloud Monitoring Preferences
          </h2>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Scan Frequency (Minutes)
            </label>
            <input
              type="number"
              value={settings.monitoring?.frequencyMinutes ?? 15}
              onChange={(e) => updateField('monitoring', 'frequencyMinutes', Number(e.target.value))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Max Candidate Items Per Connector Run
            </label>
            <input
              type="number"
              value={settings.monitoring?.maxItemsPerScan ?? 25}
              onChange={(e) => updateField('monitoring', 'maxItemsPerScan', Number(e.target.value))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
        </div>

        <div className="nexa-card rounded-3xl p-6 space-y-4">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-base font-black text-display text-nexa-text-primary">
              3. AI Engine &amp; Scriptwriting Credentials
            </h2>
            <button
              type="button"
              onClick={handleTestAI}
              disabled={testingAI}
              className="px-3.5 py-1.5 rounded-full border border-nexa-brand/40 bg-nexa-brand/10 hover:bg-nexa-brand/20 text-nexa-brand text-[11px] font-extrabold flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            >
              {testingAI ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Zap className="w-3.5 h-3.5" />}
              {testingAI ? 'Testing AI...' : 'Test AI Connection'}
            </button>
          </div>

          {aiTestResult && (
            <div
              className={`p-3.5 rounded-2xl border text-xs flex items-start gap-2.5 ${
                aiTestResult.connected
                  ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-600 dark:text-emerald-400'
                  : 'bg-rose-500/10 border-rose-500/30 text-rose-600 dark:text-rose-400'
              }`}
            >
              {aiTestResult.connected ? (
                <CheckCircle2 className="w-4 h-4 shrink-0 mt-0.5" />
              ) : (
                <ShieldAlert className="w-4 h-4 shrink-0 mt-0.5" />
              )}
              <div className="space-y-0.5">
                {aiTestResult.connected ? (
                  <>
                    <div className="font-extrabold">
                      Connected Live: {aiTestResult.modelUsed} via {aiTestResult.endpointType} ({aiTestResult.latencyMs}ms)
                    </div>
                    <div className="text-[11px] opacity-90">
                      Real Live AI is active for Discovery, Stage 0 Competitor Study, Underdog Scriptwriting, and SEO Packaging.
                    </div>
                  </>
                ) : (
                  <>
                    <div className="font-extrabold">
                      AI Connection Failed ({aiTestResult.endpointType || 'API Error'})
                    </div>
                    <div className="text-[11px] break-words">{aiTestResult.error}</div>
                  </>
                )}
              </div>
            </div>
          )}

          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              AI Provider
            </label>
            <select
              value={activeProvider}
              onChange={(e) => {
                const prov = e.target.value;
                updateField('ai', 'provider', prov);
                const opts = modelOptions[prov] || modelOptions.Groq;
                updateField('ai', 'model', opts[0].value);
              }}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            >
              <option value="Groq">Groq (Free Tier — No Billing Required)</option>
              <option value="OpenRouter">OpenRouter (Free Models — No Billing)</option>
              <option value="Google Gemini">Google Gemini (AI Studio / Vertex AI)</option>
              <option value="OpenAI">OpenAI (GPT-4o)</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              AI API Key (<code className="text-[11px]">gsk_...</code> Groq, <code className="text-[11px]">sk-or-...</code> OpenRouter, <code className="text-[11px]">AIza...</code> Gemini, or <code className="text-[11px]">sk-...</code> OpenAI)
            </label>
            <div className="relative">
              <input
                type={showGeminiKey ? 'text' : 'password'}
                value={settings.ai?.geminiApiKey ?? ''}
                onChange={(e) => handleApiKeyChange(e.target.value)}
                placeholder="Paste gsk_... (Groq), sk-or-... (OpenRouter), or AIza... (Gemini)"
                className="w-full px-3.5 py-2.5 pr-10 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary font-mono"
              />
              <button
                type="button"
                onClick={() => setShowGeminiKey((v) => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-nexa-text-muted hover:text-nexa-text-primary cursor-pointer"
                title={showGeminiKey ? 'Hide key' : 'Show key'}
              >
                {showGeminiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
                AI Model
              </label>
              <select
                value={currentModels.some((m) => m.value === settings.ai?.model) ? settings.ai?.model : currentModels[0].value}
                onChange={(e) => updateField('ai', 'model', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
              >
                {currentModels.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
                Default Documentary Duration (Min)
              </label>
              <input
                type="number"
                min={15}
                max={25}
                value={settings.ai?.defaultScriptMinutes ?? 20}
                onChange={(e) => updateField('ai', 'defaultScriptMinutes', Number(e.target.value))}
                className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              YouTube Data API v3 Key (Optional Override)
            </label>
            <div className="relative">
              <input
                type={showYoutubeKey ? 'text' : 'password'}
                value={settings.ai?.youtubeApiKey ?? ''}
                onChange={(e) => updateField('ai', 'youtubeApiKey', e.target.value)}
                placeholder="Uses server .env YOUTUBE_API_KEY if left blank"
                className="w-full px-3.5 py-2.5 pr-10 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary font-mono"
              />
              <button
                type="button"
                onClick={() => setShowYoutubeKey((v) => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-nexa-text-muted hover:text-nexa-text-primary cursor-pointer"
                title={showYoutubeKey ? 'Hide key' : 'Show key'}
              >
                {showYoutubeKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <div className="p-3 rounded-2xl bg-nexa-brand/5 border border-nexa-brand/20 text-xs text-nexa-text-secondary">
            <strong className="text-nexa-brand">Live AI Required (Zero Fallbacks):</strong> All discovery synthesis, Stage 0 competitor analysis, documentary scriptwriting, and SEO packaging require a working AI key (Groq, OpenRouter, Gemini, or OpenAI) and return an explicit error if AI fails. Click <strong>Test AI Connection</strong> above to verify your key.
          </div>

          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Narrative Style
            </label>
            <input
              type="text"
              value={settings.ai?.writingStyle ?? ''}
              onChange={(e) => updateField('ai', 'writingStyle', e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
        </div>

        <div className="nexa-card rounded-3xl p-6 space-y-4">
          <h2 className="text-base font-black text-display text-nexa-text-primary">
            4. Notification & Production Preferences
          </h2>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Producer Alert Email
            </label>
            <input
              type="email"
              value={settings.notification?.emailAddress ?? 'producer@trendscout.studio'}
              onChange={(e) => updateField('notification', 'emailAddress', e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Visual Beat Image Interval (Seconds: ≤7s Pace)
            </label>
            <input
              type="number"
              min={3}
              max={7}
              value={Math.min(7, settings.production?.imageIntervalSeconds ?? 7)}
              onChange={(e) => updateField('production', 'imageIntervalSeconds', Math.min(7, Math.max(3, Number(e.target.value))))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

// SYSTEM HEALTH & AUTONOMOUS WORKER MONITOR PAGE
export const SystemHealthPage: React.FC = () => {
  const [health, setHealth] = useState<any>(null);

  const load = () => {
    apiFetch('/api/system/health').then(setHealth);
  };

  useEffect(() => {
    load();
  }, []);

  if (!health) return <div className="p-8 text-sm text-nexa-text-muted">Loading System Health & Queue Telemetry...</div>;

  const retryJob = async (jobId: string) => {
    await apiFetch(`/api/system/jobs/${jobId}/retry`, { method: 'POST', body: '{}' });
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-display text-nexa-text-primary">
            System Health, Connectors & Persistent Job Queue
          </h1>
          <p className="text-xs text-nexa-text-secondary mt-1">
            Status: {health.runtime} • {health.database}
          </p>
        </div>
        <button
          onClick={load}
          className="px-4 py-2 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold bg-nexa-bg-surface cursor-pointer"
        >
          Refresh Telemetry
        </button>
      </div>

      <div className="nexa-card rounded-3xl p-6">
        <h2 className="text-base font-black text-display text-nexa-text-primary mb-4">
          Connected Platform Sources
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {(health.connectors || []).map((c: any) => (
            <div key={c.id} className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-xs font-extrabold text-nexa-text-primary">{c.name}</span>
                <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[10px] font-extrabold uppercase">
                  {c.status}
                </span>
              </div>
              <div className="text-[11px] text-nexa-text-muted">Platform: {c.platform}</div>
              <div className="text-[11px] text-nexa-text-muted">
                Quota: {c.quotaUsed} / {c.quotaLimit}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="nexa-card rounded-3xl p-6">
        <h2 className="text-base font-black text-display text-nexa-text-primary mb-4">
          Automated Background Tasks &amp; Discovery Queue
        </h2>
        <div className="space-y-2.5">
          {(health.jobs || []).map((j: any) => (
            <div
              key={j.id}
              className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border flex items-center justify-between text-xs"
            >
              <div>
                <span className="font-black text-display text-nexa-text-primary mr-2">{j.jobType}</span>
                <span className="px-2.5 py-0.5 rounded-full bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20 font-extrabold uppercase text-[10px]">
                  {j.status}
                </span>
                <span className="text-nexa-text-muted ml-3">
                  Attempts: {j.attempts}/{j.maxAttempts} • Created: {new Date(j.createdAt).toLocaleString()}
                </span>
              </div>
              {j.status !== 'completed' && (
                <button
                  onClick={() => retryJob(j.id)}
                  className="px-3.5 py-1.5 rounded-full bg-nexa-brand text-white font-extrabold cursor-pointer"
                >
                  Retry Job
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
