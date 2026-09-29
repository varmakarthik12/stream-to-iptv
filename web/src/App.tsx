import React, { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { api, User } from './api';
import { Navbar } from './components/Navbar';
import { Dashboard } from './pages/Dashboard';
import { Streams } from './pages/Streams';
import { Categories } from './pages/Categories';
import { Logos } from './pages/Logos';
import { EPG } from './pages/EPG';
import { Settings } from './pages/Settings';
import { SetupWizard } from './pages/SetupWizard';
import { Login } from './pages/Login';

export const App: React.FC = () => {
  const [user, setUser] = useState<User | null>(null);
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    initAuth();
  }, []);

  const initAuth = async () => {
    setLoading(true);
    try {
      const setup = await api.getSetupStatus();
      setSetupRequired(setup.setup_required);

      if (!setup.setup_required) {
        try {
          const currentUser = await api.getMe();
          setUser(currentUser);
        } catch {
          setUser(null);
        }
      }
    } catch (err) {
      console.error('Failed to init auth', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">
        <div className="flex flex-col items-center space-y-3">
          <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
          <span className="text-xs font-mono">Loading Stream to IPTV...</span>
        </div>
      </div>
    );
  }

  return (
    <BrowserRouter>
      {setupRequired ? (
        <Routes>
          <Route path="/setup" element={<SetupWizard onSetupComplete={initAuth} />} />
          <Route path="*" element={<Navigate to="/setup" replace />} />
        </Routes>
      ) : !user ? (
        <Routes>
          <Route path="/login" element={<Login onLoginSuccess={initAuth} />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      ) : (
        <div className="min-h-screen bg-slate-950 flex flex-col">
          <Navbar user={user} onLogout={() => setUser(null)} />
          <main className="flex-1">
            <Routes>
              {/* Streams and Modal Routes */}
              <Route path="/" element={<Streams />} />
              <Route path="/streams" element={<Streams />} />
              <Route path="/streams/new" element={<Streams />} />
              <Route path="/streams/:id/edit" element={<Streams />} />
              <Route path="/streams/:id/logs" element={<Streams />} />
              <Route path="/streams/:id/play" element={<Streams />} />
              <Route path="/streams/edit/:id" element={<Streams />} />
              <Route path="/streams/logs/:id" element={<Streams />} />
              <Route path="/streams/play/:id" element={<Streams />} />

              {/* IPTV Player Setup Guide Modal Route */}
              <Route path="/guide" element={<Streams />} />
              <Route path="/setup-guide" element={<Streams />} />

              {/* Dashboard and Modal Routes */}
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/dashboard/guide" element={<Dashboard />} />
              <Route path="/dashboard/setup-guide" element={<Dashboard />} />
              <Route path="/dashboard/streams/:id/logs" element={<Dashboard />} />
              <Route path="/dashboard/logs/:id" element={<Dashboard />} />

              {/* Categories and Subroutes */}
              <Route path="/categories" element={<Categories />} />
              <Route path="/categories/new" element={<Categories />} />
              <Route path="/categories/:id/edit" element={<Categories />} />
              <Route path="/categories/edit/:id" element={<Categories />} />

              {/* Logos and Subroutes */}
              <Route path="/logos" element={<Logos />} />
              <Route path="/logos/upload" element={<Logos />} />
              <Route path="/logos/import" element={<Logos />} />

              {/* EPG Sources and Subroutes */}
              <Route path="/epg" element={<EPG />} />
              <Route path="/epg/new" element={<EPG />} />
              <Route path="/epg/bulk" element={<EPG />} />
              <Route path="/epg/:id/edit" element={<EPG />} />
              <Route path="/epg/edit/:id" element={<EPG />} />

              {/* Settings */}
              <Route path="/settings" element={<Settings />} />

              {/* Fallback */}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </main>
        </div>
      )}
    </BrowserRouter>
  );
};
