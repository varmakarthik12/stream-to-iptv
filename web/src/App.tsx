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
              <Route path="/" element={<Streams />} />
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/categories" element={<Categories />} />
              <Route path="/logos" element={<Logos />} />
              <Route path="/epg" element={<EPG />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </main>
        </div>
      )}
    </BrowserRouter>
  );
};
