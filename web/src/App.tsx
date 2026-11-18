import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import { DashboardPage, DiscoveryFeedPage, LoginPage, OpportunityDetailPage } from './pages/DiscoveryPages';
import { MyDocumentariesPage, ProductionStudioPage, ResearchLibraryPage, WatchlistPage } from './pages/StudioPages';
import { NotificationsPage, ReportsPage, SettingsPage, SystemHealthPage } from './pages/SystemPages';

export const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/*"
          element={
            <Layout>
              <Routes>
                <Route path="/" element={<DashboardPage />} />
                <Route path="/discover" element={<DiscoveryFeedPage mode="discover" />} />
                <Route path="/trending" element={<DiscoveryFeedPage mode="trending" />} />
                <Route path="/emerging" element={<DiscoveryFeedPage mode="emerging" />} />
                <Route path="/underdogs" element={<DiscoveryFeedPage mode="underdogs" />} />
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
          }
        />
      </Routes>
    </BrowserRouter>
  );
};
