import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  ArrowRight, Bookmark, Compass, Film, Flame, Search, ShieldCheck,
  Sparkles, Trophy, TrendingUp, Activity, Bell, CheckCircle2, AlertCircle
} from 'lucide-react';
import { apiFetch, setAuthSession } from '../api/client';

export const StageBadge: React.FC<{ stage: string }> = ({ stage }) => {
  const styles: Record<string, string> = {
    'Major Viral Opportunity': 'bg-nexa-brand/10 text-nexa-brand border-nexa-brand/25',
    'Accelerating': 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/25',
    'Emerging': 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/25',
    'Early Whisper': 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/25',
    'Saturated': 'bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/25',
  };
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 text-[11px] font-extrabold rounded-full border ${
        styles[stage] || 'bg-nexa-bg-base text-nexa-text-secondary border-nexa-border'
      }`}
    >
      <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
      {stage}
    </span>
  );
};

// 1. LOGIN PAGE (ofia_ng Liquid Glass + Electric Blue)
export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const res = await apiFetch('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      });
      setAuthSession(res.token, res.refreshToken, res.user);
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    }
  };

  return (
    <div className="min-h-screen bg-nexa-bg-base flex items-center justify-center p-6">
      <div className="liquid-glass rounded-3xl border border-nexa-border max-w-md w-full p-8">
        <div className="flex items-center gap-3.5 mb-6">
          <img
            src="/logo.png"
            alt="TrendScout Studio"
            className="w-12 h-12 rounded-2xl object-contain shrink-0"
          />
          <div>
            <h1 className="text-xl font-black text-display text-nexa-text-primary">TrendScout Studio</h1>
            <p className="text-xs text-nexa-text-secondary font-medium">
              AI Documentary Discovery & Production OS
            </p>
          </div>
        </div>
        {error && (
          <div className="mb-4 p-3 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs font-bold">
            {error}
          </div>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary uppercase tracking-wider mb-1.5">
              Producer Email
            </label>
            <input
              type="email"
              placeholder="producer@trendscout.studio"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-3 border border-nexa-border bg-nexa-bg-surface rounded-2xl text-sm text-nexa-text-primary outline-none focus:border-nexa-brand"
            />
          </div>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary uppercase tracking-wider mb-1.5">
              Password
            </label>
            <input
              type="password"
              placeholder="••••••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 border border-nexa-border bg-nexa-bg-surface rounded-2xl text-sm text-nexa-text-primary outline-none focus:border-nexa-brand"
            />
          </div>
          <button
            type="submit"
            className="w-full py-3.5 rounded-2xl bg-nexa-brand hover:bg-nexa-brand-mid text-white font-extrabold text-sm shadow-lg shadow-nexa-brand/25 transition cursor-pointer"
          >
            Sign In to TrendScout Studio →
          </button>
        </form>
      </div>
    </div>
  );
};

// 2. DASHBOARD OVERVIEW PAGE (ofia_ng AdminStatGrid + NexaCard)
export const DashboardPage: React.FC = () => {
  const navigate = useNavigate();
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  const load = () => {
    setError(null);
    apiFetch('/api/dashboard/overview')
      .then(setData)
      .catch((err) => {
        console.error(err);
        setError(err.message || 'Unable to load overview intelligence');
      });
  };

  useEffect(() => {
    load();
    window.addEventListener('ts-refresh', load);
    return () => window.removeEventListener('ts-refresh', load);
  }, []);

  if (error && !data) {
    return (
      <div className="p-8 max-w-md mx-auto text-center space-y-4 my-12">
        <div className="w-12 h-12 rounded-2xl bg-amber-500/10 text-amber-500 flex items-center justify-center mx-auto">
          <AlertCircle className="w-6 h-6" />
        </div>
        <div>
          <h2 className="text-base font-black text-display text-nexa-text-primary">Connection Notice</h2>
          <p className="text-xs text-nexa-text-secondary mt-1">{error}</p>
        </div>
        <button
          onClick={load}
          className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white font-extrabold text-xs cursor-pointer shadow-lg shadow-nexa-brand/20 transition-all"
        >
          Retry Connection
        </button>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="p-16 flex flex-col items-center justify-center space-y-3">
        <div className="w-8 h-8 rounded-full border-2 border-nexa-brand border-t-transparent animate-spin" />
        <div className="text-xs text-nexa-text-muted font-bold tracking-wide">
          Loading TrendScout Studio intelligence...
        </div>
      </div>
    );
  }

  const m = data.metrics || {};
  const kpis = [
    {
      label: 'Discovered People',
      value: m.totalPeople ?? 0,
      change: 'Deduplicated',
      badgeClass: 'bg-nexa-brand/10 text-nexa-brand border-nexa-brand/20',
      icon: <Compass className="w-5 h-5 text-nexa-brand" />,
      sub: 'Resolved across aliases & events',
    },
    {
      label: 'Major Viral Stories',
      value: m.majorViralOpportunities ?? 0,
      change: 'Score > 92.0',
      badgeClass: 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20',
      icon: <Flame className="w-5 h-5 text-emerald-500" />,
      sub: `${m.acceleratingStories ?? 0} accelerating • ${m.emergingStories ?? 0} emerging`,
    },
    {
      label: 'Strong Underdogs',
      value: m.strongUnderdogs ?? 0,
      change: '96% Confidence',
      badgeClass: 'bg-purple-500/10 text-purple-500 border-purple-500/20',
      icon: <Trophy className="w-5 h-5 text-purple-500" />,
      sub: 'Source-grounded adversity arcs',
    },
    {
      label: 'Studio Production',
      value: `${m.scriptsGenerated ?? 0} Scripts`,
      change: `${m.completedPackages ?? 0} Exports`,
      badgeClass: 'bg-amber-500/10 text-amber-500 border-amber-500/20',
      icon: <Film className="w-5 h-5 text-amber-500" />,
      sub: '20-min scripts & 5–10s Google Flow beats',
    },
  ];

  return (
    <div className="space-y-7">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-display tracking-tight text-nexa-text-primary">
            TrendScout Master Intelligence Hub
          </h1>
          <p className="text-xs text-nexa-text-secondary mt-1">
            Real-Time Autonomous Discovery Engine • Last Scan:{' '}
            <span className="font-bold text-nexa-brand">
              {data.monitoringStatus?.lastScanAt
                ? new Date(data.monitoringStatus.lastScanAt).toLocaleString()
                : 'Active'}
            </span>
          </p>
        </div>
        <div className="flex items-center gap-2.5">
          <button
            onClick={() => navigate('/production')}
            className="px-4 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold flex items-center gap-2 shadow-md shadow-nexa-brand/25 transition cursor-pointer"
          >
            <Film className="w-4 h-4" />
            Open Production Studio →
          </button>
        </div>
      </div>

      {/* OFIA_NG ADMIN STAT GRID */}
      <section className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
        {kpis.map((kpi, i) => (
          <div
            key={i}
            className="nexa-card p-5 sm:p-6 relative overflow-hidden group hover:border-nexa-brand/40 rounded-3xl"
          >
            <div className="flex items-center justify-between mb-4">
              <div className="w-11 h-11 rounded-2xl bg-nexa-brand/10 flex items-center justify-center group-hover:scale-110 transition-transform">
                {kpi.icon}
              </div>
              <span className={`text-[10px] font-extrabold px-2.5 py-0.5 rounded-full border ${kpi.badgeClass}`}>
                {kpi.change}
              </span>
            </div>
            <p className="text-xs text-nexa-text-faint font-bold uppercase tracking-wider mb-1">
              {kpi.label}
            </p>
            <h3 className="text-2xl font-extrabold text-display text-nexa-text-primary mb-1">
              {kpi.value}
            </h3>
            <p className="text-[11px] text-nexa-text-secondary font-medium">{kpi.sub}</p>
          </div>
        ))}
      </section>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 nexa-card rounded-3xl p-6">
          <div className="flex items-center justify-between mb-5">
            <div>
              <h2 className="text-base font-black text-display text-nexa-text-primary">
                Ranked Underdog Documentary Opportunities
              </h2>
              <p className="text-xs text-nexa-text-muted">
                Real-time multi-platform momentum & biographical adversity scoring
              </p>
            </div>
            <button
              onClick={() => navigate('/underdogs')}
              className="px-3.5 py-1.5 rounded-full bg-nexa-brand/10 text-nexa-brand text-xs font-extrabold hover:bg-nexa-brand hover:text-white transition cursor-pointer"
            >
              View All →
            </button>
          </div>

          <div className="space-y-3.5">
            {(data.topDiscoveries || []).map((item: any) => (
              <div
                key={item.id}
                onClick={() => navigate(`/opportunities/${item.id}`)}
                className="p-4 rounded-2xl border border-nexa-border bg-nexa-bg-base/50 hover:border-nexa-brand/50 cursor-pointer transition flex flex-col md:flex-row md:items-center justify-between gap-4"
              >
                <div className="space-y-1.5">
                  <div className="flex items-center gap-2.5 flex-wrap">
                    <span className="font-extrabold text-display text-nexa-text-primary text-base">
                      {item.personName}
                    </span>
                    <StageBadge stage={item.trendStage} />
                    <span className="px-2.5 py-0.5 text-[10px] font-extrabold rounded-full bg-nexa-bg-surface border border-nexa-border text-nexa-text-secondary">
                      {item.underdogClassification}
                    </span>
                  </div>
                  <p className="text-xs text-nexa-text-secondary line-clamp-2">{item.achievement}</p>
                  <div className="flex items-center gap-4 text-[11px] text-nexa-text-muted font-semibold pt-1">
                    <span>🔥 {Number(item.viewsPerHour).toLocaleString()} views/hr</span>
                    <span>📈 {item.channelRelativePerf}x channel baseline</span>
                    <span>📰 {item.independentCoverageCount} independent sources</span>
                  </div>
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <div className="text-right px-3.5 py-2 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/20">
                    <div className="text-[9px] font-extrabold uppercase text-nexa-brand">Score</div>
                    <div className="text-xl font-black text-display text-nexa-brand">
                      {Number(item.overallScore).toFixed(1)}
                    </div>
                  </div>
                  <ArrowRight className="w-4 h-4 text-nexa-text-faint" />
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          <div className="nexa-card rounded-3xl p-6">
            <h2 className="text-sm font-black text-display text-nexa-text-primary mb-4">
              Source Connector Health
            </h2>
            <div className="space-y-3">
              {(data.connectors || []).map((c: any) => (
                <div
                  key={c.id}
                  className="flex items-center justify-between text-xs border-b border-nexa-border pb-2.5 last:border-0"
                >
                  <div>
                    <div className="font-bold text-nexa-text-primary">{c.name}</div>
                    <div className="text-[11px] text-nexa-text-muted">
                      Quota: {c.quotaUsed} / {c.quotaLimit}
                    </div>
                  </div>
                  <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 font-extrabold text-[10px] uppercase">
                    {c.status}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="nexa-card rounded-3xl p-6">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-black text-display text-nexa-text-primary">
                Ecosystem Live Alerts
              </h2>
              <button
                onClick={() => navigate('/notifications')}
                className="text-xs font-extrabold text-nexa-brand cursor-pointer"
              >
                All Alerts
              </button>
            </div>
            <div className="space-y-2.5">
              {(data.recentNotifications || []).slice(0, 4).map((n: any) => (
                <div
                  key={n.id}
                  onClick={() => navigate(n.actionUrl || '/')}
                  className="p-3 rounded-2xl bg-nexa-bg-base hover:border-nexa-brand/40 cursor-pointer border border-nexa-border transition"
                >
                  <div className="text-xs font-bold text-nexa-text-primary">{n.title}</div>
                  <div className="text-[11px] text-nexa-text-secondary mt-0.5 line-clamp-2">{n.message}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// 3–6. DISCOVER / TRENDING NOW / EMERGING STORIES / UNDERDOG OPPORTUNITIES
export const DiscoveryFeedPage: React.FC<{ mode: 'discover' | 'trending' | 'emerging' | 'underdogs' }> = ({ mode }) => {
  const navigate = useNavigate();
  const [items, setItems] = useState<any[]>([]);
  const [search, setSearch] = useState('');
  const [stageFilter, setStageFilter] = useState('');

  const titles: Record<string, { title: string; subtitle: string }> = {
    discover: {
      title: 'Discover People & Breakthrough Events',
      subtitle: 'Live multi-platform people and event resolution across Wikipedia, Google News, YouTube, Reddit, and Google Trends.',
    },
    trending: {
      title: 'Trending Now — High-Velocity Stories',
      subtitle: 'Stories experiencing rapid views-per-hour acceleration and high channel-relative multipliers.',
    },
    emerging: {
      title: 'Emerging Stories & Early Whispers',
      subtitle: 'Early-stage breakout signals detected before mainstream documentary saturation.',
    },
    underdogs: {
      title: 'Ranked Underdog Documentary Opportunities',
      subtitle: 'Scored across 8 transparent dimensions: Underdog Strength, Transformation, Momentum, Evidence Confidence & Saturation Risk.',
    },
  };

  useEffect(() => {
    apiFetch('/api/discoveries').then((res) => {
      let list = res.discoveries || [];
      if (mode === 'trending') {
        list = list.filter((d: any) => d.trendStage === 'Major Viral Opportunity' || d.trendStage === 'Accelerating');
      } else if (mode === 'emerging') {
        list = list.filter((d: any) => d.trendStage === 'Emerging' || d.trendStage === 'Early Whisper');
      }
      setItems(list);
    });
  }, [mode]);

  const filtered = items.filter((d) => {
    if (stageFilter && d.trendStage !== stageFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return (
        d.personName.toLowerCase().includes(q) ||
        d.headline.toLowerCase().includes(q) ||
        d.category.toLowerCase().includes(q) ||
        d.country.toLowerCase().includes(q)
      );
    }
    return true;
  });

  const handleAddWatchlist = async (e: React.MouseEvent, d: any) => {
    e.stopPropagation();
    await apiFetch('/api/watchlist', {
      method: 'POST',
      body: JSON.stringify({
        discoveryId: d.id,
        personId: d.personId,
        notes: 'Added from ' + mode + ' view',
      }),
    });
    alert(`Added ${d.personName} to Watchlist.`);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black text-display text-nexa-text-primary">{titles[mode].title}</h1>
          <p className="text-xs text-nexa-text-secondary mt-1">{titles[mode].subtitle}</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center bg-nexa-bg-surface px-3.5 py-2 rounded-full border border-nexa-border gap-2 w-64">
            <Search className="w-4 h-4 text-nexa-text-muted shrink-0" />
            <input
              type="text"
              placeholder="Search person, event, country..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="bg-transparent text-xs outline-none w-full text-nexa-text-primary"
            />
          </div>
          <select
            value={stageFilter}
            onChange={(e) => setStageFilter(e.target.value)}
            className="px-4 py-2 rounded-full border border-nexa-border text-xs bg-nexa-bg-surface text-nexa-text-primary font-bold"
          >
            <option value="">All Trend Stages</option>
            <option value="Major Viral Opportunity">Major Viral Opportunity</option>
            <option value="Accelerating">Accelerating</option>
            <option value="Emerging">Emerging</option>
            <option value="Early Whisper">Early Whisper</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {filtered.map((d) => (
          <div
            key={d.id}
            onClick={() => navigate(`/opportunities/${d.id}`)}
            className="nexa-card nexa-card-interactive rounded-3xl p-6 cursor-pointer"
          >
            <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-5">
              <div className="space-y-2.5 flex-1">
                <div className="flex items-center gap-2.5 flex-wrap">
                  <span className="text-lg font-black text-display text-nexa-text-primary">{d.personName}</span>
                  <StageBadge stage={d.trendStage} />
                  <span className="px-2.5 py-0.5 rounded-full text-xs font-extrabold bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20">
                    {d.underdogClassification}
                  </span>
                  <span className="text-xs text-nexa-text-muted font-semibold">
                    {d.country} • {d.category} • {d.eventDate}
                  </span>
                </div>
                <div className="text-sm font-bold text-nexa-text-primary">{d.headline}</div>
                <p className="text-xs text-nexa-text-secondary leading-relaxed">{d.achievement}</p>

                <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-8 gap-2 pt-2">
                  {[
                    { k: 'Underdog', v: d.underdogStrength },
                    { k: 'Transform', v: d.transformationScore },
                    { k: 'Significance', v: d.achievementSignificance },
                    { k: 'Momentum', v: d.audienceMomentum },
                    { k: 'Doc Potential', v: d.documentaryPotential },
                    { k: 'Timeliness', v: d.timelinessScore },
                    { k: 'Evidence', v: d.evidenceConfidence },
                    { k: 'Sat. Risk', v: d.saturationRisk },
                  ].map((sc) => (
                    <div key={sc.k} className="bg-nexa-bg-base rounded-2xl px-3 py-2 border border-nexa-border">
                      <div className="text-[10px] text-nexa-text-muted font-bold uppercase">{sc.k}</div>
                      <div className="text-xs font-black text-display text-nexa-text-primary">{sc.v}</div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex lg:flex-col items-center lg:items-end justify-between gap-3 shrink-0 border-t lg:border-t-0 lg:border-l border-nexa-border pt-4 lg:pt-0 lg:pl-6">
                <div className="text-right">
                  <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Opportunity Score</div>
                  <div className="text-3xl font-black text-display text-nexa-brand">
                    {Number(d.overallScore).toFixed(1)}
                  </div>
                  <div className="text-[11px] text-nexa-text-secondary font-semibold mt-0.5">
                    {Number(d.viewsPerHour).toLocaleString()} VPH ({d.channelRelativePerf}x ch)
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={(e) => handleAddWatchlist(e, d)}
                    className="px-3.5 py-2 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-bold flex items-center gap-1.5 text-nexa-text-secondary hover:text-nexa-brand cursor-pointer"
                  >
                    <Bookmark className="w-3.5 h-3.5" /> Watch
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      navigate(`/opportunities/${d.id}`);
                    }}
                    className="px-4 py-2 rounded-full bg-nexa-brand text-white hover:bg-nexa-brand-mid text-xs font-extrabold shadow-md shadow-nexa-brand/20 cursor-pointer"
                  >
                    Inspect Report →
                  </button>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// 7. OPPORTUNITY DETAIL PAGE
export const OpportunityDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [detail, setDetail] = useState<any>(null);
  const [busy, setBusy] = useState(false);

  const load = () => {
    if (!id) return;
    apiFetch(`/api/discoveries/${id}`).then(setDetail).catch(console.error);
  };

  useEffect(() => {
    load();
  }, [id]);

  if (!detail) return <div className="p-8 text-sm text-nexa-text-muted">Loading full opportunity report...</div>;

  const ua = detail.underdogAssessment || {};
  const rat = detail.scoreRationale || {};
  const research = detail.research || {};

  const handleApproveForDocumentary = async () => {
    setBusy(true);
    try {
      const res = await apiFetch(`/api/discoveries/${detail.id}/approve`, { method: 'POST', body: '{}' });
      navigate(`/production/${res.projectId}`);
    } finally {
      setBusy(false);
    }
  };

  const handleRefreshResearch = async () => {
    setBusy(true);
    try {
      await apiFetch(`/api/research/person/${detail.personId}/run`, { method: 'POST', body: '{}' });
      load();
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="nexa-card rounded-3xl p-6">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-5">
          <div className="space-y-2">
            <div className="flex items-center gap-2.5 flex-wrap">
              <StageBadge stage={detail.trendStage} />
              <span className="px-3 py-0.5 rounded-full text-xs font-extrabold bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20">
                {detail.underdogClassification}
              </span>
              <span className="text-xs text-nexa-text-muted font-semibold">
                {detail.country} • {detail.category} • Source: {detail.publisher}
              </span>
            </div>
            <h1 className="text-2xl font-black text-display text-nexa-text-primary">{detail.personName}</h1>
            <p className="text-sm font-bold text-nexa-text-primary">{detail.headline}</p>
            <p className="text-xs text-nexa-text-secondary leading-relaxed">{detail.achievement}</p>
          </div>

          <div className="flex flex-wrap items-center gap-3 shrink-0">
            <div className="px-4 py-2.5 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/25 text-right">
              <div className="text-[10px] font-extrabold uppercase text-nexa-brand">Opportunity Score</div>
              <div className="text-2xl font-black text-display text-nexa-brand">
                {Number(detail.overallScore).toFixed(1)} / 100
              </div>
            </div>
            <button
              onClick={handleRefreshResearch}
              disabled={busy}
              className="px-4 py-2.5 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold text-nexa-text-primary cursor-pointer"
            >
              Run Deep Fact Verification
            </button>
            <button
              onClick={handleApproveForDocumentary}
              disabled={busy}
              className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold flex items-center gap-2 shadow-lg shadow-nexa-brand/25 cursor-pointer"
            >
              <Film className="w-4 h-4" />
              {detail.projectId ? 'Open Documentary Workspace →' : 'Approve & Generate Documentary →'}
            </button>
          </div>
        </div>
      </div>

      {/* 8-Dimension Transparent Score Breakdown */}
      <div className="nexa-card rounded-3xl p-6">
        <h2 className="text-base font-black text-display text-nexa-text-primary mb-4">
          Transparent 8-Dimension Opportunity Scoring & Evidence Rationale
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            { key: 'underdog_strength', label: 'Underdog Strength', val: detail.underdogStrength },
            { key: 'transformation_score', label: 'Transformation Arc', val: detail.transformationScore },
            { key: 'achievement_significance', label: 'Achievement Significance', val: detail.achievementSignificance },
            { key: 'audience_momentum', label: 'Audience Momentum', val: detail.audienceMomentum },
            { key: 'documentary_potential', label: 'Documentary Potential', val: detail.documentaryPotential },
            { key: 'timeliness_score', label: 'Timeliness', val: detail.timelinessScore },
            { key: 'evidence_confidence', label: 'Evidence Confidence', val: detail.evidenceConfidence },
            { key: 'saturation_risk', label: 'Saturation Risk (Lower = Better)', val: detail.saturationRisk },
          ].map((item) => (
            <div key={item.key} className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border">
              <div className="flex items-center justify-between">
                <span className="text-xs font-extrabold text-nexa-text-primary">{item.label}</span>
                <span className="text-base font-black text-display text-nexa-brand">{item.val}</span>
              </div>
              <p className="text-[11px] text-nexa-text-secondary mt-2 leading-relaxed">
                {rat[item.key] || 'Verified via primary biographical and velocity telemetry.'}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* Underdog Life-Story Investigation */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="nexa-card rounded-3xl p-6 space-y-4">
          <h2 className="text-base font-black text-display text-nexa-text-primary">
            Underdog Narrative Investigation
          </h2>
          {[
            { title: 'Origins & Childhood', body: ua.originsSummary },
            { title: 'Disadvantages & Structural Barriers', body: ua.disadvantages },
            { title: 'Rejections, Setbacks & Crucible Moments', body: ua.rejectionsAndFailures },
            { title: 'Decisive Turning Points', body: ua.turningPoints },
            { title: 'Recent Victory & Breakthrough', body: ua.recentBreakthrough },
          ].map((sec) => (
            <div key={sec.title} className="border-l-4 border-nexa-brand pl-4 py-1">
              <div className="text-xs font-extrabold text-nexa-text-primary">{sec.title}</div>
              <div className="text-xs text-nexa-text-secondary mt-1 leading-relaxed">{sec.body}</div>
            </div>
          ))}
        </div>

        <div className="space-y-6">
          <div className="nexa-card rounded-3xl p-6">
            <h2 className="text-base font-black text-display text-nexa-text-primary mb-3">
              YouTube & Multi-Platform Velocity Snapshots
            </h2>
            <div className="space-y-2.5">
              {(detail.trendHistory || []).map((obs: any, idx: number) => (
                <div
                  key={idx}
                  className="flex items-center justify-between p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border text-xs"
                >
                  <div>
                    <div className="font-bold text-nexa-text-primary">{obs.trendStage}</div>
                    <div className="text-[11px] text-nexa-text-muted">
                      {new Date(obs.observedAt).toLocaleString()}
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-black text-display text-nexa-text-primary">
                      {Number(obs.viewsPerHour).toLocaleString()} views/hr
                    </div>
                    <div className="text-[11px] text-emerald-500 font-bold">
                      Total: {Number(obs.viewsTotal).toLocaleString()} • {obs.channelRelativePerf}x channel avg
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="nexa-card rounded-3xl p-6 space-y-3">
            <h2 className="text-base font-black text-display text-nexa-text-primary">
              Fact Verification, Contradictions & Research Gaps
            </h2>
            <div className="p-3.5 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-600 dark:text-emerald-400">
              <strong>Contradiction Check:</strong> {ua.contradictionsReport}
            </div>
            <div className="p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/20 text-xs text-amber-600 dark:text-amber-400">
              <strong>Research Gaps Report:</strong> {ua.researchGapsReport}
            </div>
            <div className="p-3.5 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/20 text-xs text-nexa-brand">
              <strong>Audience Interest Report:</strong> {ua.audienceInterestReport}
            </div>
          </div>
        </div>
      </div>

      {/* Chronological Timeline & Claims */}
      <div className="nexa-card rounded-3xl p-6">
        <h2 className="text-base font-black text-display text-nexa-text-primary mb-4">
          Chronological Life Timeline & Verified Claims
        </h2>
        <div className="space-y-3">
          {(research.timeline || []).map((t: any) => (
            <div
              key={t.id}
              className="p-4 rounded-2xl bg-nexa-bg-base border border-nexa-border flex flex-col md:flex-row md:items-center justify-between gap-3"
            >
              <div>
                <div className="flex items-center gap-2">
                  <span className="px-2.5 py-0.5 rounded-full bg-nexa-brand text-white text-[10px] font-extrabold">
                    {t.yearOrDate}
                  </span>
                  <span className="text-xs font-extrabold text-nexa-text-primary">{t.title}</span>
                  <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[10px] font-extrabold">
                    {t.verificationStatus}
                  </span>
                </div>
                <p className="text-xs text-nexa-text-secondary mt-1.5 leading-relaxed">{t.description}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
