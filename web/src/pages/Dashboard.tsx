import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
  Activity,
  Database,
  Tv,
  AlertTriangle,
  CheckCircle2,
  RefreshCw,
  Play,
  Terminal,
  ArrowRight,
  ShieldCheck,
  FolderTree,
  Calendar,
  Image as ImageIcon
} from 'lucide-react';
import { api, SystemStatus, Stream } from '../api';
import { IPTVBanner } from '../components/IPTVBanner';
import { StreamLogsModal } from '../components/StreamLogsModal';

export const Dashboard: React.FC = () => {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [streams, setStreams] = useState<Stream[]>([]);
  const [loading, setLoading] = useState(true);
  const [viewingLogsStream, setViewingLogsStream] = useState<Stream | null>(null);
  const [recoveringId, setRecoveringId] = useState<string | null>(null);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadDataQuietly, 4000);
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [sys, stms] = await Promise.all([
        api.getSystemStatus(),
        api.getStreams(),
      ]);
      setStatus(sys);
      setStreams(stms || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadDataQuietly = async () => {
    try {
      const [sys, stms] = await Promise.all([
        api.getSystemStatus(),
        api.getStreams(),
      ]);
      setStatus(sys);
      setStreams(stms || []);
    } catch (err) {
      console.error(err);
    }
  };

  const handleManualRecover = async (streamId: string) => {
    setRecoveringId(streamId);
    try {
      await api.startStream(streamId);
      loadDataQuietly();
    } catch (err: any) {
      alert(err.message || 'Recovery attempt failed');
    } finally {
      setTimeout(() => setRecoveringId(null), 1000);
    }
  };

  const problemStreams = streams.filter((s) => s.status === 'error');

  return (
    <div>
      <IPTVBanner />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-8">
        {/* Header */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center space-x-2.5">
              <Activity className="w-6 h-6 text-indigo-400" />
              <span>Birds-Eye View Dashboard</span>
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Real-time health telemetry, active FFmpeg stream processes, server resources, and troubleshooter.
            </p>
          </div>

          <button
            onClick={loadData}
            title="Refresh telemetry"
            className="flex items-center space-x-2 px-3 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-800 text-xs font-semibold transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh Telemetry</span>
          </button>
        </div>

        {/* Telemetry Metrics Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Active Streams Card */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-lg relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400">Active Transcodes</span>
              <div className="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center">
                <Tv className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline space-x-2">
              <span className="text-3xl font-extrabold text-white">
                {status?.active_streams_count || 0}
              </span>
              <span className="text-xs text-slate-500">
                / {status?.stream_count || 0} total channels
              </span>
            </div>
            <div className="mt-2 text-[11px] text-slate-400 flex items-center space-x-1.5">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
              <span>FFmpeg processes running</span>
            </div>
          </div>

          {/* CPU Cores & Concurrency Card */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-lg relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400">CPU & Concurrency</span>
              <div className="w-8 h-8 rounded-lg bg-blue-500/10 text-blue-400 flex items-center justify-center">
                <Activity className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline space-x-2">
              <span className="text-3xl font-extrabold text-white">
                {status?.cpu_cores || 0}
              </span>
              <span className="text-xs text-slate-500">
                CPU Cores
              </span>
            </div>
            <div className="mt-2 text-[11px] text-slate-400">
              Active Goroutines: <strong className="text-slate-200">{status?.goroutines || 0}</strong>
            </div>
          </div>

          {/* App Memory Footprint Card */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-lg relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400">App Memory Heap</span>
              <div className="w-8 h-8 rounded-lg bg-purple-500/10 text-purple-400 flex items-center justify-center">
                <Database className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline space-x-2">
              <span className="text-3xl font-extrabold text-white">
                {status?.memory_alloc_mb ? status.memory_alloc_mb.toFixed(1) : '0'}
              </span>
              <span className="text-xs text-slate-500">
                MB allocated
              </span>
            </div>
            <div className="mt-2 text-[11px] text-slate-400">
              System Reserved: <strong className="text-slate-200">{status?.memory_sys_mb ? status.memory_sys_mb.toFixed(1) : '0'} MB</strong>
            </div>
          </div>

          {/* Storage Directory Card */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 shadow-lg relative overflow-hidden">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400">Unified Storage</span>
              <div className="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
                <Database className="w-4 h-4" />
              </div>
            </div>
            <div className="mt-4 truncate">
              <span className="text-sm font-mono font-bold text-slate-200" title={status?.data_dir}>
                {status?.data_dir || './data'}
              </span>
            </div>
            <div className="mt-2 text-[11px] text-emerald-400 flex items-center space-x-1">
              <ShieldCheck className="w-3.5 h-3.5" />
              <span>SQLite + Logos + EPG unified</span>
            </div>
          </div>
        </div>


        {/* Problem Troubleshooter & Health Remediation */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className={`w-9 h-9 rounded-xl flex items-center justify-center border ${
                problemStreams.length > 0
                  ? 'bg-red-500/10 text-red-400 border-red-500/20'
                  : 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
              }`}>
                {problemStreams.length > 0 ? (
                  <AlertTriangle className="w-5 h-5 animate-pulse" />
                ) : (
                  <CheckCircle2 className="w-5 h-5" />
                )}
              </div>
              <div>
                <h3 className="text-base font-bold text-white flex items-center space-x-2">
                  <span>Stream Health Troubleshooter</span>
                  <span className={`text-[11px] px-2 py-0.5 rounded-full font-semibold border ${
                    problemStreams.length > 0
                      ? 'bg-red-500/15 text-red-400 border-red-500/30'
                      : 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                  }`}>
                    {problemStreams.length > 0 ? `${problemStreams.length} Issues Detected` : 'All Streams Healthy'}
                  </span>
                </h3>
                <p className="text-xs text-slate-400">
                  Automated stream watchdog monitors FFmpeg health and triggers configurable self-recovery.
                </p>
              </div>
            </div>
          </div>

          {problemStreams.length === 0 ? (
            <div className="bg-slate-950/60 border border-emerald-500/15 rounded-2xl p-6 text-center text-xs text-slate-400">
              <p className="text-emerald-400 font-semibold mb-1">Zero Stream Errors Reported</p>
              <span>All active channels are either transcoding cleanly or resting in idle on-demand mode.</span>
            </div>
          ) : (
            <div className="divide-y divide-slate-800 border border-red-500/20 rounded-2xl overflow-hidden bg-slate-950">
              {problemStreams.map((st) => (
                <div key={st.id} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div className="space-y-1">
                    <div className="flex items-center space-x-2">
                      <span className="font-semibold text-slate-200 text-sm">{st.name}</span>
                      <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">
                        {st.slug}
                      </span>
                      <span className="text-[10px] px-2 py-0.5 rounded-full bg-red-500/10 text-red-400 border border-red-500/20 uppercase font-bold">
                        Error
                      </span>
                    </div>
                    <p className="text-xs text-slate-400 font-mono truncate max-w-lg">
                      Source: {st.media_url}
                    </p>
                    <div className="text-[11px] text-indigo-400 flex items-center space-x-1 pt-1">
                      <ShieldCheck className="w-3.5 h-3.5" />
                      <span>
                        Auto-Recovery Watchdog: {st.auto_recover !== false ? `Active (limit ${st.recover_timeout_sec || 30}s)` : 'Disabled'}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2 shrink-0">
                    <button
                      onClick={() => handleManualRecover(st.id)}
                      disabled={recoveringId === st.id}
                      className="flex items-center space-x-1 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold transition-all shadow-md shadow-indigo-600/20"
                    >
                      <Play className="w-3.5 h-3.5" />
                      <span>{recoveringId === st.id ? 'Recovering...' : 'Restart Now'}</span>
                    </button>

                    <button
                      onClick={() => setViewingLogsStream(st)}
                      className="flex items-center space-x-1 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-medium border border-slate-700"
                    >
                      <Terminal className="w-3.5 h-3.5" />
                      <span>View Error Log</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Quick Navigation Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Link
            to="/"
            className="bg-slate-900 border border-slate-800 hover:border-indigo-500/40 rounded-2xl p-5 transition-all group shadow-lg"
          >
            <div className="flex items-center justify-between mb-2">
              <Tv className="w-5 h-5 text-indigo-400" />
              <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-indigo-400 group-hover:translate-x-1 transition-all" />
            </div>
            <h4 className="text-sm font-bold text-white">Manage Channels</h4>
            <p className="text-xs text-slate-400 mt-1">Configure playback URLs, categories, EPG, and logos.</p>
          </Link>

          <Link
            to="/epg"
            className="bg-slate-900 border border-slate-800 hover:border-purple-500/40 rounded-2xl p-5 transition-all group shadow-lg"
          >
            <div className="flex items-center justify-between mb-2">
              <Calendar className="w-5 h-5 text-purple-400" />
              <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-purple-400 group-hover:translate-x-1 transition-all" />
            </div>
            <h4 className="text-sm font-bold text-white">EPG Guide Providers</h4>
            <p className="text-xs text-slate-400 mt-1">Sync schedules, manage providers, and view combined feed.</p>
          </Link>

          <Link
            to="/logos"
            className="bg-slate-900 border border-slate-800 hover:border-blue-500/40 rounded-2xl p-5 transition-all group shadow-lg"
          >
            <div className="flex items-center justify-between mb-2">
              <ImageIcon className="w-5 h-5 text-blue-400" />
              <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-blue-400 group-hover:translate-x-1 transition-all" />
            </div>
            <h4 className="text-sm font-bold text-white">Channel Logo Assets</h4>
            <p className="text-xs text-slate-400 mt-1">Upload and store local logos in the app data directory.</p>
          </Link>
        </div>
      </div>

      {viewingLogsStream && (
        <StreamLogsModal
          stream={viewingLogsStream}
          onClose={() => setViewingLogsStream(null)}
        />
      )}
    </div>
  );
};
