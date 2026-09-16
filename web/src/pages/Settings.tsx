import React, { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, Server, Shield, HardDrive, Check, Copy, AlertCircle, RefreshCw, Terminal } from 'lucide-react';
import { api, SystemStatus } from '../api';

export const Settings: React.FC = () => {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedDocker, setCopiedDocker] = useState(false);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setLoading(true);
    try {
      const [data, sys] = await Promise.all([
        api.getSettings(),
        api.getSystemStatus(),
      ]);
      setSettings(data || {});
      setSystemStatus(sys);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleFieldChange = (key: string, value: string) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setSavedSuccess(false);

    try {
      await api.updateSettings(settings);
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 3000);
      loadSettings();
    } catch (err: any) {
      setError(err.message || 'Failed to update settings');
    } finally {
      setSaving(false);
    }
  };

  const dockerCommand = `docker run -d \\
  --name stream-to-iptv \\
  -p ${settings.port || '8068'}:8068 \\
  -v $(pwd)/data:/app/data \\
  --restart unless-stopped \\
  ghcr.io/varmakarthik12/stream-to-iptv:latest`;

  const copyDockerCmd = () => {
    navigator.clipboard.writeText(dockerCommand);
    setCopiedDocker(true);
    setTimeout(() => setCopiedDocker(false), 2000);
  };

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-16 text-center text-slate-500 text-sm">
        Loading system configuration...
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center space-x-2">
            <SettingsIcon className="w-6 h-6 text-indigo-400" />
            <span>System Settings</span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Global server configuration, IPTV feed token security, and storage volume bindings.
          </p>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-center space-x-2 text-red-400 text-xs">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {savedSuccess && (
        <div className="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-2xl flex items-center space-x-2 text-emerald-400 text-xs">
          <Check className="w-4 h-4 shrink-0" />
          <span>Settings saved successfully. Changes take effect immediately.</span>
        </div>
      )}

      <form onSubmit={handleSave} className="space-y-8">
        {/* Network & Server Settings */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl space-y-5">
          <h3 className="text-base font-bold text-white flex items-center space-x-2">
            <Server className="w-4 h-4 text-indigo-400" />
            <span>Network & Endpoints</span>
          </h3>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                Server Listening Port
              </label>
              <input
                type="text"
                placeholder="8068"
                value={settings.port || '8068'}
                onChange={(e) => handleFieldChange('port', e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 font-mono"
              />
              <span className="text-[11px] text-slate-500 mt-1 block">Default port is 8068</span>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                Public Base URL Override (Optional)
              </label>
              <input
                type="text"
                placeholder="http://192.168.1.100:8068 or https://iptv.example.com"
                value={settings.server_url || ''}
                onChange={(e) => handleFieldChange('server_url', e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 font-mono"
              />
              <span className="text-[11px] text-slate-500 mt-1 block">
                Overrides the host domain used in M3U URLs and EPG guide references.
              </span>
            </div>
          </div>
        </div>

        {/* FFmpeg Engine Configuration */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl space-y-5">
          <h3 className="text-base font-bold text-white flex items-center space-x-2">
            <Terminal className="w-4 h-4 text-indigo-400" />
            <span>FFmpeg Transcoding Engine</span>
          </h3>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5">
              Custom FFmpeg Executable Path (Optional)
            </label>
            <input
              type="text"
              placeholder={systemStatus?.ffmpeg_path || 'Auto-detected system FFmpeg'}
              value={settings.ffmpeg_path || ''}
              onChange={(e) => handleFieldChange('ffmpeg_path', e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 font-mono"
            />
            <div className="flex flex-col sm:flex-row sm:items-center justify-between text-[11px] text-slate-500 mt-1 gap-1">
              <span>Leave blank to automatically discover system FFmpeg without wrapper shims.</span>
              {systemStatus?.ffmpeg_path && (
                <span className="text-slate-400 font-mono">
                  Active: <strong className="text-indigo-300">{systemStatus.ffmpeg_path}</strong>
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Security & IPTV Player Access */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl space-y-5">
          <h3 className="text-base font-bold text-white flex items-center space-x-2">
            <Shield className="w-4 h-4 text-indigo-400" />
            <span>IPTV Feed Security Token</span>
          </h3>

          <div>
            <label className="block text-xs font-semibold text-slate-300 mb-1.5">
              Secret Access Token (Optional)
            </label>
            <input
              type="text"
              placeholder="e.g. my-secret-key-123"
              value={settings.iptv_token || ''}
              onChange={(e) => handleFieldChange('iptv_token', e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 font-mono"
            />
            <span className="text-[11px] text-slate-500 mt-1 block">
              Leave blank to keep IPTV endpoints publicly accessible on your LAN. If filled, requires <code className="text-slate-400">?token=&lt;key&gt;</code> on /playlist.m3u and /epg.xml.
            </span>
          </div>
        </div>

        {/* Storage & Volume Persistence Card */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl space-y-4">
          <h3 className="text-base font-bold text-white flex items-center space-x-2">
            <HardDrive className="w-4 h-4 text-indigo-400" />
            <span>Unified Storage & Docker Volume Mount</span>
          </h3>
          <p className="text-xs text-slate-400">
            All state, database records, logos, and cached EPGs are stored in a single root directory:
          </p>

          <div className="bg-slate-950 p-3 rounded-xl border border-slate-800 text-xs font-mono text-indigo-300">
            Current Data Directory: {systemStatus?.data_dir || './data'}
          </div>

          <div className="space-y-2 pt-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-300">Recommended Docker Run Command:</span>
              <button
                type="button"
                onClick={copyDockerCmd}
                className="flex items-center space-x-1 text-xs text-indigo-400 hover:text-indigo-300 transition-colors"
              >
                {copiedDocker ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                <span>{copiedDocker ? 'Copied Command' : 'Copy'}</span>
              </button>
            </div>
            <pre className="bg-slate-950 p-4 rounded-xl border border-slate-800 text-xs font-mono text-slate-300 overflow-x-auto leading-relaxed">
              {dockerCommand}
            </pre>
          </div>
        </div>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="flex items-center space-x-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-2.5 px-6 rounded-xl text-sm shadow-lg shadow-indigo-600/20 transition-all"
          >
            <Save className="w-4 h-4" />
            <span>{saving ? 'Saving...' : 'Save System Settings'}</span>
          </button>
        </div>
      </form>
    </div>
  );
};
