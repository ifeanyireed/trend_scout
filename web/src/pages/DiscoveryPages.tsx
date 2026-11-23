import React, { useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import {
  Archive, ArchiveRestore, ArrowRight, Bookmark, Compass, Copy, ExternalLink, Eye, EyeOff, Film, Flame, Hash, Lock,
  MessageSquare, PlayCircle, RefreshCw, Save, Search, ShieldCheck, Sparkles, Tag, ThumbsUp, Trophy, TrendingUp, Activity, Bell, CheckCircle2, AlertCircle, Clock, Users
} from 'lucide-react';
import { apiFetch, hasActiveSession, setAuthSession } from '../api/client';
import { PaginationController } from '../components/PaginationController';

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

// 1. LOGIN PAGE (JWT Auth + Session Expiry Handling)
export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(true);
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const searchParams = new URLSearchParams(location.search);
  const sessionExpired = searchParams.get('expired') === '1' || Boolean((location.state as any)?.expired);
  const redirectTo = (location.state as any)?.from || '/';

  useEffect(() => {
    if (hasActiveSession() && !sessionExpired) {
      navigate(redirectTo, { replace: true });
    }
  }, [navigate, redirectTo, sessionExpired]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!email.trim() || !password) {
      setError('Please enter both your producer email and password.');
      return;
    }
    setLoading(true);
    try {
      const res = await apiFetch('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email: email.trim(), password, rememberMe }),
      });
      setAuthSession(res.token, res.refreshToken, res.user, res.expiresAt, res.refreshExpiresAt);
      navigate(redirectTo, { replace: true });
    } catch (err: any) {
      setError(err.message || 'Authentication failed. Check your credentials.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-nexa-bg-base flex items-center justify-center p-6">
      <div className="liquid-glass rounded-3xl border border-nexa-border max-w-md w-full p-8 shadow-xl">
        <div className="flex items-center gap-3.5 mb-6">
          <img
            src="/logo.png"
            alt="TrendScout Studio"
            className="w-12 h-12 rounded-2xl object-contain shrink-0"
          />
          <div>
            <h1 className="text-xl font-black text-display text-nexa-text-primary">TrendScout Studio</h1>
            <p className="text-xs text-nexa-text-secondary font-medium">
              AI Documentary Discovery &amp; Production OS
            </p>
          </div>
        </div>

        {sessionExpired && !error && (
          <div className="mb-4 p-3.5 rounded-2xl bg-amber-500/10 border border-amber-500/25 text-amber-600 dark:text-amber-400 text-xs font-bold flex items-center gap-2.5">
            <Clock className="w-4 h-4 shrink-0" />
            <span>Your studio session expired for security. Please sign in again.</span>
          </div>
        )}

        {error && (
          <div className="mb-4 p-3.5 rounded-2xl bg-rose-500/10 border border-rose-500/25 text-rose-500 text-xs font-bold flex items-center gap-2.5">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary uppercase tracking-wider mb-1.5">
              Email Address
            </label>
            <input
              type="email"
              required
              autoComplete="email"
              placeholder="Enter your email address"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-3 border border-nexa-border bg-nexa-bg-surface rounded-2xl text-sm text-nexa-text-primary outline-none focus:border-nexa-brand transition"
            />
          </div>
          <div>
            <label className="block text-xs font-bold text-nexa-text-secondary uppercase tracking-wider mb-1.5">
              Password
            </label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                required
                autoComplete="current-password"
                placeholder="••••••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 pr-11 border border-nexa-border bg-nexa-bg-surface rounded-2xl text-sm text-nexa-text-primary outline-none focus:border-nexa-brand transition"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3.5 top-1/2 -translate-y-1/2 text-nexa-text-muted hover:text-nexa-text-primary cursor-pointer"
                title={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <label className="flex items-center justify-between gap-2 text-xs text-nexa-text-secondary cursor-pointer select-none pt-1">
            <span className="flex items-center gap-2 font-semibold">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                className="rounded border-nexa-border accent-nexa-brand cursor-pointer"
              />
              Keep me signed in
            </span>
          </label>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3.5 rounded-2xl bg-nexa-brand hover:bg-nexa-brand-mid disabled:opacity-60 text-white font-extrabold text-sm shadow-lg shadow-nexa-brand/25 transition cursor-pointer flex items-center justify-center gap-2"
          >
            <Lock className="w-4 h-4" />
            <span>{loading ? 'Signing in...' : 'Sign In to TrendScout Studio →'}</span>
          </button>
        </form>

        <div className="mt-6 pt-4 border-t border-nexa-border flex items-center justify-center text-[11px] text-nexa-text-muted font-semibold">
          <span className="flex items-center gap-1.5">
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-500" />
            Secure Studio Access
          </span>
        </div>
      </div>
    </div>
  );
};

// 2. DASHBOARD OVERVIEW PAGE (ofia_ng AdminStatGrid + NexaCard)
export const DashboardPage: React.FC = () => {
  const navigate = useNavigate();
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [dashPage, setDashPage] = useState(1);
  const [dashPageSize, setDashPageSize] = useState(6);

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

  const handleArchiveDashboardItem = async (e: React.MouseEvent, discoveryId: string) => {
    e.stopPropagation();
    try {
      await apiFetch(`/api/discoveries/${discoveryId}/archive`, { method: 'POST', body: '{}' });
      load();
    } catch (err) {
      console.error(err);
    }
  };

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

  const topDiscoveries: any[] = Array.isArray(data.topDiscoveries) ? data.topDiscoveries : [];
  const dashTotalPages = Math.max(1, Math.ceil(topDiscoveries.length / dashPageSize));
  const safeDashPage = Math.min(dashPage, dashTotalPages);
  const pagedTopDiscoveries = topDiscoveries.slice(
    (safeDashPage - 1) * dashPageSize,
    safeDashPage * dashPageSize,
  );

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
      sub: '20-min scripts & ≤7s Google Flow beats',
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
            {topDiscoveries.length === 0 ? (
              <div className="p-8 rounded-2xl border border-dashed border-nexa-border bg-nexa-bg-base/40 text-center space-y-3">
                <div className="w-11 h-11 rounded-2xl bg-nexa-brand/10 text-nexa-brand flex items-center justify-center mx-auto">
                  <Compass className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-sm font-black text-display text-nexa-text-primary">No Live Discoveries Yet</h3>
                  <p className="text-xs text-nexa-text-secondary mt-1">
                    Run a live multi-domain discovery scan across Google News, Google Trends, Wikipedia, Reddit, and YouTube.
                  </p>
                </div>
                <button
                  onClick={() => window.dispatchEvent(new CustomEvent('ts-trigger-scan'))}
                  className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold shadow-md shadow-nexa-brand/25 cursor-pointer"
                >
                  Run Live Discovery Scan Now →
                </button>
              </div>
            ) : (
              pagedTopDiscoveries.map((item: any) => (
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
                  <div className="flex items-center gap-4 text-[11px] text-nexa-text-muted font-semibold pt-1 flex-wrap">
                    <span>🔥 {Number(item.viewsPerHour).toLocaleString()} views/hr</span>
                    <span>📈 {item.channelRelativePerf}x channel baseline</span>
                    <span>📰 {item.independentCoverageCount} independent sources</span>
                    <span>📡 {item.publisher}</span>
                  </div>
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <button
                    onClick={(e) => handleArchiveDashboardItem(e, item.id)}
                    title="Archive discovery"
                    className="p-2 rounded-full border border-nexa-border hover:border-amber-500/60 text-nexa-text-muted hover:text-amber-500 transition cursor-pointer"
                  >
                    <Archive className="w-3.5 h-3.5" />
                  </button>
                  <div className="text-right px-3.5 py-2 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/20">
                    <div className="text-[9px] font-extrabold uppercase text-nexa-brand">Score</div>
                    <div className="text-xl font-black text-display text-nexa-brand">
                      {Number(item.overallScore).toFixed(1)}
                    </div>
                  </div>
                  <ArrowRight className="w-4 h-4 text-nexa-text-faint" />
                </div>
              </div>
            ))
            )}
          </div>

          {topDiscoveries.length > 0 && (
            <div className="mt-4">
              <PaginationController
                currentPage={safeDashPage}
                totalItems={topDiscoveries.length}
                pageSize={dashPageSize}
                pageSizeOptions={[6, 10, 20]}
                onPageChange={(p) => setDashPage(p)}
                onPageSizeChange={(sz) => {
                  setDashPageSize(sz);
                  setDashPage(1);
                }}
              />
            </div>
          )}
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [stageFilter, setStageFilter] = useState('');
  const [statusTab, setStatusTab] = useState<'active' | 'archived'>('active');
  const [activeCount, setActiveCount] = useState(0);
  const [archivedCount, setArchivedCount] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const titles: Record<string, { title: string; subtitle: string }> = {
    discover: {
      title: 'Discover People & Breakthrough Events',
      subtitle: 'Live multi-domain discovery across Science, Tech, Business, Arts, Humanitarian, and Sports via Google News, Google Trends, Wikipedia, Reddit & YouTube.',
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

  const load = () => {
    setError(null);
    apiFetch(`/api/discoveries?status=${statusTab}`)
      .then((res) => {
        let list = Array.isArray(res?.discoveries) ? res.discoveries : [];
        setActiveCount(Number(res?.activeCount ?? 0));
        setArchivedCount(Number(res?.archivedCount ?? 0));
        if (mode === 'trending') {
          list = list.filter((d: any) => d.trendStage === 'Major Viral Opportunity' || d.trendStage === 'Accelerating');
        } else if (mode === 'emerging') {
          list = list.filter((d: any) => d.trendStage === 'Emerging' || d.trendStage === 'Early Whisper');
        }
        setItems(list);
      })
      .catch((err) => {
        console.error(err);
        setError(err?.message || 'Unable to load discovery feed');
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    setLoading(true);
    setCurrentPage(1);
    load();
    window.addEventListener('ts-refresh', load);
    return () => window.removeEventListener('ts-refresh', load);
  }, [mode, statusTab]);

  useEffect(() => {
    setCurrentPage(1);
  }, [search, stageFilter, pageSize]);

  const filtered = items.filter((d) => {
    if (stageFilter && d.trendStage !== stageFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return (
        (d.personName || '').toLowerCase().includes(q) ||
        (d.headline || '').toLowerCase().includes(q) ||
        (d.category || '').toLowerCase().includes(q) ||
        (d.country || '').toLowerCase().includes(q)
      );
    }
    return true;
  });

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const safePage = Math.min(currentPage, totalPages);
  const paginatedItems = filtered.slice((safePage - 1) * pageSize, safePage * pageSize);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleToggleArchive = async (e: React.MouseEvent, d: any) => {
    e.stopPropagation();
    const action = d.status === 'archived' ? 'restore' : 'archive';
    try {
      await apiFetch(`/api/discoveries/${d.id}/${action}`, { method: 'POST', body: '{}' });
      load();
    } catch (err) {
      console.error(err);
    }
  };

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
        <div className="flex items-center gap-2.5 flex-wrap">
          <div className="inline-flex items-center p-1 rounded-full bg-nexa-bg-surface border border-nexa-border">
            <button
              type="button"
              onClick={() => setStatusTab('active')}
              className={`px-3.5 py-1.5 rounded-full text-xs font-extrabold transition cursor-pointer ${
                statusTab === 'active'
                  ? 'bg-nexa-brand text-white shadow-sm'
                  : 'text-nexa-text-secondary hover:text-nexa-text-primary'
              }`}
            >
              Active ({activeCount})
            </button>
            <button
              type="button"
              onClick={() => setStatusTab('archived')}
              className={`px-3.5 py-1.5 rounded-full text-xs font-extrabold transition cursor-pointer flex items-center gap-1.5 ${
                statusTab === 'archived'
                  ? 'bg-nexa-brand text-white shadow-sm'
                  : 'text-nexa-text-secondary hover:text-nexa-text-primary'
              }`}
            >
              <Archive className="w-3.5 h-3.5" />
              Archived ({archivedCount})
            </button>
          </div>
          <div className="flex items-center bg-nexa-bg-surface px-3.5 py-2 rounded-full border border-nexa-border gap-2 w-60">
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

      {loading && items.length === 0 ? (
        <div className="p-14 flex flex-col items-center justify-center space-y-3">
          <div className="w-8 h-8 rounded-full border-2 border-nexa-brand border-t-transparent animate-spin" />
          <div className="text-xs text-nexa-text-muted font-bold">Loading live opportunities...</div>
        </div>
      ) : error && items.length === 0 ? (
        <div className="nexa-card rounded-3xl p-8 text-center space-y-3">
          <AlertCircle className="w-6 h-6 text-amber-500 mx-auto" />
          <div className="text-sm font-black text-display text-nexa-text-primary">Unable to load feed</div>
          <p className="text-xs text-nexa-text-secondary">{error}</p>
          <button
            onClick={load}
            className="px-5 py-2 rounded-full bg-nexa-brand text-white text-xs font-extrabold cursor-pointer"
          >
            Retry
          </button>
        </div>
      ) : filtered.length === 0 ? (
        <div className="nexa-card rounded-3xl p-10 text-center space-y-3">
          <Compass className="w-7 h-7 text-nexa-brand mx-auto" />
          <div className="text-base font-black text-display text-nexa-text-primary">
            No Matching Opportunities Found
          </div>
          <p className="text-xs text-nexa-text-secondary max-w-md mx-auto">
            {search || stageFilter
              ? 'Try clearing your search or trend stage filter.'
              : 'Trigger a live scan to discover fresh multi-domain stories across Science, Business, Arts, Tech, Humanitarian, and Sports.'}
          </p>
          <button
            onClick={() => window.dispatchEvent(new CustomEvent('ts-trigger-scan'))}
            className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold shadow-md shadow-nexa-brand/25 cursor-pointer"
          >
            Run Live Discovery Scan Now →
          </button>
        </div>
      ) : (
      <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4">
        {paginatedItems.map((d) => (
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
                <div className="flex gap-2 flex-wrap justify-end">
                  <button
                    onClick={(e) => handleToggleArchive(e, d)}
                    className="px-3.5 py-2 rounded-full border border-nexa-border hover:border-amber-500 text-xs font-bold flex items-center gap-1.5 text-nexa-text-secondary hover:text-amber-500 cursor-pointer"
                  >
                    {d.status === 'archived' ? (
                      <>
                        <ArchiveRestore className="w-3.5 h-3.5" /> Restore
                      </>
                    ) : (
                      <>
                        <Archive className="w-3.5 h-3.5" /> Archive
                      </>
                    )}
                  </button>
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
      <PaginationController
        currentPage={safePage}
        totalItems={filtered.length}
        pageSize={pageSize}
        pageSizeOptions={[10, 20, 50]}
        onPageChange={handlePageChange}
        onPageSizeChange={(sz) => {
          setPageSize(sz);
          setCurrentPage(1);
        }}
      />
      </div>
      )}
    </div>
  );
};

// 7. OPPORTUNITY DETAIL PAGE
export const OpportunityDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [detail, setDetail] = useState<any>(null);
  const [busy, setBusy] = useState(false);
  const [seoBusy, setSeoBusy] = useState(false);
  const [ytBusy, setYtBusy] = useState(false);
  const [reprocessing, setReprocessing] = useState(false);
  const [aiActionError, setAiActionError] = useState('');
  const [copiedKey, setCopiedKey] = useState('');
  const [videoDescription, setVideoDescription] = useState('');
  const [tagsInput, setTagsInput] = useState('');
  const [hashtagsInput, setHashtagsInput] = useState('');
  const [seoSaved, setSeoSaved] = useState(false);

  const syncSeoState = (d: any) => {
    setVideoDescription(d?.videoDescription || '');
    setTagsInput(Array.isArray(d?.videoTags) ? d.videoTags.join(', ') : '');
    setHashtagsInput(Array.isArray(d?.videoHashtags) ? d.videoHashtags.join(' ') : '');
  };

  const load = () => {
    if (!id) return;
    apiFetch(`/api/discoveries/${id}`)
      .then((res) => {
        setDetail(res);
        syncSeoState(res);
      })
      .catch(console.error);
  };

  useEffect(() => {
    load();
  }, [id]);

  if (!detail) return <div className="p-8 text-sm text-nexa-text-muted">Loading full opportunity report...</div>;

  const ua = detail.underdogAssessment || {};
  const rat = detail.scoreRationale || {};
  const research = detail.research || {};

  const parsedTags = tagsInput
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);
  const parsedHashtags = hashtagsInput
    .split(/[\s,]+/)
    .map((h) => h.trim())
    .filter(Boolean)
    .map((h) => (h.startsWith('#') ? h : `#${h}`));
  const tagsCharCount = parsedTags.join(', ').length;

  const copyText = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(''), 1800);
  };

  const handleApproveForDocumentary = async () => {
    setBusy(true);
    setAiActionError('');
    try {
      const res = await apiFetch(`/api/discoveries/${detail.id}/approve`, { method: 'POST', body: '{}' });
      navigate(`/production/${res.projectId}`);
    } catch (err: any) {
      setAiActionError(err?.message || 'AI documentary generation failed');
    } finally {
      setBusy(false);
    }
  };

  const handleRerunScriptProcessing = async () => {
    setReprocessing(true);
    setAiActionError('');
    try {
      const res = await apiFetch(`/api/discoveries/${detail.id}/reprocess-script`, {
        method: 'POST',
        body: JSON.stringify({
          targetDurationMinutes: 20,
          style: 'Investigative Archival Documentary',
          biographicalField: 'auto',
        }),
      });
      if (res?.projectId) {
        navigate(`/production/${res.projectId}?tab=script`);
      } else {
        load();
      }
    } catch (err: any) {
      setAiActionError(err?.message || 'AI script reprocessing failed');
    } finally {
      setReprocessing(false);
    }
  };

  const handleGenerateSEO = async () => {
    setSeoBusy(true);
    setAiActionError('');
    try {
      const res = await apiFetch(`/api/discoveries/${detail.id}/seo/generate`, {
        method: 'POST',
        body: '{}',
      });
      setVideoDescription(res.videoDescription || '');
      setTagsInput(Array.isArray(res.videoTags) ? res.videoTags.join(', ') : '');
      setHashtagsInput(Array.isArray(res.videoHashtags) ? res.videoHashtags.join(' ') : '');
      load();
    } catch (err: any) {
      setAiActionError(err?.message || 'AI YouTube SEO packaging failed');
    } finally {
      setSeoBusy(false);
    }
  };

  const handleSaveSEO = async () => {
    setSeoBusy(true);
    try {
      const res = await apiFetch(`/api/discoveries/${detail.id}/seo`, {
        method: 'PUT',
        body: JSON.stringify({
          videoDescription,
          videoTags: parsedTags,
          videoHashtags: parsedHashtags,
        }),
      });
      setVideoDescription(res.videoDescription || '');
      setTagsInput(Array.isArray(res.videoTags) ? res.videoTags.join(', ') : '');
      setHashtagsInput(Array.isArray(res.videoHashtags) ? res.videoHashtags.join(' ') : '');
      setSeoSaved(true);
      setTimeout(() => setSeoSaved(false), 2000);
    } finally {
      setSeoBusy(false);
    }
  };

  const handleRefreshYouTubeIntel = async () => {
    setYtBusy(true);
    try {
      const updated = await apiFetch(`/api/discoveries/${detail.id}/youtube/refresh`, {
        method: 'POST',
        body: '{}',
      });
      if (updated && updated.id) {
        setDetail(updated);
        syncSeoState(updated);
      } else {
        load();
      }
    } finally {
      setYtBusy(false);
    }
  };

  const formatVideoDuration = (sec: number) => {
    if (!sec || sec <= 0) return '—';
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
    return `${m}:${String(s).padStart(2, '0')}`;
  };

  const ytIntel = detail.youtubeIntelligence || {
    videoCount: 0,
    totalViews: 0,
    totalLikes: 0,
    totalComments: 0,
    peakViewsPerHour: detail.viewsPerHour || 0,
    peakViewsPerDay: (detail.viewsPerHour || 0) * 24,
    topOutlierMultiplier: detail.channelRelativePerf || 1,
    avgEngagementRate: 0,
    longFormCount: 0,
    midFormCount: 0,
    shortFormCount: 0,
    competing50kCount: 0,
    docGapLabel: 'Unverified — Click Refresh YouTube Metrics',
    videos: [],
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

  const handleToggleArchiveDetail = async () => {
    setBusy(true);
    try {
      const action = detail.status === 'archived' ? 'restore' : 'archive';
      await apiFetch(`/api/discoveries/${detail.id}/${action}`, { method: 'POST', body: '{}' });
      load();
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      {aiActionError && (
        <div className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-xs text-rose-500 flex items-center justify-between gap-3">
          <span><strong>AI Required Error:</strong> {aiActionError}</span>
          <button
            type="button"
            onClick={() => navigate('/settings')}
            className="px-3.5 py-1.5 rounded-full bg-rose-500 text-white text-[11px] font-extrabold shrink-0 cursor-pointer"
          >
            Open AI Settings →
          </button>
        </div>
      )}
      <div className="nexa-card rounded-3xl p-6">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-5">
          <div className="space-y-2">
            <div className="flex items-center gap-2.5 flex-wrap">
              <StageBadge stage={detail.trendStage} />
              {detail.status === 'archived' && (
                <span className="px-3 py-0.5 rounded-full text-xs font-extrabold bg-amber-500/10 text-amber-500 border border-amber-500/25">
                  Archived
                </span>
              )}
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

          <div className="flex flex-col sm:flex-row lg:items-start gap-3 shrink-0 w-full lg:w-auto">
            <div className="px-4 py-2.5 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/25 text-center sm:text-right shrink-0">
              <div className="text-[10px] font-extrabold uppercase text-nexa-brand">Opportunity Score</div>
              <div className="text-2xl font-black text-display text-nexa-brand">
                {Number(detail.overallScore).toFixed(1)} / 100
              </div>
            </div>
            <div className="flex flex-col items-start sm:items-end gap-2">
              <button
                onClick={handleApproveForDocumentary}
                disabled={busy || reprocessing}
                className="w-fit px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold flex items-center gap-2 shadow-lg shadow-nexa-brand/25 cursor-pointer whitespace-nowrap"
              >
                <Film className="w-4 h-4" />
                {detail.projectId ? 'Open Documentary Workspace →' : 'Approve & Generate Documentary →'}
              </button>
              {detail.projectId && (
                <button
                  onClick={handleRerunScriptProcessing}
                  disabled={busy || reprocessing}
                  className="w-fit px-4 py-2.5 rounded-full border border-emerald-500/40 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-500 text-xs font-extrabold flex items-center gap-1.5 cursor-pointer whitespace-nowrap"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${reprocessing ? 'animate-spin' : ''}`} />
                  {reprocessing ? 'Reprocessing Script...' : 'Rerun Script Processing'}
                </button>
              )}
              <button
                onClick={handleRefreshResearch}
                disabled={busy || reprocessing}
                className="w-fit px-4 py-2.5 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold text-nexa-text-primary cursor-pointer whitespace-nowrap"
              >
                Run Deep Fact Verification
              </button>
              <button
                onClick={handleToggleArchiveDetail}
                disabled={busy}
                className="w-fit px-4 py-2.5 rounded-full border border-nexa-border hover:border-amber-500 text-xs font-extrabold text-nexa-text-secondary hover:text-amber-500 flex items-center gap-1.5 cursor-pointer whitespace-nowrap"
              >
                {detail.status === 'archived' ? (
                  <>
                    <ArchiveRestore className="w-3.5 h-3.5" /> Restore
                  </>
                ) : (
                  <>
                    <Archive className="w-3.5 h-3.5" /> Archive
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* YouTube Market Intelligence, Format Gap & Breakout Videos */}
      <div className="nexa-card rounded-3xl p-6 space-y-5">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-black text-display text-nexa-text-primary flex items-center gap-2">
              <PlayCircle className="w-4 h-4 text-rose-500" />
              YouTube Intelligence &amp; Documentary Market Gap
            </h2>
            <p className="text-xs text-nexa-text-secondary mt-0.5">
              Live YouTube Data API v3 velocity, true channel-baseline outlier scores, engagement ratios, and long-form vs. shorts saturation.
            </p>
          </div>
          <button
            type="button"
            onClick={handleRefreshYouTubeIntel}
            disabled={ytBusy}
            className="px-4 py-2 rounded-full bg-rose-600 hover:bg-rose-500 text-white text-xs font-extrabold flex items-center gap-1.5 shadow-md shadow-rose-600/20 cursor-pointer shrink-0"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${ytBusy ? 'animate-spin' : ''}`} />
            {ytBusy ? 'Pulling Live YouTube Stats...' : 'Refresh Live YouTube Metrics'}
          </button>
        </div>

        {/* Documentary Format Gap Banner */}
        <div
          className={`p-3.5 rounded-2xl border flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs ${
            ytIntel.longFormCount === 0 && ytIntel.videoCount > 0
              ? 'bg-emerald-500/10 border-emerald-500/25 text-emerald-600 dark:text-emerald-400'
              : ytIntel.longFormCount <= 1 && ytIntel.videoCount > 0
                ? 'bg-nexa-brand/10 border-nexa-brand/25 text-nexa-brand'
                : 'bg-nexa-bg-base border-nexa-border text-nexa-text-secondary'
          }`}
        >
          <div className="font-extrabold">
            Documentary Supply Verdict: <span className="font-black">{ytIntel.docGapLabel}</span>
          </div>
          <div className="flex items-center gap-3 text-[11px] font-bold flex-wrap">
            <span>Long-Form (8m+): {ytIntel.longFormCount}</span>
            <span>Mid-Form (1–8m): {ytIntel.midFormCount}</span>
            <span>Shorts (&lt;60s): {ytIntel.shortFormCount}</span>
          </div>
        </div>

        {/* 6 YouTube KPI Cards */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
          <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border">
            <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Peak Velocity</div>
            <div className="text-lg font-black text-display text-nexa-text-primary mt-1">
              {Number(ytIntel.peakViewsPerHour || 0).toLocaleString()}
              <span className="text-[10px] font-bold text-nexa-text-muted ml-1">VPH</span>
            </div>
            <div className="text-[10px] text-nexa-text-secondary font-semibold mt-0.5">
              {Number(ytIntel.peakViewsPerDay || 0).toLocaleString()} views/day
            </div>
          </div>

          <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border">
            <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Top Outlier Score</div>
            <div className="text-lg font-black text-display text-emerald-500 mt-1">
              {Number(ytIntel.topOutlierMultiplier || 1).toFixed(1)}x
            </div>
            <div className="text-[10px] text-nexa-text-secondary font-semibold mt-0.5">
              vs. channel avg views
            </div>
          </div>

          <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border">
            <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Avg Engagement</div>
            <div className="text-lg font-black text-display text-nexa-brand mt-1">
              {Number(ytIntel.avgEngagementRate || 0).toFixed(2)}%
            </div>
            <div className="text-[10px] text-nexa-text-secondary font-semibold mt-0.5">
              (Likes + Comments) / Views
            </div>
          </div>

          <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border">
            <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Tracked Reach</div>
            <div className="text-lg font-black text-display text-nexa-text-primary mt-1">
              {Number(ytIntel.totalViews || 0).toLocaleString()}
            </div>
            <div className="text-[10px] text-nexa-text-secondary font-semibold mt-0.5">
              Across {ytIntel.videoCount} video{ytIntel.videoCount === 1 ? '' : 's'}
            </div>
          </div>

          <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border">
            <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">Audience Interaction</div>
            <div className="text-sm font-black text-display text-nexa-text-primary mt-1.5 flex items-center gap-1.5">
              <ThumbsUp className="w-3.5 h-3.5 text-nexa-brand shrink-0" />
              {Number(ytIntel.totalLikes || 0).toLocaleString()}
            </div>
            <div className="text-[10px] text-nexa-text-secondary font-semibold mt-0.5 flex items-center gap-1">
              <MessageSquare className="w-3 h-3 shrink-0" />
              {Number(ytIntel.totalComments || 0).toLocaleString()} comments
            </div>
          </div>

          <div className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border">
            <div className="text-[10px] font-extrabold uppercase text-nexa-text-muted">50k+ Competitors</div>
            <div className="text-lg font-black text-display text-amber-500 mt-1">
              {ytIntel.competing50kCount}
            </div>
            <div className="text-[10px] text-nexa-text-secondary font-semibold mt-0.5">
              Sat. Risk: {detail.saturationRisk}%
            </div>
          </div>
        </div>

        {/* Tracked YouTube Breakout & Competing Videos Table/Cards */}
        {(ytIntel.videos || []).length > 0 ? (
          <div className="space-y-3">
            <div className="text-xs font-extrabold uppercase text-nexa-text-muted">
              Top Competing &amp; Breakout YouTube Videos ({ytIntel.videos.length})
            </div>
            <div className="space-y-2.5">
              {(ytIntel.videos || []).map((v: any) => (
                <div
                  key={v.id}
                  className="p-3.5 rounded-2xl bg-nexa-bg-base border border-nexa-border hover:border-nexa-brand/40 transition flex flex-col md:flex-row gap-4 items-start md:items-center"
                >
                  {v.thumbnailUrl && (
                    <a
                      href={v.url}
                      target="_blank"
                      rel="noreferrer"
                      className="relative w-full md:w-36 aspect-video rounded-xl overflow-hidden border border-nexa-border shrink-0 bg-black"
                    >
                      <img src={v.thumbnailUrl} alt={v.title} className="w-full h-full object-cover" />
                      {v.durationSeconds > 0 && (
                        <span className="absolute bottom-1.5 right-1.5 px-1.5 py-0.5 rounded bg-black/80 text-white text-[10px] font-extrabold font-mono">
                          {formatVideoDuration(v.durationSeconds)}
                        </span>
                      )}
                    </a>
                  )}
                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] font-extrabold border ${
                          String(v.videoFormat || '').includes('Long-Form')
                            ? 'bg-purple-500/10 text-purple-500 border-purple-500/25'
                            : String(v.videoFormat || '').includes('Short')
                              ? 'bg-rose-500/10 text-rose-500 border-rose-500/25'
                              : 'bg-nexa-bg-surface text-nexa-text-secondary border-nexa-border'
                        }`}
                      >
                        {v.videoFormat || 'Standard'}
                      </span>
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-extrabold bg-emerald-500/10 text-emerald-500 border border-emerald-500/25">
                        {Number(v.channelRelativeMultiplier || 1).toFixed(1)}x vs. channel avg
                      </span>
                      <span className="text-[11px] text-nexa-text-muted font-semibold">
                        Published {v.publishedAt}
                      </span>
                    </div>
                    <a
                      href={v.url}
                      target="_blank"
                      rel="noreferrer"
                      className="text-xs font-extrabold text-nexa-text-primary hover:text-nexa-brand inline-flex items-center gap-1.5"
                    >
                      <span className="line-clamp-1">{v.title}</span>
                      <ExternalLink className="w-3 h-3 shrink-0" />
                    </a>
                    <div className="flex items-center gap-3 flex-wrap text-[11px] text-nexa-text-secondary font-semibold">
                      <span className="inline-flex items-center gap-1">
                        <Users className="w-3 h-3 text-nexa-text-muted" />
                        {v.channelName}
                        {v.channelSubscribers > 0 && ` (${Number(v.channelSubscribers).toLocaleString()} subs)`}
                      </span>
                      {v.channelBaselineViews > 0 && (
                        <span className="text-nexa-text-muted">
                          • Channel avg: {Number(v.channelBaselineViews).toLocaleString()} views/vid
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="grid grid-cols-3 md:grid-cols-1 gap-2 md:text-right shrink-0 w-full md:w-auto border-t md:border-t-0 md:border-l border-nexa-border pt-2.5 md:pt-0 md:pl-4">
                    <div>
                      <div className="text-xs font-black text-display text-nexa-text-primary">
                        {Number(v.views || 0).toLocaleString()} views
                      </div>
                      <div className="text-[10px] font-bold text-nexa-brand">
                        {Number(v.viewsPerHour || 0).toLocaleString()} VPH
                      </div>
                    </div>
                    <div className="text-[10px] text-nexa-text-secondary font-semibold">
                      {Number(v.likes || 0).toLocaleString()} likes • {Number(v.comments || 0).toLocaleString()} comments
                    </div>
                    <div className="text-[10px] font-extrabold text-emerald-500">
                      {v.engagementRate > 0 ? `${Number(v.engagementRate).toFixed(2)}% engagement` : 'Engagement N/A'}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="p-6 rounded-2xl border border-dashed border-nexa-border bg-nexa-bg-base/40 text-center space-y-2">
            <div className="text-xs font-bold text-nexa-text-primary">No live YouTube videos analyzed for this opportunity yet.</div>
            <p className="text-[11px] text-nexa-text-secondary">
              Click <strong>Refresh Live YouTube Metrics</strong> above to pull top competing videos, channel baselines, durations, engagement rates, and the long-form documentary gap.
            </p>
          </div>
        )}
      </div>

      {/* Video Description, High-Performing Tags & Hashtags */}
      <div className="nexa-card rounded-3xl p-6 space-y-5">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-black text-display text-nexa-text-primary flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-nexa-brand" />
              Video Description, High-Performing Tags &amp; Hashtags
            </h2>
            <p className="text-xs text-nexa-text-secondary mt-0.5">
              Search-optimized YouTube documentary description with chapter timestamps, high-intent SEO tags (≤500 chars), and hashtags.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={handleGenerateSEO}
              disabled={seoBusy}
              className="px-4 py-2 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white text-xs font-extrabold flex items-center gap-1.5 shadow-md shadow-nexa-brand/20 cursor-pointer"
            >
              <Sparkles className="w-3.5 h-3.5" />
              {seoBusy ? 'Generating SEO...' : videoDescription ? 'Regenerate Description, Tags & Hashtags' : 'Generate Description, Tags & Hashtags'}
            </button>
            <button
              type="button"
              onClick={handleSaveSEO}
              disabled={seoBusy}
              className="px-4 py-2 rounded-full border border-nexa-border hover:border-nexa-brand text-xs font-extrabold text-nexa-text-primary flex items-center gap-1.5 cursor-pointer"
            >
              <Save className="w-3.5 h-3.5" />
              {seoSaved ? 'Saved!' : 'Save Changes'}
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-5">
          <div className="lg:col-span-7 space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-xs font-extrabold uppercase text-nexa-text-muted">
                YouTube Video Description ({videoDescription.length} chars)
              </label>
              <button
                type="button"
                onClick={() => copyText(videoDescription, 'OPP_DESC')}
                disabled={!videoDescription}
                className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 disabled:opacity-40 cursor-pointer"
              >
                <Copy className="w-3 h-3" /> {copiedKey === 'OPP_DESC' ? 'Copied Description!' : 'Copy Description'}
              </button>
            </div>
            <textarea
              value={videoDescription}
              onChange={(e) => setVideoDescription(e.target.value)}
              rows={11}
              placeholder="Click 'Generate Description, Tags & Hashtags' to write a YouTube-ready documentary description with chapter timestamps..."
              className="w-full p-3.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs leading-relaxed text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
            />
          </div>

          <div className="lg:col-span-5 space-y-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-2">
                <label className="text-xs font-extrabold uppercase text-nexa-text-muted flex items-center gap-1.5">
                  <Tag className="w-3.5 h-3.5 text-nexa-brand" />
                  High-Performing Tags ({parsedTags.length})
                </label>
                <div className="flex items-center gap-2">
                  <span
                    className={`text-[10px] font-extrabold px-2 py-0.5 rounded-full border ${
                      tagsCharCount > 500
                        ? 'bg-rose-500/10 text-rose-500 border-rose-500/25'
                        : 'bg-emerald-500/10 text-emerald-500 border-emerald-500/25'
                    }`}
                  >
                    {tagsCharCount} / 500 chars
                  </span>
                  <button
                    type="button"
                    onClick={() => copyText(parsedTags.join(', '), 'OPP_TAGS')}
                    disabled={parsedTags.length === 0}
                    className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 disabled:opacity-40 cursor-pointer"
                  >
                    <Copy className="w-3 h-3" /> {copiedKey === 'OPP_TAGS' ? 'Copied!' : 'Copy Tags'}
                  </button>
                </div>
              </div>
              <textarea
                value={tagsInput}
                onChange={(e) => setTagsInput(e.target.value)}
                rows={4}
                placeholder="Comma-separated high-performing YouTube tags..."
                className="w-full p-3 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
              />
              {parsedTags.length > 0 && (
                <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto pt-1">
                  {parsedTags.map((t, idx) => (
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
                <label className="text-xs font-extrabold uppercase text-nexa-text-muted flex items-center gap-1.5">
                  <Hash className="w-3.5 h-3.5 text-nexa-brand" />
                  Hashtags ({parsedHashtags.length})
                </label>
                <button
                  type="button"
                  onClick={() => copyText(parsedHashtags.join(' '), 'OPP_HASH')}
                  disabled={parsedHashtags.length === 0}
                  className="text-[11px] font-extrabold text-nexa-brand hover:underline flex items-center gap-1 disabled:opacity-40 cursor-pointer"
                >
                  <Copy className="w-3 h-3" /> {copiedKey === 'OPP_HASH' ? 'Copied!' : 'Copy Hashtags'}
                </button>
              </div>
              <input
                type="text"
                value={hashtagsInput}
                onChange={(e) => setHashtagsInput(e.target.value)}
                placeholder="#Documentary #Biography #TrueStory"
                className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary focus:outline-none focus:border-nexa-brand"
              />
              {parsedHashtags.length > 0 && (
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {parsedHashtags.map((h, idx) => (
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
