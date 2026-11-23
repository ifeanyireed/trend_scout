import React, { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import { DashboardPage, DiscoveryFeedPage, LoginPage, OpportunityDetailPage } from './pages/DiscoveryPages';
import { MyDocumentariesPage, ProductionStudioPage, ResearchLibraryPage, WatchlistPage } from './pages/StudioPages';
import { NotificationsPage, ReportsPage, SettingsPage, SystemHealthPage } from './pages/SystemPages';
import { ensureValidAccessToken, hasActiveSession, isTokenExpired, getAuthToken, getRefreshToken, clearAuthSession } from './api/client';

const RequireAuth: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const [checking, setChecking] = useState(true);
  const [authed, setAuthed] = useState(() => hasActiveSession());

  useEffect(() => {
    let active = true;
    const verify = async () => {
      if (!hasActiveSession()) {
        if (active) {
          setAuthed(false);
          setChecking(false);
        }
        return;
      }
      const ok = await ensureValidAccessToken();
      if (active) {
        setAuthed(ok);
        setChecking(false);
      }
    };
    verify();

    const handleExpired = () => {
      if (!active) return;
      setAuthed(false);
      navigate('/login?expired=1', {
        replace: true,
        state: { from: location.pathname + location.search, expired: true },
      });
    };
    window.addEventListener('ts-auth-expired', handleExpired);

    // Periodic JWT session expiry monitor (every 20 seconds)
    const interval = setInterval(async () => {
      const tok = getAuthToken();
      const ref = getRefreshToken();
      if (!tok || isTokenExpired(tok, 30)) {
        if (ref && !isTokenExpired(ref, 10)) {
          const ok = await ensureValidAccessToken();
          if (!ok) handleExpired();
        } else {
          clearAuthSession();
          handleExpired();
        }
      }
    }, 20000);

    return () => {
      active = false;
      window.removeEventListener('ts-auth-expired', handleExpired);
      clearInterval(interval);
    };
  }, [navigate, location.pathname, location.search]);

  if (checking) {
    return (
      <div className="min-h-screen bg-nexa-bg-base flex flex-col items-center justify-center gap-3">
        <div className="w-8 h-8 rounded-full border-2 border-nexa-brand border-t-transparent animate-spin" />
        <div className="text-xs font-bold text-nexa-text-muted">Verifying secure studio session...</div>
      </div>
    );
  }

  if (!authed) {
    return <Navigate to="/login" state={{ from: location.pathname + location.search }} replace />;
  }

  return <>{children}</>;
};

export const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/*"
          element={
            <RequireAuth>
              <Layout>
                <Routes>
                  <Route path="/" element={<DashboardPage />} />
                  <Route path="/discover" element={<DiscoveryFeedPage mode="discover" />} />
                  <Route path="/trending" element={<DiscoveryFeedPage mode="trending" />} />
                  <Route path="/emerging" element={<DiscoveryFeedPage mode="emerging" />} />
                  <Route path="/underdogs" element={<DiscoveryFeedPage mode="underdogs" />} />
                  <Route path="/archived" element={<DiscoveryFeedPage mode="archived" />} />
                  <Route path="/opportunities/:id" element={<OpportunityDetailPage />} />
                  <Route path="/watchlist" element={<WatchlistPage />} />
                  <Route path="/research" element={<ResearchLibraryPage />} />
                  <Route path="/documentaries" element={<MyDocumentariesPage />} />
                  <Route path="/production" element={<ProductionStudioPage />} />
                  <Route path="/production/:id" element={<ProductionStudioPage />} />
                  <Route path="/notifications" element={<NotificationsPage />} />
                  <Route path="/reports" element={<ReportsPage />} />
                  <Route path="/settings" element={<SettingsPage />} />
                  <Route path="/health" element={<SystemHealthPage />} />
                  <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
              </Layout>
            </RequireAuth>
          }
        />
      </Routes>
    </BrowserRouter>
  );
};
