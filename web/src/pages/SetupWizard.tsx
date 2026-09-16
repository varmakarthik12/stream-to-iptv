import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Radio, ShieldCheck, Download, KeyRound, User as UserIcon, CheckCircle2, AlertCircle } from 'lucide-react';
import { api, SetupStatus } from '../api';

interface SetupWizardProps {
  onSetupComplete: () => void;
}

export const SetupWizard: React.FC<SetupWizardProps> = ({ onSetupComplete }) => {
  const navigate = useNavigate();
  const [status, setStatus] = useState<SetupStatus | null>(null);
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [importExisting, setImportExisting] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    checkStatus();
  }, []);

  const checkStatus = async () => {
    try {
      const res = await api.getSetupStatus();
      setStatus(res);
      if (!res.setup_required) {
        navigate('/login');
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password) {
      setError('Please fill in all required fields');
      return;
    }
    if (password.length < 4) {
      setError('Password must be at least 4 characters');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.setup({
        username: username.trim(),
        password,
        import_existing: importExisting,
      });
      onSetupComplete();
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Setup failed');
    } finally {
      setLoading(false);
    }
  };

  if (!status) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-400">
        Loading setup status...
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col justify-center items-center px-4 bg-slate-950 selection:bg-indigo-500 selection:text-white relative overflow-hidden">
      {/* Background glow */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-[500px] h-[300px] bg-indigo-600/15 blur-[120px] rounded-full pointer-events-none" />

      <div className="max-w-md w-full bg-slate-900 border border-slate-800 rounded-3xl p-8 shadow-2xl relative z-10">
        <div className="text-center mb-8">
          <div className="w-14 h-14 mx-auto rounded-2xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/30 mb-4">
            <Radio className="w-7 h-7 text-white" />
          </div>
          <h2 className="text-2xl font-bold text-white tracking-tight">
            Initial System Setup
          </h2>
          <p className="text-xs text-slate-400 mt-2">
            Welcome to Stream to IPTV 2.0. Create your administrator credentials to initialize the database and secure your management console.
          </p>
        </div>

        {error && (
          <div className="mb-6 p-3.5 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-red-400 text-xs">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center space-x-1.5">
              <UserIcon className="w-3.5 h-3.5 text-indigo-400" />
              <span>Admin Username</span>
            </label>
            <input
              type="text"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center space-x-1.5">
              <KeyRound className="w-3.5 h-3.5 text-indigo-400" />
              <span>Admin Password</span>
            </label>
            <input
              type="password"
              required
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5">
              Confirm Password
            </label>
            <input
              type="password"
              required
              placeholder="••••••••"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
            />
          </div>

          {status.has_config_to_import && (
            <div className="p-4 bg-indigo-950/30 border border-indigo-500/20 rounded-2xl">
              <label className="flex items-start space-x-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={importExisting}
                  onChange={(e) => setImportExisting(e.target.checked)}
                  className="mt-1 rounded bg-slate-900 border-slate-800 text-indigo-600 focus:ring-0"
                />
                <div>
                  <span className="text-xs font-semibold text-indigo-300 flex items-center space-x-1.5">
                    <Download className="w-3.5 h-3.5 text-indigo-400" />
                    <span>Migrate existing config.json channels</span>
                  </span>
                  <p className="text-[11px] text-slate-400 mt-1">
                    Detected an existing config.json file in the repository. Check this box to automatically import all your existing channels into SQLite.
                  </p>
                </div>
              </label>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full mt-4 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-2.5 px-4 rounded-xl text-sm shadow-lg shadow-indigo-600/25 transition-all flex items-center justify-center space-x-2"
          >
            <ShieldCheck className="w-4 h-4" />
            <span>{loading ? 'Initializing...' : 'Complete Setup & Launch'}</span>
          </button>
        </form>
      </div>
    </div>
  );
};
