import React, { useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  CheckCircle2, Copy, Download, ExternalLink, Film,
  Hash, Mic, RefreshCw, Save, Sparkles, Tag, Trash2, Upload
} from 'lucide-react';
import { apiFetch } from '../api/client';
import { StageBadge } from './DiscoveryPages';

// WATCHLIST PAGE
export const WatchlistPage: React.FC = () => {
  const navigate = useNavigate();
  const [items, setItems] = useState<any[]>([]);

  const load = () => {
    apiFetch('/api/watchlist').then((r) => setItems(r.watchlist || []));
  };

  useEffect(() => {
    load();
  }, []);

  const handleRemove = async (id: string) => {
    await apiFetch(`/api/watchlist/${id}`, { method: 'DELETE' });
    load();
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-black text-display text-nexa-text-primary">
          Watched People & Momentum Alerts
        </h1>
        <p className="text-xs text-nexa-text-secondary mt-1">
          Automatic stage-transition alerts when watched underdogs move from Emerging to Accelerating or Major Viral.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {items.map((w) => (
          <div
            key={w.id}
            className="nexa-card rounded-3xl p-6 flex flex-col md:flex-row md:items-center justify-between gap-4"
          >
            <div className="space-y-1.5">
              <div className="flex items-center gap-2.5 flex-wrap">
                <span className="text-base font-black text-display text-nexa-text-primary">
                  {w.personName}
                </span>
                <StageBadge stage={w.trendStage} />
                <span className="text-xs font-extrabold text-nexa-brand bg-nexa-brand/10 border border-nexa-brand/20 px-2.5 py-0.5 rounded-full">
                  Score: {Number(w.opportunityScore).toFixed(1)}
                </span>
              </div>
              <div className="text-xs font-bold text-nexa-text-primary">{w.headline}</div>
              <div className="text-xs text-nexa-text-muted">Producer Notes: {w.notes}</div>
            </div>
            <div className="flex items-center gap-2.5 shrink-0">
              <button
                onClick={() => navigate(`/opportunities/${w.discoveryId}`)}
                className="px-4 py-2 rounded-full bg-nexa-brand text-white text-xs font-extrabold shadow-md shadow-nexa-brand/20 cursor-pointer"
              >
                Open Report →
              </button>
              <button
                onClick={() => handleRemove(w.id)}
                className="p-2.5 rounded-full border border-nexa-border text-rose-500 hover:bg-rose-500/10 cursor-pointer"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// RESEARCH LIBRARY & FACT VERIFICATION WORKSPACE
export const ResearchLibraryPage: React.FC = () => {
  const [data, setData] = useState<{ sources: any[]; claims: any[] }>({ sources: [], claims: [] });
  const [filterStatus, setFilterStatus] = useState('');

  const load = () => {
    apiFetch('/api/research/library').then(setData);
  };

  useEffect(() => {
    load();
  }, []);

  const handleStatusChange = async (claimId: string, newStatus: string) => {
    await apiFetch(`/api/research/claims/${claimId}/verify`, {
      method: 'POST',
      body: JSON.stringify({ verificationStatus: newStatus }),
    });
    load();
  };

  const claims = data.claims.filter((c) => !filterStatus || c.verificationStatus === filterStatus);

  return (
    <div className="space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black text-display text-nexa-text-primary">
            Research Library & Claim Verification Matrix
          </h1>
          <p className="text-xs text-nexa-text-secondary mt-1">
            Every biographical claim is linked to its primary publisher source and labeled (Verified, Reported, Attributed, Interpretation, Unverified, Unknown).
          </p>
        </div>
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          className="px-4 py-2.5 rounded-full border border-nexa-border bg-nexa-bg-surface text-nexa-text-primary text-xs font-bold"
        >
          <option value="">All Verification Labels</option>
          <option value="Verified">Verified</option>
          <option value="Reported">Reported</option>
          <option value="Attributed">Attributed</option>
          <option value="Interpretation">Interpretation</option>
          <option value="Unverified">Unverified</option>
          <option value="Unknown">Unknown</option>
        </select>
      </div>

      <div className="nexa-card rounded-3xl p-6">
        <h2 className="text-base font-black text-display text-nexa-text-primary mb-4">
          Biographical Claims ({claims.length})
        </h2>
        <div className="space-y-3">
          {claims.map((c) => (
            <div
              key={c.id}
              className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border flex flex-col lg:flex-row lg:items-center justify-between gap-4"
            >
              <div className="space-y-1.5 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-xs font-extrabold text-nexa-text-primary">{c.personName}</span>
                  <span className="px-2.5 py-0.5 rounded-full bg-nexa-brand/10 text-nexa-brand text-[10px] font-extrabold">
                    {c.category}
                  </span>
                  <span className="text-[11px] text-nexa-text-muted">{c.eventDate}</span>
                </div>
                <p className="text-xs text-nexa-text-secondary leading-relaxed">{c.claimText}</p>
                {c.sourceUrl && (
                  <a
                    href={c.sourceUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-[11px] text-nexa-brand font-extrabold hover:underline"
                  >
                    Source Citation <ExternalLink className="w-3 h-3" />
                  </a>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <select
                  value={c.verificationStatus}
                  onChange={(e) => handleStatusChange(c.id, e.target.value)}
                  className="px-3.5 py-2 rounded-full border border-nexa-border text-xs font-extrabold bg-nexa-bg-surface text-nexa-text-primary"
                >
                  {['Verified', 'Reported', 'Attributed', 'Interpretation', 'Unverified', 'Unknown'].map((st) => (
                    <option key={st} value={st}>{st}</option>
                  ))}
                </select>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="nexa-card rounded-3xl p-6">
        <h2 className="text-base font-black text-display text-nexa-text-primary mb-4">
          Indexed Archival & News Sources ({data.sources.length})
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {data.sources.map((s) => (
            <div key={s.id} className="p-5 rounded-2xl bg-nexa-bg-base border border-nexa-border space-y-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs font-extrabold text-nexa-text-primary">{s.title}</span>
                <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[10px] font-extrabold shrink-0">
                  Reliability: {Math.round(Number(s.reliabilityScore || 0.95) * 100)}%
                </span>
              </div>
              <div className="text-[11px] text-nexa-text-muted font-semibold">
                Subject: {s.personName} • Publisher: {s.publisher}
              </div>
              <p className="text-xs text-nexa-text-secondary line-clamp-2">{s.rawExcerpt}</p>
              <a
                href={s.url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 text-xs font-extrabold text-nexa-brand hover:underline"
              >
                Open Source URL <ExternalLink className="w-3 h-3" />
              </a>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// MY DOCUMENTARIES PAGE
export const MyDocumentariesPage: React.FC = () => {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<any[]>([]);

  useEffect(() => {
    apiFetch('/api/projects').then((r) => setProjects(r.projects || []));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-black text-display text-nexa-text-primary">My Documentary Projects</h1>
        <p className="text-xs text-nexa-text-secondary mt-1">
          Manage long-form 20-minute realigned documentary scripts, ≤7s Google Flow visual timelines, character reference locks, narration audio, and ZIP/MP4 exports.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {projects.map((p) => (
          <div
            key={p.id}
            onClick={() => navigate(`/production/${p.id}`)}
            className="nexa-card nexa-card-interactive rounded-3xl p-6 cursor-pointer space-y-4"
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <span className="px-2.5 py-0.5 rounded-full text-[10px] font-extrabold bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 uppercase">
                  {p.status}
                </span>
                <h2 className="text-lg font-black text-display text-nexa-text-primary mt-2">{p.title}</h2>
                <p className="text-xs text-nexa-text-muted mt-0.5 font-semibold">
                  Subject: {p.personName} • Target: {p.targetDurationMinutes} mins ({p.aspectRatio})
                </p>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-3 pt-2">
              <div className="bg-nexa-bg-base p-3.5 rounded-2xl border border-nexa-border">
                <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Visual Beats</div>
                <div className="text-base font-black text-display text-nexa-text-primary">{p.segmentCount} segments</div>
              </div>
              <div className="bg-nexa-bg-base p-3.5 rounded-2xl border border-nexa-border">
                <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Character Refs</div>
                <div className="text-base font-black text-display text-nexa-text-primary">{p.characterCount} sheets</div>
              </div>
              <div className="bg-nexa-bg-base p-3.5 rounded-2xl border border-nexa-border">
                <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Exports</div>
                <div className="text-base font-black text-display text-nexa-text-primary">{p.exportCount} packages</div>
              </div>
            </div>

            <div className="flex items-center justify-between pt-3 border-t border-nexa-border text-xs font-extrabold text-nexa-brand">
              <span>Style: {p.visualStyle}</span>
              <span>Open Production Studio →</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// COMPLETE 9-TAB PRODUCTION STUDIO WORKSPACE (ofia_ng Sub-Tab Pill Style)
const STUDIO_TABS = [
  { id: 'overview', label: '1. Overview' },
  { id: 'script', label: '2. Script' },
  { id: 'research', label: '3. Research' },
  { id: 'timeline', label: '4. Visual Timeline (Google Flow)' },
  { id: 'characters', label: '5. Character References' },
  { id: 'images', label: '6. Images' },
  { id: 'audio', label: '7. Audio' },
  { id: 'subtitles', label: '8. Subtitles' },
  { id: 'export', label: '9. Export & Video Assembly' },
];

export const ProductionStudioPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const activeTab = searchParams.get('tab') || 'overview';

  const [ws, setWs] = useState<any>(null);
  const [scriptText, setScriptText] = useState('');
  const [durationMinutes, setDurationMinutes] = useState(20);
  const [biographicalField, setBiographicalField] = useState('auto');
  const [narrativeStyle, setNarrativeStyle] = useState('Investigative Archival Documentary');
  const [scriptViewMode, setScriptViewMode] = useState<'realigned' | 'raw'>('realigned');
  const [ytDescription, setYtDescription] = useState('');
  const [ytTagsInput, setYtTagsInput] = useState('');
  const [ytHashtagsInput, setYtHashtagsInput] = useState('');
  const [seoBusy, setSeoBusy] = useState(false);
  const [briefBusy, setBriefBusy] = useState(false);
  const [seoSaved, setSeoSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  const [scriptError, setScriptError] = useState('');
  const [briefError, setBriefError] = useState('');
  const [seoError, setSeoError] = useState('');
  const [expandedTranscriptIdx, setExpandedTranscriptIdx] = useState<number | null>(null);
  const [copiedId, setCopiedId] = useState('');
  const [exportResult, setExportResult] = useState<any>(null);

  const loadWorkspace = (targetId?: string) => {
    const resolvedId = targetId || id || ws?.project?.id;
    if (!resolvedId) return;
    apiFetch(`/api/projects/${resolvedId}`).then((data) => {
      setWs(data);
      setScriptText(data.script?.fullNarrationText || '');
      setDurationMinutes(data.project?.targetDurationMinutes || 20);
      setYtDescription(data.project?.youtubeDescription || '');
      setYtTagsInput(Array.isArray(data.project?.youtubeTags) ? data.project.youtubeTags.join(', ') : '');
      setYtHashtagsInput(Array.isArray(data.project?.youtubeHashtags) ? data.project.youtubeHashtags.join(' ') : '');
    }).catch(() => {
      navigate('/documentaries', { replace: true });
    });
  };

  useEffect(() => {
    if (!id) {
      apiFetch('/api/projects').then((res) => {
        const list = res.projects || [];
        if (list.length > 0) {
          navigate(`/production/${list[0].id}`, { replace: true });
        } else {
          navigate('/documentaries', { replace: true });
        }
      }).catch(() => navigate('/documentaries'));
      return;
    }
    loadWorkspace(id);
  }, [id]);

  if (!ws) return <div className="p-8 text-sm text-nexa-text-muted">Loading Production Studio Workspace...</div>;

  const proj = ws.project || {};
  const scr = ws.script || {};
  const chars = ws.characterReferences || [];
  const segs = ws.visualSegments || [];
  const audios = ws.audioAssets || [];
  const subs = ws.subtitleAssets || [];
  const exportsList = ws.exports || [];
  const research = ws.research || {};

  const setTab = (tabId: string) => setSearchParams({ tab: tabId });

  const copyPrompt = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(key);
    setTimeout(() => setCopiedId(''), 1800);
  };

  const handleSaveScript = async () => {
    setBusy(true);
    try {
      await apiFetch(`/api/projects/${proj.id}/script`, {
        method: 'PUT',
        body: JSON.stringify({ fullNarrationText: scriptText, changeSummary: 'Producer Studio Edit' }),
      });
      loadWorkspace();
    } finally {
      setBusy(false);
    }
  };

  const handleRegenerateScript = async () => {
    setBusy(true);
    setScriptError('');
    try {
      await apiFetch(`/api/projects/${proj.id}/script/generate`, {
        method: 'POST',
        body: JSON.stringify({
          targetDurationMinutes: durationMinutes,
          style: narrativeStyle,
          biographicalField,
        }),
      });
      loadWorkspace(proj.id);
    } catch (err: any) {
      setScriptError(err?.message || 'Failed to regenerate documentary script');
    } finally {
      setBusy(false);
    }
  };

  const handleRegenerateParagraph = async (idx: number) => {
    setBusy(true);
    setScriptError('');
    try {
      await apiFetch(`/api/projects/${proj.id}/script/regenerate-paragraph`, {
        method: 'POST',
        body: JSON.stringify({ paragraphIndex: idx }),
      });
      loadWorkspace();
    } catch (err: any) {
      setScriptError(err?.message || 'AI shot refinement failed');
    } finally {
      setBusy(false);
    }
  };

  const handleApproveScript = async () => {
    setBusy(true);
    try {
      await apiFetch(`/api/projects/${proj.id}/script/approve`, { method: 'POST', body: '{}' });
      loadWorkspace();
    } finally {
      setBusy(false);
    }
  };

  const handleRebuildCharacters = async () => {
    setBusy(true);
    try {
      await apiFetch(`/api/projects/${proj.id}/characters/generate`, { method: 'POST', body: '{}' });
      loadWorkspace();
    } finally {
      setBusy(false);
    }
  };

  const handleRebuildTimeline = async () => {
    setBusy(true);
    try {
      await apiFetch(`/api/projects/${proj.id}/timeline/generate`, { method: 'POST', body: '{}' });
      loadWorkspace();
    } finally {
      setBusy(false);
    }
  };

  const handleGenerateAudio = async () => {
    setBusy(true);
    try {
      await apiFetch(`/api/projects/${proj.id}/audio/generate`, { method: 'POST', body: '{}' });
      loadWorkspace();
    } finally {
      setBusy(false);
    }
  };

  const handleAudioUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setBusy(true);
    try {
      const fd = new FormData();
      fd.append('audio', file);
      await apiFetch(`/api/projects/${proj.id}/audio/upload`, {
        method: 'POST',
        body: fd,
      });
      loadWorkspace();
    } finally {
      setBusy(false);
    }
  };

  const handleExportPackage = async (assembleVideo: boolean) => {
    setBusy(true);
    try {
      const endpoint = assembleVideo ? `/api/projects/${proj.id}/assemble-video` : `/api/projects/${proj.id}/export`;
      const res = await apiFetch(endpoint, { method: 'POST', body: '{}' });
      setExportResult(res);
      loadWorkspace(proj.id);
    } finally {
      setBusy(false);
    }
  };

  const parsedYtTags = ytTagsInput
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);
  const parsedYtHashtags = ytHashtagsInput
    .split(/[\s,]+/)
    .map((h) => h.trim())
    .filter(Boolean)
    .map((h) => (h.startsWith('#') ? h : `#${h}`));
  const ytTagsCharCount = parsedYtTags.join(', ').length;

  const handleGenerateProjectSEO = async () => {
    setSeoBusy(true);
    setSeoError('');
    try {
      await apiFetch(`/api/projects/${proj.id}/seo/generate`, { method: 'POST', body: '{}' });
      loadWorkspace(proj.id);
    } catch (err: any) {
      setSeoError(err?.message || 'AI YouTube SEO packaging failed');
    } finally {
      setSeoBusy(false);
    }
  };

  const handleGenerateCompetitiveBrief = async () => {
    setBriefBusy(true);
    setBriefError('');
    try {
      await apiFetch(`/api/projects/${proj.id}/competitive-brief/generate`, { method: 'POST', body: '{}' });
      loadWorkspace(proj.id);
    } catch (err: any) {
      setBriefError(err?.message || 'AI Stage 0 Competitor Study failed');
    } finally {
      setBriefBusy(false);
    }
  };

  const handleSaveProjectSEO = async () => {
    setSeoBusy(true);
    try {
      await apiFetch(`/api/projects/${proj.id}/seo`, {
        method: 'PUT',
        body: JSON.stringify({
          youtubeDescription: ytDescription,
          youtubeTags: parsedYtTags,
          youtubeHashtags: parsedYtHashtags,
        }),
      });
      setSeoSaved(true);
      setTimeout(() => setSeoSaved(false), 2000);
      loadWorkspace(proj.id);
    } finally {
      setSeoBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="nexa-card rounded-3xl p-6">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5 flex-wrap">
              <span className="px-3 py-0.5 rounded-full bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20 text-xs font-extrabold uppercase">
                {proj.status}
              </span>
              <span className="text-xs text-nexa-text-muted font-semibold">
                Target: {proj.targetDurationMinutes} mins • {proj.aspectRatio} • Visual Interval: {proj.imageIntervalSeconds}s
              </span>
            </div>
            <h1 className="text-xl lg:text-2xl font-black text-display text-nexa-text-primary mt-2">
              {proj.title}
            </h1>
            <p className="text-xs text-nexa-text-secondary mt-1">Visual Style: {proj.visualStyle}</p>
          </div>
          <div className="flex flex-wrap gap-2.5">
            <button
              onClick={() => handleExportPackage(false)}
              disabled={busy}
              className="px-4 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white font-extrabold text-xs flex items-center gap-2 shadow-md shadow-nexa-brand/25 cursor-pointer"
            >
              <Download className="w-4 h-4" />
              {busy ? 'Processing...' : 'Export Production ZIP'}
            </button>
            <button
              onClick={() => handleExportPackage(true)}
              disabled={busy}
              className="px-4 py-2.5 rounded-full bg-emerald-600 hover:bg-emerald-500 text-white font-extrabold text-xs flex items-center gap-2 shadow-md shadow-emerald-600/20 cursor-pointer"
            >
              <Film className="w-4 h-4" />
              Assemble 16:9 MP4 Video
            </button>
          </div>
        </div>

        {/* OFIA_NG HORIZONTAL SUB-NAVIGATION PILL TABS */}
        <div className="flex items-center gap-2 overflow-x-auto pb-1 scrollbar-hide border-t border-nexa-border mt-5 pt-4">
          {STUDIO_TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`px-4 py-2 rounded-full text-xs font-semibold flex items-center gap-2 transition-all cursor-pointer whitespace-nowrap ${
                activeTab === t.id
                  ? 'bg-nexa-brand text-white shadow-md shadow-nexa-brand/25 font-bold border border-nexa-brand'
                  : 'bg-nexa-bg-base hover:bg-nexa-bg-surface text-nexa-text-secondary hover:text-nexa-text-primary border border-nexa-border'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* TAB 1: OVERVIEW */}
      {activeTab === 'overview' && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 nexa-card rounded-3xl p-6 space-y-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-base font-black text-display text-nexa-text-primary">
                YouTube Packaging, SEO &amp; Creative Direction
              </h2>
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  onClick={handleGenerateProjectSEO}
                  disabled={seoBusy}
                  className="px-3.5 py-1.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold flex items-center gap-1.5 shadow-sm shadow-nexa-brand/20 cursor-pointer"
                >
                  <Sparkles className="w-3.5 h-3.5" />
                  {seoBusy ? 'Writing SEO...' : 'Generate Description, Tags & Hashtags'}
                </button>
                <button
                  type="button"
                  onClick={handleSaveProjectSEO}
                  disabled={seoBusy}
                  className="px-3.5 py-1.5 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold text-nexa-text-primary flex items-center gap-1 cursor-pointer"
                >
                  <Save className="w-3.5 h-3.5" /> {seoSaved ? 'Saved!' : 'Save SEO'}
                </button>
              </div>
            </div>

            {seoError && (
              <div className="p-3.5 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-xs text-rose-500 flex items-center justify-between gap-3">
                <span><strong>AI SEO Error:</strong> {seoError}</span>
                <button
                  type="button"
                  onClick={() => navigate('/settings')}
                  className="px-3 py-1 rounded-full bg-rose-500 text-white text-[11px] font-extrabold shrink-0 cursor-pointer"
                >
                  Open AI Settings →
                </button>
              </div>
            )}

            <div>
              <div className="text-xs font-extrabold text-nexa-text-muted uppercase">
                Recommended YouTube Titles
              </div>
              <ul className="mt-2 space-y-2">
                {(proj.youtubeTitles || []).map((t: string, i: number) => (
                  <li
                    key={i}
                    className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border text-xs font-bold text-nexa-text-primary flex items-center justify-between gap-2"
                  >
                    <span>{i + 1}. {t}</span>
                    <button
                      type="button"
                      onClick={() => copyPrompt(t, `TITLE_${i}`)}
                      className="text-[11px] font-extrabold text-nexa-brand hover:underline shrink-0 cursor-pointer"
                    >
                      {copiedId === `TITLE_${i}` ? 'Copied!' : 'Copy'}
                    </button>
                  </li>
                ))}
              </ul>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <div className="text-xs font-extrabold text-nexa-text-muted uppercase">
                  Video Description &amp; Script Chapter Timestamps ({ytDescription.length} chars)
                </div>
                <button
                  type="button"
                  onClick={() => copyPrompt(ytDescription, 'YT_DESC')}
                  disabled={!ytDescription}
                  className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 disabled:opacity-40 cursor-pointer"
                >
                  <Copy className="w-3 h-3" /> {copiedId === 'YT_DESC' ? 'Copied Description!' : 'Copy Description'}
                </button>
              </div>
              <textarea
                value={ytDescription}
                onChange={(e) => setYtDescription(e.target.value)}
                rows={9}
                placeholder="Click 'Generate Description, Tags & Hashtags' to write a YouTube description synced with your script chapters..."
                className="w-full p-3.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs leading-relaxed text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-extrabold text-nexa-text-muted uppercase flex items-center gap-1">
                    <Tag className="w-3.5 h-3.5 text-nexa-brand" />
                    High-Performing Tags ({parsedYtTags.length})
                  </span>
                  <div className="flex items-center gap-2">
                    <span
                      className={`text-[10px] font-extrabold px-2 py-0.5 rounded-full border ${
                        ytTagsCharCount > 500
                          ? 'bg-rose-500/10 text-rose-500 border-rose-500/25'
                          : 'bg-emerald-500/10 text-emerald-500 border-emerald-500/25'
                      }`}
                    >
                      {ytTagsCharCount}/500
                    </span>
                    <button
                      type="button"
                      onClick={() => copyPrompt(parsedYtTags.join(', '), 'YT_TAGS')}
                      disabled={parsedYtTags.length === 0}
                      className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 disabled:opacity-40 cursor-pointer"
                    >
                      <Copy className="w-3 h-3" /> {copiedId === 'YT_TAGS' ? 'Copied!' : 'Copy'}
                    </button>
                  </div>
                </div>
                <textarea
                  value={ytTagsInput}
                  onChange={(e) => setYtTagsInput(e.target.value)}
                  rows={4}
                  placeholder="Comma-separated high-performing YouTube tags..."
                  className="w-full p-3 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
                />
                {parsedYtTags.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                    {parsedYtTags.map((t, idx) => (
                      <span
                        key={idx}
                        className="px-2.5 py-0.5 rounded-full bg-nexa-bg-base border border-nexa-border text-[11px] font-semibold text-nexa-text-secondary"
                      >
                        {t}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-extrabold text-nexa-text-muted uppercase flex items-center gap-1">
                    <Hash className="w-3.5 h-3.5 text-nexa-brand" />
                    Hashtags ({parsedYtHashtags.length})
                  </span>
                  <button
                    type="button"
                    onClick={() => copyPrompt(parsedYtHashtags.join(' '), 'YT_HASH')}
                    disabled={parsedYtHashtags.length === 0}
                    className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 disabled:opacity-40 cursor-pointer"
                  >
                    <Copy className="w-3 h-3" /> {copiedId === 'YT_HASH' ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <textarea
                  value={ytHashtagsInput}
                  onChange={(e) => setYtHashtagsInput(e.target.value)}
                  rows={4}
                  placeholder="#Documentary #Biography #TrueStory"
                  className="w-full p-3 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
                />
                {parsedYtHashtags.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {parsedYtHashtags.map((h, idx) => (
                      <span
                        key={idx}
                        className="px-2.5 py-0.5 rounded-full bg-nexa-brand/10 border border-nexa-brand/20 text-[11px] font-extrabold text-nexa-brand"
                      >
                        {h}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div>
              <div className="text-xs font-extrabold text-nexa-text-muted uppercase">Thumbnail Concept</div>
              <p className="mt-1 p-3.5 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/20 text-xs text-nexa-text-primary">
                {proj.thumbnailConcept}
              </p>
            </div>
            <div>
              <div className="text-xs font-extrabold text-nexa-text-muted uppercase">
                Music & Sound Design Direction
              </div>
              <p className="mt-1 p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border text-xs text-nexa-text-secondary">
                {proj.musicSuggestions}
              </p>
            </div>
          </div>

          <div className="nexa-card rounded-3xl p-6 space-y-3">
            <h2 className="text-base font-black text-display text-nexa-text-primary">
              Production Pipeline Checklist
            </h2>
            {[
              { label: 'Deep Biographical Research & Claims', count: `${(research.claims || []).length} Verified Claims`, tab: 'research' },
              { label: 'Realigned Documentary Script (≤7s Pace)', count: `v${scr.currentVersion} (${scr.shotCount || segs.length} shots)`, tab: 'script' },
              { label: 'Character Consistency Sheets', count: `${chars.length} Age/Era Locks`, tab: 'characters' },
              { label: '≤7s Visual Timeline & Prompts', count: `${segs.length} Timed Cuts`, tab: 'timeline' },
              { label: 'Storyboard & Generated Frames', count: `${segs.length} 16:9 Assets`, tab: 'images' },
              { label: 'Narration Audio & Alignment', count: `${audios.length} Master Track`, tab: 'audio' },
              { label: 'SRT Subtitles', count: `${subs.length} Timed Track`, tab: 'subtitles' },
              { label: 'Production ZIP & MP4 Exports', count: `${exportsList.length} Archives`, tab: 'export' },
            ].map((row) => (
              <div
                key={row.label}
                onClick={() => setTab(row.tab)}
                className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border hover:border-nexa-brand cursor-pointer flex items-center justify-between transition"
              >
                <span className="text-xs font-bold text-nexa-text-primary">{row.label}</span>
                <span className="text-[11px] font-extrabold text-emerald-500 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-0.5 rounded-full">
                  {row.count}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TAB 2: SCRIPT EDITOR */}
      {activeTab === 'script' && (
        <div className="space-y-6">
          {/* STAGE 0: PRE-SCRIPT YOUTUBE COMPETITOR TRANSCRIPT & OUTPERFORMANCE BRIEF */}
          <div className="nexa-card rounded-3xl p-6 space-y-4">
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="px-2.5 py-0.5 rounded-full bg-rose-500/10 text-rose-500 border border-rose-500/25 text-[10px] font-extrabold uppercase">
                    Stage 0 • Pre-Script Intelligence
                  </span>
                  {proj.competitiveBrief?.analyzedVideoCount > 0 && (
                    <>
                      <span className="px-2.5 py-0.5 rounded-full bg-nexa-bg-base border border-nexa-border text-[10px] font-bold text-nexa-text-secondary">
                        {proj.competitiveBrief.analyzedVideoCount} Videos Studied
                      </span>
                      <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[10px] font-extrabold">
                        {proj.competitiveBrief.transcribedVideoCount || 0} Transcribed
                      </span>
                      <span className="px-2.5 py-0.5 rounded-full bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20 text-[10px] font-extrabold">
                        {proj.competitiveBrief.commentsAnalyzedCount || 0} Viewer Comments Mined
                      </span>
                      <span
                        className={`px-2.5 py-0.5 rounded-full border text-[10px] font-extrabold ${
                          proj.competitiveBrief.usedGemini
                            ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/25'
                            : 'bg-amber-500/10 text-amber-500 border-amber-500/25'
                        }`}
                      >
                        {proj.competitiveBrief.usedGemini ? 'Live AI Strategy' : 'Legacy Cached Strategy'}
                      </span>
                    </>
                  )}
                </div>
                <h2 className="text-base font-black text-display text-nexa-text-primary mt-1.5">
                  YouTube Competitor Transcript Analysis &amp; Outperformance Strategy
                </h2>
                <p className="text-xs text-nexa-text-muted mt-0.5">
                  Automatically transcribes top competing YouTube videos &amp; mines viewer comments before writing so this script outperforms them on hooks, depth, and retention.
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2 shrink-0">
                <button
                  type="button"
                  onClick={handleGenerateCompetitiveBrief}
                  disabled={briefBusy || busy}
                  className="px-4 py-2 rounded-full border border-rose-500/40 bg-rose-500/10 hover:bg-rose-500/20 text-rose-500 text-xs font-extrabold flex items-center gap-1.5 cursor-pointer w-fit"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${briefBusy ? 'animate-spin' : ''}`} />
                  {briefBusy ? 'Transcribing & Studying Videos...' : proj.competitiveBrief ? 'Refresh Competitor Study' : 'Run Competitor Study'}
                </button>
                {proj.competitiveBrief && (
                  <button
                    type="button"
                    onClick={handleRegenerateScript}
                    disabled={busy || briefBusy}
                    className="px-4 py-2 rounded-full bg-nexa-brand hover:bg-nexa-brand-hover text-white text-xs font-extrabold flex items-center gap-1.5 cursor-pointer w-fit disabled:opacity-50"
                  >
                    <Sparkles className="w-3.5 h-3.5" />
                    {busy ? 'Rewriting Script...' : 'Apply Brief & Rewrite Script'}
                  </button>
                )}
              </div>
            </div>

            {briefError && (
              <div className="p-3.5 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-xs text-rose-500 flex items-center justify-between gap-3">
                <span><strong>Stage 0 AI Error:</strong> {briefError}</span>
                <button
                  type="button"
                  onClick={() => navigate('/settings')}
                  className="px-3 py-1 rounded-full bg-rose-500 text-white text-[11px] font-extrabold shrink-0 cursor-pointer"
                >
                  Open AI Settings →
                </button>
              </div>
            )}

            {proj.competitiveBrief ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
                  <div className="p-4 rounded-2xl bg-nexa-brand/5 border border-nexa-brand/20 space-y-1">
                    <div className="text-[10px] font-extrabold uppercase text-nexa-brand">Outperformance Strategy</div>
                    <p className="text-xs text-nexa-text-primary leading-relaxed">
                      {proj.competitiveBrief.outperformanceStrategy}
                    </p>
                  </div>
                  <div className="p-4 rounded-2xl bg-emerald-500/5 border border-emerald-500/20 space-y-1">
                    <div className="text-[10px] font-extrabold uppercase text-emerald-500">Cold Open Edge (0:00 – 1:30)</div>
                    <p className="text-xs text-nexa-text-primary leading-relaxed">
                      {proj.competitiveBrief.recommendedColdOpenEdge}
                    </p>
                  </div>
                  <div className="p-4 rounded-2xl bg-purple-500/5 border border-purple-500/20 space-y-1">
                    <div className="text-[10px] font-extrabold uppercase text-purple-500">Format Advantage</div>
                    <p className="text-xs text-nexa-text-primary leading-relaxed">
                      {proj.competitiveBrief.formatAdvantage}
                    </p>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border space-y-2">
                    <div className="text-[11px] font-extrabold uppercase text-emerald-500">What&apos;s Working in Top Outliers</div>
                    <ul className="space-y-1.5 text-xs text-nexa-text-secondary">
                      {(proj.competitiveBrief.winningHooks || proj.competitiveBrief.whatsWorking || []).map((item: string, idx: number) => (
                        <li key={idx} className="leading-relaxed">• {item}</li>
                      ))}
                    </ul>
                  </div>
                  <div className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border space-y-2">
                    <div className="text-[11px] font-extrabold uppercase text-amber-500">Competitor Blind Spots to Exploit</div>
                    <ul className="space-y-1.5 text-xs text-nexa-text-secondary">
                      {(proj.competitiveBrief.competitorBlindSpots || []).map((item: string, idx: number) => (
                        <li key={idx} className="leading-relaxed">• {item}</li>
                      ))}
                    </ul>
                  </div>
                  <div className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border space-y-2">
                    <div className="text-[11px] font-extrabold uppercase text-nexa-brand">Unanswered Viewer Questions</div>
                    <ul className="space-y-1.5 text-xs text-nexa-text-secondary">
                      {(proj.competitiveBrief.audienceQuestions || []).map((item: string, idx: number) => (
                        <li key={idx} className="leading-relaxed">• {item}</li>
                      ))}
                    </ul>
                  </div>
                </div>

                {(proj.competitiveBrief.scriptExecutionPlan || []).length > 0 && (
                  <div className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border space-y-2">
                    <div className="text-[11px] font-extrabold uppercase text-nexa-text-primary">
                      How This Outperformance Strategy Maps Into Your Script
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-2.5">
                      {(proj.competitiveBrief.scriptExecutionPlan || []).map((step: string, idx: number) => (
                        <div key={idx} className="p-3 rounded-xl bg-nexa-bg-card border border-nexa-border text-xs text-nexa-text-secondary leading-relaxed">
                          {step}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {(proj.competitiveBrief.studiedVideos || []).length > 0 && (
                  <div className="space-y-2 pt-1">
                    <div className="flex flex-wrap gap-2">
                      {(proj.competitiveBrief.studiedVideos || []).map((sv: any, idx: number) => (
                        <div
                          key={idx}
                          className="px-3 py-1.5 rounded-full bg-nexa-bg-base border border-nexa-border text-[11px] font-bold text-nexa-text-primary inline-flex items-center gap-2"
                        >
                          <a
                            href={sv.url}
                            target="_blank"
                            rel="noreferrer"
                            className="truncate max-w-[210px] hover:text-nexa-brand inline-flex items-center gap-1"
                          >
                            <span className="truncate">{sv.title}</span>
                            <ExternalLink className="w-3 h-3 text-nexa-text-muted shrink-0" />
                          </a>
                          <span className="text-emerald-500 font-extrabold">{Number(sv.channelMultiplier || 1).toFixed(1)}x</span>
                          {sv.transcriptAvailable && (
                            <button
                              type="button"
                              onClick={() => setExpandedTranscriptIdx(expandedTranscriptIdx === idx ? null : idx)}
                              className="px-2 py-0.5 rounded-full bg-emerald-500/15 hover:bg-emerald-500/25 text-emerald-500 text-[9px] font-extrabold cursor-pointer"
                            >
                              {expandedTranscriptIdx === idx ? 'HIDE TRANSCRIPT' : 'READ TRANSCRIPT'}
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                    {expandedTranscriptIdx !== null && proj.competitiveBrief.studiedVideos[expandedTranscriptIdx]?.transcriptExcerpt && (
                      <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-emerald-500/25 text-xs text-nexa-text-secondary space-y-1">
                        <div className="text-[10px] font-extrabold uppercase text-emerald-500">
                          Transcribed Competitor Excerpt — {proj.competitiveBrief.studiedVideos[expandedTranscriptIdx].channelName}
                        </div>
                        <p className="italic leading-relaxed">
                          &ldquo;{proj.competitiveBrief.studiedVideos[expandedTranscriptIdx].transcriptExcerpt}&rdquo;
                        </p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ) : (
              <div className="p-4 rounded-2xl border border-dashed border-nexa-border bg-nexa-bg-base/40 text-xs text-nexa-text-secondary">
                No pre-script competitor study cached yet for this project. Click <strong>Run Competitor Study</strong> above or click <strong>Rerun Script Processing</strong> below to automatically transcribe top YouTube videos, mine viewer comments, and position the script to outperform them.
              </div>
            )}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 nexa-card rounded-3xl p-6 space-y-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="text-base font-black text-display text-nexa-text-primary">
                  Realigned Biographical Documentary Script (v{scr.currentVersion} • {scr.status})
                </h2>
                <p className="text-xs text-nexa-text-muted mt-0.5">
                  Side-by-side narrator monologue &amp; visual shots paced at a maximum of 7 seconds per image.
                </p>
                <div className="flex flex-wrap items-center gap-2 mt-2.5">
                  {scr.generationMeta?.usedGemini ? (
                    <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/15 text-emerald-500 border border-emerald-500/30 text-[10px] font-extrabold">
                      ✓ Written by Live AI ({scr.generationMeta.model || scr.generationMeta.engine || 'Active Model'})
                    </span>
                  ) : (
                    <span className="px-2.5 py-0.5 rounded-full bg-rose-500/10 text-rose-500 border border-rose-500/30 text-[10px] font-extrabold">
                      ⚠ Legacy Script (Re-run with Live AI)
                    </span>
                  )}
                  {scr.generationMeta?.openingArchetype && (
                    <span className="px-2.5 py-0.5 rounded-full bg-purple-500/10 text-purple-500 border border-purple-500/25 text-[10px] font-extrabold">
                      Opening: {scr.generationMeta.openingArchetype}
                    </span>
                  )}
                  {scr.biographicalField && (
                    <span className="px-2.5 py-0.5 rounded-full bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20 text-[10px] font-extrabold">
                      Field: {scr.biographicalField}
                    </span>
                  )}
                  <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[10px] font-extrabold">
                    Max Cut Pace: ≤{scr.maxShotDurationSeconds ? Number(scr.maxShotDurationSeconds).toFixed(0) : 7}s / Image
                  </span>
                  <span className="px-2.5 py-0.5 rounded-full bg-nexa-bg-base text-nexa-text-secondary border border-nexa-border text-[10px] font-bold">
                    {scr.shotCount || segs.length} Visual Cuts
                  </span>
                  <span className="px-2.5 py-0.5 rounded-full bg-nexa-bg-base text-nexa-text-secondary border border-nexa-border text-[10px] font-bold">
                    {scr.wordCount || 0} Spoken Words
                  </span>
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <select
                  value={biographicalField}
                  onChange={(e) => setBiographicalField(e.target.value)}
                  className="px-3 py-1.5 rounded-full border border-nexa-border bg-nexa-bg-base text-xs font-bold"
                  title="Dynamic Biographical Field"
                >
                  <option value="auto">Field: Auto-Detect</option>
                  <option value="Technology & Innovation">Technology &amp; Innovation</option>
                  <option value="Business & Founders">Business &amp; Founders</option>
                  <option value="Science & Medicine">Science &amp; Medicine</option>
                  <option value="Politics, Law & Governance">Politics, Law &amp; Governance</option>
                  <option value="Sports & Competition">Sports &amp; Competition</option>
                  <option value="Arts, Film & Music">Arts, Film &amp; Music</option>
                  <option value="Humanitarian & Civic Impact">Humanitarian &amp; Civic Impact</option>
                </select>
                <select
                  value={narrativeStyle}
                  onChange={(e) => setNarrativeStyle(e.target.value)}
                  className="px-3 py-1.5 rounded-full border border-nexa-border bg-nexa-bg-base text-xs font-bold"
                  title="Documentary Narrative Style"
                >
                  <option value="Investigative Archival Documentary">Investigative Archival</option>
                  <option value="Biographical Portrait (Restrained & Journalistic)">Restrained Journalistic</option>
                  <option value="Chronological Reckoning & Public Record">Chronological Reckoning</option>
                </select>
                <select
                  value={durationMinutes}
                  onChange={(e) => setDurationMinutes(Number(e.target.value))}
                  className="px-3 py-1.5 rounded-full border border-nexa-border bg-nexa-bg-base text-xs font-bold"
                >
                  <option value={15}>15 Minutes</option>
                  <option value={20}>20 Minutes (Default)</option>
                  <option value={25}>25 Minutes</option>
                </select>
                <button
                  onClick={handleRegenerateScript}
                  disabled={busy}
                  className="px-3.5 py-1.5 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-bold flex items-center gap-1.5 cursor-pointer"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${busy ? 'animate-spin' : ''}`} />
                  {busy ? 'Writing Chapters...' : 'Rerun Script Processing'}
                </button>
                <button
                  onClick={handleSaveScript}
                  disabled={busy}
                  className="px-3.5 py-1.5 rounded-full bg-nexa-brand text-white text-xs font-extrabold flex items-center gap-1 cursor-pointer"
                >
                  <Save className="w-3.5 h-3.5" /> Save Version
                </button>
                <button
                  onClick={handleApproveScript}
                  disabled={busy}
                  className="px-3.5 py-1.5 rounded-full bg-emerald-600 text-white text-xs font-extrabold flex items-center gap-1 cursor-pointer"
                >
                  <CheckCircle2 className="w-3.5 h-3.5" /> Approve Script
                </button>
              </div>
            </div>

            {scriptError && (
              <div className="p-3.5 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-xs text-rose-500 flex items-center justify-between gap-3">
                <span><strong>Script Generation Blocked:</strong> {scriptError}</span>
                <button
                  type="button"
                  onClick={() => navigate('/settings')}
                  className="px-3 py-1 rounded-full bg-rose-500 text-white text-[11px] font-extrabold shrink-0 cursor-pointer"
                >
                  Fix AI Key →
                </button>
              </div>
            )}

            {!scriptError && scr.generationMeta && !scr.generationMeta.usedGemini && scr.generationMeta.aiError && (
              <div className="p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/30 text-xs text-amber-600 dark:text-amber-400 flex items-center justify-between gap-3">
                <span>
                  <strong>Why this script used local fallback instead of Gemini AI:</strong> {scr.generationMeta.aiError}
                </span>
                <button
                  type="button"
                  onClick={() => navigate('/settings')}
                  className="px-3 py-1 rounded-full bg-amber-500 text-white text-[11px] font-extrabold shrink-0 cursor-pointer"
                >
                  Open AI Settings →
                </button>
              </div>
            )}

            {scr.generationMeta?.usedGemini && scr.generationMeta?.underdogAngle && (
              <div className="p-3.5 rounded-2xl bg-emerald-500/5 border border-emerald-500/25 text-xs text-nexa-text-primary">
                <span className="font-extrabold text-emerald-500 uppercase text-[10px] mr-2">
                  AI Underdog Angle ({scr.generationMeta.transcriptsUsed || 0} Transcripts • {scr.generationMeta.commentsUsed || 0} Comments):
                </span>
                {scr.generationMeta.underdogAngle}
              </div>
            )}

            <div className="flex items-center justify-between gap-2 pt-1">
              <div className="inline-flex items-center p-1 rounded-full bg-nexa-bg-base border border-nexa-border">
                <button
                  type="button"
                  onClick={() => setScriptViewMode('realigned')}
                  className={`px-3.5 py-1 rounded-full text-xs font-extrabold cursor-pointer transition ${
                    scriptViewMode === 'realigned'
                      ? 'bg-nexa-brand text-white'
                      : 'text-nexa-text-secondary hover:text-nexa-text-primary'
                  }`}
                >
                  Realigned Shot-by-Shot View ({(scr.sections || []).length} Chapters)
                </button>
                <button
                  type="button"
                  onClick={() => setScriptViewMode('raw')}
                  className={`px-3.5 py-1 rounded-full text-xs font-extrabold cursor-pointer transition ${
                    scriptViewMode === 'raw'
                      ? 'bg-nexa-brand text-white'
                      : 'text-nexa-text-secondary hover:text-nexa-text-primary'
                  }`}
                >
                  Raw Master Script Editor
                </button>
              </div>
              <button
                type="button"
                onClick={() => copyPrompt(scriptText, 'FULL_SCRIPT')}
                className="px-3 py-1.5 rounded-full border border-nexa-border hover:border-nexa-brand text-[11px] font-bold flex items-center gap-1 cursor-pointer"
              >
                <Copy className="w-3 h-3" /> {copiedId === 'FULL_SCRIPT' ? 'Copied Full Script!' : 'Copy Realigned Script'}
              </button>
            </div>

            {scriptViewMode === 'realigned' && (scr.sections || []).length > 0 ? (
              <div className="space-y-5 max-h-[680px] overflow-y-auto pr-1">
                {(scr.sections || []).map((sec: any, sIdx: number) => (
                  <div key={sIdx} className="rounded-2xl border border-nexa-border bg-nexa-bg-base p-4 space-y-3">
                    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-nexa-border pb-2.5">
                      <h3 className="text-sm font-black text-display text-nexa-text-primary">
                        {sec.title} {sec.timeRange ? `(${sec.timeRange})` : ''}
                      </h3>
                      <span className="text-[11px] font-extrabold text-nexa-brand">
                        {(sec.shots || []).length} cuts • ≤7s/cut
                      </span>
                    </div>
                    <div className="space-y-2.5">
                      {(sec.shots || []).map((sh: any, shIdx: number) => (
                        <div
                          key={shIdx}
                          className="p-3 rounded-xl bg-nexa-bg-surface border border-nexa-border space-y-1.5"
                        >
                          <div className="flex flex-wrap items-center gap-2 text-[11px]">
                            <span className="px-2 py-0.5 rounded-md bg-nexa-brand/15 text-nexa-brand font-extrabold font-mono">
                              [{sh.timestamp}]
                            </span>
                            <span className="px-2 py-0.5 rounded-md bg-emerald-500/10 text-emerald-500 font-extrabold">
                              VISUAL
                            </span>
                            <span className="text-nexa-text-muted font-semibold ml-auto">
                              {sh.durationSeconds}s
                            </span>
                          </div>
                          <p className="text-xs text-nexa-text-secondary leading-relaxed">
                            {sh.visual}
                          </p>
                          {sh.narration ? (
                            <p className="text-xs text-nexa-text-primary font-medium leading-relaxed">
                              <strong className="text-nexa-brand">NARRATOR (V.O.):</strong> {sh.narration}
                            </p>
                          ) : (
                            <p className="text-xs italic text-nexa-text-muted">
                              (Audio: {sh.audioCue || 'Silence'})
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <textarea
                value={scriptText}
                onChange={(e) => setScriptText(e.target.value)}
                rows={22}
                className="w-full p-4 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs font-mono leading-relaxed text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
              />
            )}

            <div className="space-y-2 pt-2">
              <div className="text-xs font-extrabold text-nexa-text-primary">
                Shot &amp; Block-Level Source Refinement
              </div>
              <div className="space-y-2 max-h-56 overflow-y-auto">
                {scriptText.split('\n\n').slice(0, 40).map((para, idx) => (
                  <div
                    key={idx}
                    className="p-3 rounded-2xl bg-nexa-bg-base border border-nexa-border flex items-start justify-between gap-3"
                  >
                    <p className="text-xs text-nexa-text-secondary line-clamp-2 flex-1">
                      <strong>#{idx + 1}:</strong> {para}
                    </p>
                    <button
                      onClick={() => handleRegenerateParagraph(idx)}
                      disabled={busy}
                      className="px-3 py-1 rounded-full border border-nexa-border bg-nexa-bg-surface hover:border-nexa-brand text-[11px] font-bold shrink-0 cursor-pointer"
                    >
                      Refine #{idx + 1}
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="nexa-card rounded-3xl p-6 space-y-4">
            <h2 className="text-base font-black text-display text-nexa-text-primary">Revision History</h2>
            <div className="space-y-2.5">
              {(scr.versions || []).map((v: any) => (
                <div
                  key={v.id}
                  onClick={() => setScriptText(v.fullNarrationText)}
                  className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border hover:border-nexa-brand cursor-pointer"
                >
                  <div className="flex items-center justify-between text-xs font-extrabold text-nexa-text-primary">
                    <span>Version {v.versionNumber}</span>
                    <span className="text-[11px] text-nexa-brand">{v.wordCount} words</span>
                  </div>
                  <div className="text-xs text-nexa-text-secondary mt-1">{v.changeSummary}</div>
                  <div className="text-[10px] text-nexa-text-muted mt-1">
                    {v.createdBy} • {new Date(v.createdAt).toLocaleString()}
                  </div>
                </div>
              ))}
            </div>
          </div>
          </div>
        </div>
      )}

      {/* TAB 3: RESEARCH DOSSIER */}
      {activeTab === 'research' && (
        <div className="nexa-card rounded-3xl p-6 space-y-6">
          <div>
            <h2 className="text-base font-black text-display text-nexa-text-primary">
              Source-Grounded Biographical Dossier: {research.canonicalName}
            </h2>
            <p className="text-xs text-nexa-text-secondary mt-2 whitespace-pre-line leading-relaxed">
              {research.biography}
            </p>
          </div>
          <div>
            <h3 className="text-sm font-black text-display text-nexa-text-primary mb-3">
              Supporting Claims & Citations
            </h3>
            <div className="space-y-2.5">
              {(research.claims || []).map((c: any) => (
                <div
                  key={c.id}
                  className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border flex items-center justify-between gap-4"
                >
                  <div>
                    <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[10px] font-extrabold mr-2">
                      {c.verificationStatus}
                    </span>
                    <span className="text-xs font-bold text-nexa-text-primary">{c.category}: </span>
                    <span className="text-xs text-nexa-text-secondary">{c.claimText}</span>
                  </div>
                  {c.sourceUrl && (
                    <a
                      href={c.sourceUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="text-xs font-extrabold text-nexa-brand shrink-0"
                    >
                      Source →
                    </a>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 4: VISUAL TIMELINE & GOOGLE FLOW WORKFLOW */}
      {activeTab === 'timeline' && (
        <div className="space-y-4">
          <div className="nexa-card rounded-3xl p-6 flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 className="text-base font-black text-display text-nexa-text-primary">
                ≤7 Second Visual Timeline &amp; Google Flow Prompt Matrix ({segs.length} Continuous Cuts)
              </h2>
              <p className="text-xs text-nexa-text-secondary">
                Every ≤7-second narration beat is paired with a domain-matched Character Reference, Location Reference, and copyable Google Flow 16:9 prompt.
              </p>
            </div>
            <div className="flex gap-2.5">
              <button
                onClick={() => {
                  const allPrompts = segs
                    .map((s: any) => `[${s.segmentCode} | ${s.startFormatted} -> ${s.endFormatted}]\n${s.imagePrompt}`)
                    .join('\n\n');
                  copyPrompt(allPrompts, 'ALL_PROMPTS');
                }}
                className="px-4 py-2 rounded-full bg-nexa-brand text-white text-xs font-extrabold flex items-center gap-1.5 shadow-md shadow-nexa-brand/25 cursor-pointer"
              >
                <Copy className="w-3.5 h-3.5" />
                {copiedId === 'ALL_PROMPTS' ? 'Copied All Google Flow Prompts!' : 'Copy All Prompts for Google Flow'}
              </button>
              <button
                onClick={handleRebuildTimeline}
                disabled={busy}
                className="px-4 py-2 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold cursor-pointer"
              >
                Re-Segment Timeline from Narration
              </button>
            </div>
          </div>

          <div className="space-y-3">
            {segs.map((s: any) => (
              <div
                key={s.id}
                className="nexa-card rounded-3xl p-5 grid grid-cols-1 lg:grid-cols-12 gap-4 items-center"
              >
                <div className="lg:col-span-2 space-y-1.5">
                  <div className="inline-block px-3 py-1 rounded-full bg-nexa-brand text-white text-xs font-extrabold">
                    {s.segmentCode}
                  </div>
                  <div className="text-xs font-bold text-nexa-text-primary">
                    {s.startFormatted} → {s.endFormatted}
                  </div>
                  <div className="text-[11px] text-nexa-text-muted">Duration: {s.durationSeconds}s</div>
                  <div className="text-[10px] font-extrabold text-emerald-500 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-0.5 rounded-full inline-block">
                    {s.characterAgeRange}
                  </div>
                </div>

                <div className="lg:col-span-4 space-y-1">
                  <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">
                    Continuous Narration Passage
                  </div>
                  <p className="text-xs text-nexa-text-primary leading-relaxed bg-nexa-bg-base p-3 rounded-2xl border border-nexa-border">
                    "{s.narrationPassage}"
                  </p>
                </div>

                <div className="lg:col-span-4 space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-extrabold uppercase text-nexa-text-muted">
                      Google Flow 16:9 Image Prompt
                    </span>
                    <button
                      onClick={() => copyPrompt(s.imagePrompt, s.id)}
                      className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 cursor-pointer"
                    >
                      <Copy className="w-3 h-3" /> {copiedId === s.id ? 'Copied!' : 'Copy Prompt'}
                    </button>
                  </div>
                  <p className="text-[11px] text-nexa-text-secondary leading-relaxed bg-nexa-brand/5 p-3 rounded-2xl border border-nexa-brand/20">
                    {s.imagePrompt}
                  </p>
                </div>

                <div className="lg:col-span-2">
                  {s.imageUrl && (
                    <img
                      src={s.imageUrl}
                      alt={s.segmentCode}
                      className="w-full aspect-video object-cover rounded-2xl border border-nexa-border"
                    />
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TAB 5: CHARACTER REFERENCES */}
      {activeTab === 'characters' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-base font-black text-display text-nexa-text-primary">
                Character Consistency Reference Package
              </h2>
              <p className="text-xs text-nexa-text-secondary">
                Maintains visual continuity across childhood, contender era, and championship breakthrough.
              </p>
            </div>
            <button
              onClick={handleRebuildCharacters}
              disabled={busy}
              className="px-4 py-2 rounded-full bg-nexa-brand text-white text-xs font-extrabold cursor-pointer"
            >
              Regenerate Character Sheets
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {chars.map((c: any) => (
              <div key={c.id} className="nexa-card rounded-3xl overflow-hidden flex flex-col">
                {c.referenceImagePath && (
                  <img
                    src={c.referenceImagePath}
                    alt={c.ageRange}
                    className="w-full aspect-video object-cover border-b border-nexa-border"
                  />
                )}
                <div className="p-5 space-y-2.5 flex-1 flex flex-col justify-between">
                  <div className="space-y-2">
                    <div className="text-xs font-extrabold uppercase text-nexa-brand">{c.role}</div>
                    <h3 className="text-base font-black text-display text-nexa-text-primary">
                      {c.characterName} — {c.ageRange}
                    </h3>
                    <p className="text-xs text-nexa-text-secondary"><strong>Appearance:</strong> {c.appearanceDescription}</p>
                    <p className="text-xs text-nexa-text-secondary"><strong>Hair:</strong> {c.hairDescription}</p>
                    <p className="text-xs text-nexa-text-secondary"><strong>Clothing:</strong> {c.clothingDescription}</p>
                    <p className="text-xs text-nexa-text-secondary"><strong>Locations:</strong> {c.locationsDescription}</p>
                    <p className="text-xs text-nexa-text-secondary"><strong>Palette:</strong> {c.colorPalette}</p>
                  </div>
                  <div className="pt-3 border-t border-nexa-border">
                    <button
                      onClick={() => copyPrompt(c.masterPrompt, c.id)}
                      className="w-full py-2.5 rounded-full bg-nexa-brand/10 hover:bg-nexa-brand hover:text-white text-nexa-brand text-xs font-extrabold border border-nexa-brand/20 transition cursor-pointer"
                    >
                      {copiedId === c.id ? 'Copied Character Lock Prompt!' : 'Copy Google Flow Character Lock'}
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TAB 6: IMAGES GALLERY */}
      {activeTab === 'images' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {segs.map((s: any) => (
            <div key={s.id} className="nexa-card rounded-3xl overflow-hidden">
              {s.imageUrl && <img src={s.imageUrl} alt={s.segmentCode} className="w-full aspect-video object-cover" />}
              <div className="p-4 space-y-1">
                <div className="flex items-center justify-between text-xs font-extrabold text-nexa-text-primary">
                  <span className="text-nexa-brand">{s.segmentCode}</span>
                  <span className="text-nexa-text-muted">{s.startFormatted} → {s.endFormatted}</span>
                </div>
                <p className="text-xs text-nexa-text-secondary line-clamp-2">{s.narrationPassage}</p>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* TAB 7: AUDIO STUDIO */}
      {activeTab === 'audio' && (
        <div className="nexa-card rounded-3xl p-6 space-y-6">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 className="text-base font-black text-display text-nexa-text-primary">Audio & Narration Studio</h2>
              <p className="text-xs text-nexa-text-secondary">
                Synthesize master documentary voiceover or upload recorded narration audio to align visual segment timestamps.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                onClick={handleGenerateAudio}
                disabled={busy}
                className="px-4 py-2.5 rounded-full bg-nexa-brand text-white text-xs font-extrabold flex items-center gap-1.5 shadow-md shadow-nexa-brand/25 cursor-pointer"
              >
                <Mic className="w-4 h-4" /> Synthesize Master Narration
              </button>
              <label className="px-4 py-2.5 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold cursor-pointer flex items-center gap-1.5">
                <Upload className="w-4 h-4" /> Upload Audio (.wav/.mp3)
                <input type="file" accept="audio/*" onChange={handleAudioUpload} className="hidden" />
              </label>
            </div>
          </div>

          <div className="space-y-3">
            {audios.map((a: any) => (
              <div
                key={a.id}
                className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border flex flex-col md:flex-row md:items-center justify-between gap-4"
              >
                <div>
                  <div className="text-sm font-bold text-nexa-text-primary">{a.title}</div>
                  <div className="text-xs text-nexa-text-muted">
                    Provider: {a.provider} • Voice: {a.voiceId} • Duration: {a.durationSeconds}s
                  </div>
                </div>
                <audio controls src={a.fileUrl} className="h-10" />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TAB 8: SUBTITLES */}
      {activeTab === 'subtitles' && (
        <div className="nexa-card rounded-3xl p-6 space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-base font-black text-display text-nexa-text-primary">Aligned SRT Subtitle Tracks</h2>
              <p className="text-xs text-nexa-text-secondary">Time-aligned with the ≤7 second visual cuts.</p>
            </div>
            {subs[0] && (
              <a
                href={subs[0].fileUrl}
                download
                className="px-4 py-2 rounded-full bg-nexa-brand text-white text-xs font-extrabold"
              >
                Download subtitles.srt →
              </a>
            )}
          </div>
          {subs[0] && (
            <pre className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border text-nexa-text-primary text-xs overflow-x-auto max-h-96">
              {subs[0].contentText}
            </pre>
          )}
        </div>
      )}

      {/* TAB 9: EXPORT & FFMPEG VIDEO ASSEMBLY */}
      {activeTab === 'export' && (
        <div className="space-y-6">
          <div className="nexa-card rounded-3xl p-6 space-y-4">
            <h2 className="text-base font-black text-display text-nexa-text-primary">
              Production Package Export & FFmpeg 16:9 Video Assembly
            </h2>
            <p className="text-xs text-nexa-text-secondary">
              Produces an organized ZIP archive containing <code>documentary-script.docx</code>, <code>realigned-script.md</code>, <code>narration.txt</code>,{' '}
              <code>youtube-seo-packaging.md</code>, <code>visual-timeline.csv</code>, <code>image-prompts.txt</code>, <code>subtitles.srt</code>,{' '}
              <code>production-manifest.json</code>, <code>research-sources.md</code>, <code>fact-check-report.md</code>, character reference sheets, storyboard frames, and narration audio.
            </p>
            <div className="flex flex-wrap gap-3">
              <button
                onClick={() => handleExportPackage(false)}
                disabled={busy}
                className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white font-extrabold text-xs shadow-md shadow-nexa-brand/25 cursor-pointer"
              >
                Generate Complete ZIP Production Package
              </button>
              <button
                onClick={() => handleExportPackage(true)}
                disabled={busy}
                className="px-5 py-2.5 rounded-full bg-emerald-600 hover:bg-emerald-500 text-white font-extrabold text-xs shadow-md shadow-emerald-600/20 cursor-pointer"
              >
                Generate ZIP + Assemble 16:9 MP4 Video via FFmpeg
              </button>
            </div>

            {exportResult && (
              <div className="p-4 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-600 dark:text-emerald-400 space-y-2">
                <div className="font-black">Production Package Built Successfully!</div>
                <div className="flex flex-wrap gap-4">
                  <a href={exportResult.packageZipUrl} download className="font-extrabold underline">
                    Download ZIP Archive ({Math.round((exportResult.fileSizeBytes || 0) / 1024)} KB)
                  </a>
                  {exportResult.videoMp4Url && (
                    <a href={exportResult.videoMp4Url} target="_blank" rel="noreferrer" className="font-extrabold underline">
                      Open Assembled 16:9 MP4 Video
                    </a>
                  )}
                </div>
              </div>
            )}
          </div>

          <div className="nexa-card rounded-3xl p-6 space-y-3">
            <h3 className="text-sm font-black text-display text-nexa-text-primary">Export History</h3>
            {exportsList.map((ex: any) => (
              <div
                key={ex.id}
                className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border flex items-center justify-between text-xs"
              >
                <div>
                  <span className="font-extrabold text-nexa-brand uppercase mr-2">{ex.exportType}</span>
                  <span className="text-nexa-text-muted">{new Date(ex.createdAt).toLocaleString()}</span>
                </div>
                <a
                  href={ex.fileUrl}
                  download
                  className="px-3.5 py-1.5 rounded-full bg-nexa-brand text-white font-extrabold"
                >
                  Download ({Math.round(Number(ex.fileSizeBytes || 0) / 1024)} KB)
                </a>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
