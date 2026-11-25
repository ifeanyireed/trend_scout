import React, { useState, useEffect } from 'react';
import { NavLink, useNavigate, useLocation } from 'react-router-dom';
import {
  Archive, Compass, Flame, Sparkles, Trophy, Bookmark, BookOpen, Film, Clapperboard,
  Bell, FileSpreadsheet, Settings, Activity, LayoutDashboard, LogOut, RefreshCw,
  ChevronLeft, ChevronRight, Search, X, Sun, Moon, ShieldCheck, UserPlus, ArrowRight
} from 'lucide-react';
import { apiFetch, clearAuthSession, getAuthUser, getSessionRemainingMs } from '../api/client';

const NAV_ITEMS = [
  { to: '/', label: 'Master Overview', short: 'Overview', icon: LayoutDashboard, badge: 'HUB', section: 'Intelligence & Hub' },
  { to: '/discover', label: 'Discover People', short: 'Discover', icon: Compass, badge: 'Live', section: 'Intelligence & Hub' },
  { to: '/trending', label: 'Trending Now', short: 'Trending', icon: Flame, badge: 'VPH', section: 'Intelligence & Hub' },
  { to: '/emerging', label: 'Emerging Stories', short: 'Emerging', icon: Sparkles, badge: 'Early', section: 'Intelligence & Hub' },
  { to: '/underdogs', label: 'Underdog Opportunities', short: 'Underdogs', icon: Trophy, badge: 'Ranked', section: 'Intelligence & Hub' },
  { to: '/archived', label: 'Archived', short: 'Archived', icon: Archive, badge: 'Vault', section: 'Intelligence & Hub' },

  { to: '/research', label: 'Research Library', short: 'Research', icon: BookOpen, badge: 'Verified', section: 'Biographical Dossier' },
  { to: '/watchlist', label: 'Watchlist Pipeline', short: 'Watchlist', icon: Bookmark, badge: 'Alerts', section: 'Biographical Dossier' },

  { to: '/documentaries', label: 'My Documentaries', short: 'Docs', icon: Film, badge: '20m', section: 'Documentary Studio' },
  { to: '/production', label: 'Production Studio', short: 'Studio', icon: Clapperboard, badge: 'Flow', section: 'Documentary Studio' },

  { to: '/notifications', label: 'Notifications & Alerts', short: 'Alerts', icon: Bell, badge: 'Feed', section: 'Governance & Infra' },
  { to: '/reports', label: 'Intelligence Reports', short: 'Reports', icon: FileSpreadsheet, badge: 'Top 10', section: 'Governance & Infra' },
  { to: '/health', label: 'System & Queue Health', short: 'System', icon: Activity, badge: 'PG 18', section: 'Governance & Infra' },
  { to: '/settings', label: 'Platform Settings', short: 'Settings', icon: Settings, badge: 'Config', section: 'Governance & Infra' },
];

export const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [scanning, setScanning] = useState(false);
  const [scanBanner, setScanBanner] = useState<{ type: 'success' | 'error' | 'info'; message: string } | null>(null);
  const [suggestModalOpen, setSuggestModalOpen] = useState(false);
  const [suggestName, setSuggestName] = useState('');
  const [suggestAngle, setSuggestAngle] = useState('');
  const [suggestCategory, setSuggestCategory] = useState('auto');
  const [suggestCountry, setSuggestCountry] = useState('');
  const [suggestBusy, setSuggestBusy] = useState(false);
  const [suggestError, setSuggestError] = useState('');
  const [isDark, setIsDark] = useState(() => localStorage.getItem('ts_theme') === 'dark');
  const [sessionLabel, setSessionLabel] = useState('');
  const authUser = getAuthUser();
  const userInitials = (authUser?.name || authUser?.email || 'LP')
    .split(/\s+/)
    .map((p) => p[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();

  useEffect(() => {
    const updateSessionLabel = () => {
      const remMs = getSessionRemainingMs();
      if (remMs <= 0) {
        setSessionLabel('Refreshing...');
        return;
      }
      const totalMins = Math.ceil(remMs / 60000);
      if (totalMins >= 60) {
        const hrs = Math.floor(totalMins / 60);
        const mins = totalMins % 60;
        setSessionLabel(`${hrs}h ${mins}m left`);
      } else {
        setSessionLabel(`${totalMins}m left`);
      }
    };
    updateSessionLabel();
    const timer = setInterval(updateSessionLabel, 30000);
    window.addEventListener('ts-auth-updated', updateSessionLabel);
    return () => {
      clearInterval(timer);
      window.removeEventListener('ts-auth-updated', updateSessionLabel);
    };
  }, []);

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('ts_theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('ts_theme', 'light');
    }
  }, [isDark]);

  const handleScanNow = async (silent = false) => {
    setScanning(true);
    if (!silent) {
      setScanBanner({ type: 'info', message: 'Scanning live sources (Google News, Google Trends, Reddit, Wikipedia, YouTube)...' });
    }
    try {
      const res = await apiFetch('/api/discoveries/scan', { method: 'POST', body: '{}' });
      window.dispatchEvent(new CustomEvent('ts-refresh'));
      const newCount = res?.newDiscoveries ?? 0;
      const fetched = res?.candidatesFetched ?? 0;
      const sec = res?.durationMs ? (res.durationMs / 1000).toFixed(1) : '0';
      setScanBanner({
        type: 'success',
        message:
          newCount > 0
            ? `Live scan complete (${sec}s): +${newCount} new opportunities discovered from ${fetched} live signals.`
            : `Live scan complete (${sec}s): Checked ${fetched} live signals & refreshed velocity metrics.`,
      });
      setTimeout(() => setScanBanner(null), 7000);
    } catch (err: any) {
      if (!silent) {
        setScanBanner({
          type: 'error',
          message: `Live scan failed: ${err?.message || 'Unable to reach discovery engine'}`,
        });
      }
    } finally {
      setScanning(false);
    }
  };

  // Allow pages (e.g. empty-state CTAs) to trigger the live discovery scan bar or Suggest a Person modal
  useEffect(() => {
    const onTrigger = () => handleScanNow(false);
    const onOpenSuggest = (e: Event) => {
      const custom = e as CustomEvent;
      if (custom?.detail?.personName) {
        setSuggestName(String(custom.detail.personName));
      }
      if (custom?.detail?.focusAngle) {
        setSuggestAngle(String(custom.detail.focusAngle));
      }
      setSuggestError('');
      setSuggestModalOpen(true);
    };
    window.addEventListener('ts-trigger-scan', onTrigger);
    window.addEventListener('ts-open-suggest-person', onOpenSuggest);
    return () => {
      window.removeEventListener('ts-trigger-scan', onTrigger);
      window.removeEventListener('ts-open-suggest-person', onOpenSuggest);
    };
  }, []);

  const handleSuggestPersonSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cleanName = suggestName.trim();
    if (!cleanName) {
      setSuggestError('Please enter the name of the person you want to scout.');
      return;
    }
    setSuggestBusy(true);
    setSuggestError('');
    try {
      const res = await apiFetch('/api/discoveries/suggest', {
        method: 'POST',
        body: JSON.stringify({
          personName: cleanName,
          focusAngle: suggestAngle.trim(),
          category: suggestCategory === 'auto' ? '' : suggestCategory,
          country: suggestCountry.trim(),
        }),
      });
      window.dispatchEvent(new CustomEvent('ts-refresh'));
      setSuggestModalOpen(false);
      setSuggestName('');
      setSuggestAngle('');
      setSuggestCategory('auto');
      setSuggestCountry('');
      const src = res?.sourcesScanned || {};
      const sec = res?.durationMs ? (res.durationMs / 1000).toFixed(1) : '0';
      setScanBanner({
        type: 'success',
        message: `Pulled opportunity for ${res.personName || cleanName} in ${sec}s (News: ${src['src-google-news'] ?? 0}, YouTube: ${src['src-youtube'] ?? 0}, Reddit: ${src['src-reddit'] ?? 0}, Wikipedia: ${src['src-wikipedia'] ?? 0}).`,
      });
      setTimeout(() => setScanBanner(null), 8000);
      if (res?.discoveryId) {
        navigate(`/opportunities/${res.discoveryId}`);
      }
    } catch (err: any) {
      setSuggestError(err?.message || 'Failed to scout and pull opportunity for this person.');
    } finally {
      setSuggestBusy(false);
    }
  };

  // Automatic live freshness check: if database is empty or last scan is > 30 mins old, sync live sources once per session
  useEffect(() => {
    if (sessionStorage.getItem('ts_auto_synced') === '1') return;
    apiFetch('/api/dashboard/overview')
      .then((ov) => {
        const count = ov?.metrics?.newDiscoveries ?? 0;
        const lastScan = ov?.monitoringStatus?.lastScanAt;
        const isStale = !lastScan || Date.now() - new Date(lastScan).getTime() > 30 * 60 * 1000;
        if (count === 0 || isStale) {
          sessionStorage.setItem('ts_auto_synced', '1');
          handleScanNow(true);
        }
      })
      .catch(() => {});
  }, []);

  const filteredNavItems = NAV_ITEMS.filter((item) =>
    item.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
    item.section.toLowerCase().includes(searchQuery.toLowerCase())
  );

  let currentSection = '';

  return (
    <div className="min-h-screen bg-nexa-bg-base text-nexa-text-primary flex flex-col md:flex-row relative font-sans">
      {/* MOBILE TOP HEADER BAR (< md) */}
      <header className="md:hidden sticky top-0 z-40 bg-nexa-bg-surface/95 backdrop-blur-md border-b border-nexa-border px-4 py-2.5 flex items-center justify-between">
        <div onClick={() => navigate('/')} className="flex items-center gap-2.5 cursor-pointer">
          <img
            src="/logo.png"
            alt="TrendScout Studio"
            className="w-8 h-8 rounded-xl object-contain shrink-0"
          />
          <div>
            <div className="flex items-center gap-1.5">
              <span className="text-xs font-black text-display text-nexa-text-primary">TrendScout Studio</span>
              <span className="text-[8px] font-extrabold px-1.5 py-0.5 rounded-full uppercase bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20">
                PRODUCER
              </span>
            </div>
            <div className="text-[9px] font-bold text-nexa-text-muted">Cloud Studio Online</div>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={() => {
              setSuggestError('');
              setSuggestModalOpen(true);
            }}
            className="px-2.5 py-1 rounded-full border border-nexa-brand/40 bg-nexa-brand/10 text-nexa-brand text-[10px] font-extrabold flex items-center gap-1"
          >
            <UserPlus className="w-3 h-3" />
            Suggest
          </button>
          <button
            onClick={() => handleScanNow(false)}
            disabled={scanning}
            className="px-2.5 py-1 rounded-full bg-nexa-brand text-white text-[10px] font-extrabold flex items-center gap-1"
          >
            <RefreshCw className={`w-3 h-3 ${scanning ? 'animate-spin' : ''}`} />
            Scan
          </button>
          <button
            onClick={() => setIsDark(!isDark)}
            className="p-1.5 rounded-full border border-nexa-border text-nexa-text-secondary hover:bg-nexa-bg-base"
          >
            {isDark ? <Sun className="w-3.5 h-3.5 text-amber-400" /> : <Moon className="w-3.5 h-3.5" />}
          </button>
        </div>
      </header>

      {/* DESKTOP SIDEBAR (ofia_ng SuperAdminShell architecture) */}
      <aside
        className={`hidden md:flex bg-nexa-bg-surface border-r border-nexa-border transition-all duration-300 flex-col z-30 sticky top-0 h-screen shrink-0 ${
          isSidebarOpen ? 'w-72' : 'w-20'
        }`}
      >
        {/* COLLAPSE TOGGLE BUTTON */}
        <button
          onClick={() => setIsSidebarOpen(!isSidebarOpen)}
          className="absolute -right-3 top-24 w-6 h-6 bg-nexa-bg-surface border border-nexa-border shadow-md rounded-full flex items-center justify-center text-nexa-text-secondary hover:text-nexa-brand hover:scale-110 transition-transform z-50 cursor-pointer"
          title={isSidebarOpen ? 'Collapse sidebar' : 'Expand sidebar'}
        >
          {isSidebarOpen ? <ChevronLeft className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
        </button>

        {/* BRAND HEADER */}
        <div className="p-6 pb-3 flex items-center justify-between">
          {isSidebarOpen ? (
            <div onClick={() => navigate('/')} className="flex items-center gap-3 cursor-pointer min-w-0">
              <img
                src="/logo.png"
                alt="TrendScout Studio"
                className="w-10 h-10 rounded-2xl object-contain shrink-0"
              />
              <div className="flex flex-col min-w-0">
                <span className="text-sm font-black text-display leading-tight text-nexa-text-primary truncate">
                  TrendScout Studio
                </span>
                <span className="text-[11px] font-black text-nexa-brand tracking-wide uppercase mt-0.5">
                  DOCUMENTARY OS
                </span>
                <span className="text-[9px] font-bold text-nexa-text-muted uppercase tracking-wider">
                  Cloud Platform Active
                </span>
              </div>
            </div>
          ) : (
            <img
              onClick={() => navigate('/')}
              src="/logo.png"
              alt="TrendScout Studio"
              className="w-10 h-10 mx-auto rounded-2xl object-contain cursor-pointer"
            />
          )}
        </div>

        {/* SEARCH CONSOLE PILL + LIVE SCAN ACTION */}
        <div className="px-4 py-2 space-y-2">
          {isSidebarOpen ? (
            <>
              <div className="flex items-center bg-nexa-bg-base px-3.5 py-2 rounded-full border border-nexa-border gap-2.5 w-full focus-within:border-nexa-brand transition-all">
                <Search className="w-3.5 h-3.5 text-nexa-text-muted shrink-0" />
                <input
                  type="text"
                  placeholder="Search console..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="bg-transparent text-xs outline-none w-full text-nexa-text-primary placeholder:text-nexa-text-muted font-medium"
                />
                {searchQuery && (
                  <button
                    type="button"
                    onClick={() => setSearchQuery('')}
                    className="text-xs text-nexa-text-muted hover:text-nexa-text-primary cursor-pointer"
                  >
                    <X className="w-3 h-3" />
                  </button>
                )}
              </div>

              <button
                onClick={() => handleScanNow(false)}
                disabled={scanning}
                className="w-full py-2.5 px-4 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid text-white font-bold text-xs flex items-center justify-center gap-2 shadow-md shadow-nexa-brand/20 transition cursor-pointer"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${scanning ? 'animate-spin' : ''}`} />
                <span>{scanning ? 'Scanning Live Sources...' : 'Run Discovery Scan Now'}</span>
              </button>

              <button
                type="button"
                onClick={() => {
                  setSuggestError('');
                  setSuggestModalOpen(true);
                }}
                className="w-full py-2 px-4 rounded-full border border-nexa-brand/40 bg-nexa-brand/10 hover:bg-nexa-brand hover:text-white text-nexa-brand font-extrabold text-xs flex items-center justify-center gap-2 transition cursor-pointer"
              >
                <UserPlus className="w-3.5 h-3.5" />
                <span>Suggest a Person to Scout</span>
              </button>
            </>
          ) : (
            <div className="space-y-2 flex flex-col items-center">
              <button
                onClick={() => handleScanNow(false)}
                disabled={scanning}
                title="Run Discovery Scan Now"
                className="w-10 h-10 mx-auto rounded-full bg-nexa-brand text-white flex items-center justify-center shadow-md shadow-nexa-brand/20 cursor-pointer"
              >
                <RefreshCw className={`w-4 h-4 ${scanning ? 'animate-spin' : ''}`} />
              </button>
              <button
                type="button"
                onClick={() => {
                  setSuggestError('');
                  setSuggestModalOpen(true);
                }}
                title="Suggest a Person & Pull Opportunity"
                className="w-10 h-10 mx-auto rounded-full border border-nexa-brand/40 bg-nexa-brand/10 text-nexa-brand hover:bg-nexa-brand hover:text-white flex items-center justify-center transition cursor-pointer"
              >
                <UserPlus className="w-4 h-4" />
              </button>
            </div>
          )}
        </div>

        {/* CATEGORIZED ROUNDED-FULL PILL NAVIGATION */}
        <nav className="flex-1 px-4 space-y-1 mt-2 overflow-y-auto scrollbar-hide">
          {filteredNavItems.map((item, i) => {
            const Icon = item.icon;
            const isActive =
              item.to === '/'
                ? location.pathname === '/'
                : location.pathname === item.to ||
                  (item.to.startsWith('/production') && location.pathname.startsWith('/production'));

            const showSectionHeader = item.section !== currentSection;
            if (showSectionHeader) {
              currentSection = item.section;
            }

            return (
              <React.Fragment key={item.to}>
                {showSectionHeader && (
                  <div className={`pt-3 pb-1 ${i === 0 ? 'pt-0' : ''}`}>
                    {isSidebarOpen ? (
                      <div className="text-[10px] font-extrabold uppercase tracking-wider text-nexa-text-muted px-3">
                        {item.section}
                      </div>
                    ) : (
                      <div className="w-8 h-[1px] bg-nexa-border mx-auto my-1" />
                    )}
                  </div>
                )}

                <NavLink to={item.to}>
                  <div
                    className={`w-full flex items-center gap-3.5 p-3 rounded-full transition-all group mb-1 cursor-pointer ${
                      isActive
                        ? 'bg-nexa-brand text-white shadow-lg shadow-nexa-brand/25 font-bold'
                        : 'text-nexa-text-secondary hover:bg-nexa-bg-base hover:text-nexa-text-primary font-semibold'
                    }`}
                    title={!isSidebarOpen ? item.label : undefined}
                  >
                    <Icon
                      className={`w-5 h-5 shrink-0 transition-transform group-hover:scale-110 ${
                        isActive ? 'text-white' : 'text-nexa-text-muted group-hover:text-nexa-brand'
                      }`}
                    />
                    {isSidebarOpen && (
                      <div className="flex-1 flex items-center justify-between text-left min-w-0">
                        <span className="text-[14px] truncate">{item.label}</span>
                        {item.badge && (
                          <span
                            className={`text-[9px] font-extrabold px-2 py-0.5 rounded-full shrink-0 ${
                              isActive ? 'bg-white/20 text-white' : 'bg-nexa-brand/10 text-nexa-brand'
                            }`}
                          >
                            {item.badge}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                </NavLink>
              </React.Fragment>
            );
          })}
        </nav>

        {/* FOOTER OPERATOR CARD & THEME/LOGOUT CONTROLS */}
        <div className="p-4 border-t border-nexa-border space-y-2">
          {isSidebarOpen ? (
            <div className="flex items-center justify-between p-2.5 rounded-2xl bg-nexa-bg-base/80 border border-nexa-border">
              <div className="flex items-center gap-2.5 min-w-0">
                <div className="w-9 h-9 rounded-full bg-nexa-brand/15 border border-nexa-brand/30 text-nexa-brand font-black text-xs flex items-center justify-center shrink-0 relative">
                  {userInitials}
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-white dark:border-slate-900 absolute -bottom-0.5 -right-0.5" />
                </div>
                <div className="min-w-0">
                  <p className="text-xs font-bold text-nexa-text-primary truncate">
                    {authUser?.name || 'Lead Producer'}
                  </p>
                  <div className="flex items-center gap-1.5 mt-0.5 flex-wrap">
                    <span className="text-[8px] font-extrabold px-1.5 py-0.2 rounded-full uppercase bg-nexa-brand/10 text-nexa-brand border border-nexa-brand/20">
                      {(authUser?.role || 'ADMIN').toUpperCase()}
                    </span>
                    <span className="text-[9px] text-emerald-500 font-bold" title="Active session">
                      • {sessionLabel || 'Active'}
                    </span>
                  </div>
                </div>
              </div>
              <button
                onClick={() => navigate('/notifications')}
                className="p-2 rounded-full border border-nexa-border text-nexa-text-secondary hover:text-nexa-brand hover:bg-nexa-bg-surface cursor-pointer"
                title="Notifications"
              >
                <Bell className="w-4 h-4" />
              </button>
            </div>
          ) : null}

          <div className="flex items-center justify-between gap-2">
            <button
              onClick={async () => {
                try {
                  await apiFetch('/api/auth/logout', { method: 'POST', body: '{}' });
                } catch (_) {}
                clearAuthSession();
                navigate('/login', { replace: true });
              }}
              className={`flex items-center gap-3 p-2.5 rounded-full text-rose-500 hover:bg-rose-500/10 transition-all cursor-pointer ${
                isSidebarOpen ? 'flex-1 px-3.5' : 'w-full justify-center'
              }`}
              title="Log Out"
            >
              <LogOut className="w-4 h-4 shrink-0" />
              {isSidebarOpen && <span className="font-bold text-xs">Log Out</span>}
            </button>

            <button
              onClick={() => setIsDark(!isDark)}
              className="p-2.5 rounded-full border border-nexa-border bg-nexa-bg-base text-nexa-text-secondary hover:text-nexa-brand transition-colors cursor-pointer"
              title="Toggle Light / Dark Mode"
            >
              {isDark ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4" />}
            </button>
          </div>
        </div>
      </aside>

      {/* MAIN WORKSPACE CONTENT */}
      <main className="flex-1 flex flex-col min-w-0 overflow-y-auto pb-24 md:pb-10">
        <div className="max-w-7xl w-full mx-auto px-3.5 sm:px-6 md:px-8 py-5 sm:py-7 space-y-6">
          {scanBanner && (
            <div
              className={`px-4 py-3 rounded-2xl border text-xs font-bold flex items-center justify-between gap-3 ${
                scanBanner.type === 'error'
                  ? 'bg-rose-500/10 border-rose-500/30 text-rose-500'
                  : scanBanner.type === 'success'
                  ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-600 dark:text-emerald-400'
                  : 'bg-nexa-brand/10 border-nexa-brand/30 text-nexa-brand'
              }`}
            >
              <span>{scanBanner.message}</span>
              <button
                type="button"
                onClick={() => setScanBanner(null)}
                className="opacity-70 hover:opacity-100 cursor-pointer"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
          {children}
        </div>
      </main>

      {/* SUGGEST A PERSON & PULL OPPORTUNITY MODAL */}
      {suggestModalOpen && (
        <div
          className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
          onClick={() => !suggestBusy && setSuggestModalOpen(false)}
        >
          <div
            className="nexa-card rounded-3xl border border-nexa-border max-w-lg w-full p-6 sm:p-7 shadow-2xl space-y-5"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="w-11 h-11 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/25 text-nexa-brand flex items-center justify-center shrink-0">
                  <UserPlus className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-black text-display text-nexa-text-primary">
                    Suggest a Person &amp; Pull Opportunity
                  </h2>
                  <p className="text-xs text-nexa-text-secondary">
                    The system goes out across Wikipedia, Google News, YouTube &amp; Reddit to pull and score the full documentary opportunity.
                  </p>
                </div>
              </div>
              <button
                type="button"
                disabled={suggestBusy}
                onClick={() => setSuggestModalOpen(false)}
                className="p-1.5 rounded-full border border-nexa-border text-nexa-text-muted hover:text-nexa-text-primary cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {suggestError && (
              <div className="p-3.5 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-xs font-bold text-rose-500">
                {suggestError}
              </div>
            )}

            <form onSubmit={handleSuggestPersonSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-extrabold uppercase tracking-wider text-nexa-text-secondary mb-1.5">
                  Person Name *
                </label>
                <input
                  type="text"
                  required
                  autoFocus
                  disabled={suggestBusy}
                  placeholder="e.g., Katalin Karikó, Jensen Huang, Lamine Yamal, Demis Hassabis..."
                  value={suggestName}
                  onChange={(e) => setSuggestName(e.target.value)}
                  className="w-full px-4 py-3 rounded-2xl border border-nexa-border bg-nexa-bg-base text-sm text-nexa-text-primary outline-none focus:border-nexa-brand transition font-semibold"
                />
              </div>

              <div>
                <label className="block text-xs font-extrabold uppercase tracking-wider text-nexa-text-secondary mb-1.5">
                  Documentary Angle / Focus (Optional)
                </label>
                <input
                  type="text"
                  disabled={suggestBusy}
                  placeholder="e.g., Underdog origin story, early rejections, world record breakthrough..."
                  value={suggestAngle}
                  onChange={(e) => setSuggestAngle(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary outline-none focus:border-nexa-brand transition"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-extrabold uppercase tracking-wider text-nexa-text-secondary mb-1.5">
                    Category
                  </label>
                  <select
                    disabled={suggestBusy}
                    value={suggestCategory}
                    onChange={(e) => setSuggestCategory(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary font-bold outline-none focus:border-nexa-brand"
                  >
                    <option value="auto">Auto-Detect from Live Sources</option>
                    <option value="Science & Medicine">Science &amp; Medicine</option>
                    <option value="Technology & Innovation">Technology &amp; Innovation</option>
                    <option value="Business & Founders">Business &amp; Founders</option>
                    <option value="Arts, Film & Music">Arts, Film &amp; Music</option>
                    <option value="Humanitarian & Civic Impact">Humanitarian &amp; Civic Impact</option>
                    <option value="Sports & Competition">Sports &amp; Competition</option>
                    <option value="Mind Sports & Chess">Mind Sports &amp; Chess</option>
                    <option value="Exploration & Survival">Exploration &amp; Survival</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-extrabold uppercase tracking-wider text-nexa-text-secondary mb-1.5">
                    Country / Region (Optional)
                  </label>
                  <input
                    type="text"
                    disabled={suggestBusy}
                    placeholder="Auto-Detect (or e.g. United States)"
                    value={suggestCountry}
                    onChange={(e) => setSuggestCountry(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-2xl border border-nexa-border bg-nexa-bg-base text-xs text-nexa-text-primary outline-none focus:border-nexa-brand"
                  />
                </div>
              </div>

              {suggestBusy && (
                <div className="p-3.5 rounded-2xl bg-nexa-brand/10 border border-nexa-brand/25 text-xs text-nexa-brand font-bold flex items-center gap-2.5">
                  <RefreshCw className="w-4 h-4 animate-spin shrink-0" />
                  <span>
                    Scouting Wikipedia, Google News RSS, YouTube velocity &amp; channel baselines, Reddit discussions, and scoring 8 underdog dimensions...
                  </span>
                </div>
              )}

              <div className="flex items-center justify-end gap-2.5 pt-2">
                <button
                  type="button"
                  disabled={suggestBusy}
                  onClick={() => setSuggestModalOpen(false)}
                  className="px-4 py-2.5 rounded-full border border-nexa-border text-xs font-bold text-nexa-text-secondary hover:text-nexa-text-primary cursor-pointer"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={suggestBusy}
                  className="px-5 py-2.5 rounded-full bg-nexa-brand hover:bg-nexa-brand-mid disabled:opacity-60 text-white text-xs font-extrabold flex items-center gap-2 shadow-lg shadow-nexa-brand/25 cursor-pointer"
                >
                  {suggestBusy ? (
                    <>
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      <span>Pulling Live Opportunity...</span>
                    </>
                  ) : (
                    <>
                      <Sparkles className="w-3.5 h-3.5" />
                      <span>Scout &amp; Pull Opportunity</span>
                      <ArrowRight className="w-3.5 h-3.5" />
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MOBILE BOTTOM PILL BAR */}
      <nav className="md:hidden fixed bottom-0 inset-x-0 z-50 bg-nexa-bg-surface/95 backdrop-blur-md border-t border-nexa-border shadow-2xl">
        <div className="flex items-center gap-1.5 px-2 py-1.5 overflow-x-auto scrollbar-hide">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon;
            const isActive =
              item.to === '/'
                ? location.pathname === '/'
                : location.pathname.startsWith(item.to);
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={`shrink-0 flex flex-col items-center justify-center min-w-[64px] px-2 py-1 rounded-2xl transition-all ${
                  isActive ? 'text-nexa-brand font-bold' : 'text-nexa-text-muted'
                }`}
              >
                <span
                  className={`w-10 h-7 rounded-full flex items-center justify-center ${
                    isActive ? 'bg-nexa-brand text-white' : 'text-nexa-text-secondary'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                </span>
                <span className="text-[10px] mt-1 truncate">{item.short}</span>
              </NavLink>
            );
          })}
        </div>
      </nav>
    </div>
  );
};
