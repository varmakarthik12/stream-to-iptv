import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Radio, Lock, User as UserIcon, AlertCircle, ArrowRight } from 'lucide-react';
import { api } from '../api';

interface LoginProps {
  onLoginSuccess: () => void;
}

export const Login: React.FC<LoginProps> = ({ onLoginSuccess }) => {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password) {
      setError('Please enter username and password');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.login({ username: username.trim(), password });
      onLoginSuccess();
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Invalid username or password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col justify-center items-center px-4 bg-slate-950 selection:bg-indigo-500 selection:text-white relative overflow-hidden">
      {/* Glow */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-[450px] h-[250px] bg-indigo-600/15 blur-[120px] rounded-full pointer-events-none" />

      <div className="max-w-sm w-full bg-slate-900 border border-slate-800 rounded-3xl p-8 shadow-2xl relative z-10">
        <div className="text-center mb-8">
          <div className="w-12 h-12 mx-auto rounded-2xl bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center shadow-lg shadow-indigo-500/30 mb-4">
            <Radio className="w-6 h-6 text-white" />
          </div>
          <h2 className="text-2xl font-bold text-white tracking-tight">
            Sign In
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Access your IPTV management console
          </p>
        </div>

        {error && (
          <div className="mb-6 p-3.5 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-red-400 text-xs">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center space-x-1.5">
              <UserIcon className="w-3.5 h-3.5 text-indigo-400" />
              <span>Username</span>
            </label>
            <input
              type="text"
              required
              placeholder="Admin username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2.5 text-sm text-slate-100 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center space-x-1.5">
              <Lock className="w-3.5 h-3.5 text-indigo-400" />
              <span>Password</span>
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

          <button
            type="submit"
            disabled={loading}
            className="w-full mt-4 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-2.5 px-4 rounded-xl text-sm shadow-lg shadow-indigo-600/25 transition-all flex items-center justify-center space-x-2"
          >
            <span>{loading ? 'Signing in...' : 'Sign In'}</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </form>
      </div>
    </div>
  );
};
