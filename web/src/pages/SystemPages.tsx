import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Save } from 'lucide-react';
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
          <h2 className="text-base font-black text-display text-nexa-text-primary">
            3. AI & Scriptwriting Preferences
          </h2>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary mb-1">
              Default Documentary Duration (Minutes)
            </label>
            <input
              type="number"
              value={settings.ai?.defaultScriptMinutes ?? 20}
              onChange={(e) => updateField('ai', 'defaultScriptMinutes', Number(e.target.value))}
              className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary"
            />
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
              Visual Beat Image Interval (Seconds: 5–10s)
            </label>
            <input
              type="number"
              min={5}
              max={10}
              value={settings.production?.imageIntervalSeconds ?? 8}
              onChange={(e) => updateField('production', 'imageIntervalSeconds', Number(e.target.value))}
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
