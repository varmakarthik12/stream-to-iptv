import React, { useState, useEffect } from 'react';
import { Calendar, Plus, RefreshCw, Trash2, Edit, CheckCircle2, AlertCircle, Copy, Check, Clock, Radio } from 'lucide-react';
import { api, EPGSource, SystemStatus } from '../api';
import { copyToClipboard } from '../utils/clipboard';

export const EPG: React.FC = () => {
  const [sources, setSources] = useState<EPGSource[]>([]);
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(true);

  // Form state
  const [addMode, setAddMode] = useState<'single' | 'bulk'>('single');
  const [bulkText, setBulkText] = useState('');
  const [bulkInterval, setBulkInterval] = useState(24);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [refreshInterval, setRefreshInterval] = useState(24);
  const [editingSource, setEditingSource] = useState<EPGSource | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [copiedXml, setCopiedXml] = useState(false);
  const [copiedGz, setCopiedGz] = useState(false);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadSourcesQuietly, 6000);
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [srcs, sys] = await Promise.all([
        api.getEPGSources(),
        api.getSystemStatus(),
      ]);
      setSources(srcs || []);
      setSystemStatus(sys);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadSourcesQuietly = async () => {
    try {
      const srcs = await api.getEPGSources();
      setSources(srcs || []);
    } catch (err) {
      console.error(err);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !url.trim()) return;

    setSubmitting(true);
    setError(null);
    try {
      if (editingSource) {
        await api.updateEPGSource(editingSource.id, {
          name: name.trim(),
          url: url.trim(),
          refresh_interval_hours: Number(refreshInterval) || 24,
        });
      } else {
        await api.createEPGSource({
          name: name.trim(),
          url: url.trim(),
          refresh_interval_hours: Number(refreshInterval) || 24,
        });
      }
      setName('');
      setUrl('');
      setRefreshInterval(24);
      setEditingSource(null);
      loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to save EPG source');
    } finally {
      setSubmitting(false);
    }
  };

  const handleBulkSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!bulkText.trim()) return;

    setSubmitting(true);
    setError(null);
    try {
      const lines = bulkText.split('\n');
      const sourcesToCreate: { name: string; url: string; refresh_interval_hours: number }[] = [];

      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) continue;

        let srcName = '';
        let srcUrl = '';

        if (trimmed.includes('|')) {
          const parts = trimmed.split('|');
          srcName = parts[0].trim();
          srcUrl = parts.slice(1).join('|').trim();
        } else {
          srcUrl = trimmed;
          try {
            const parsed = new URL(srcUrl);
            const pathParts = parsed.pathname.split('/').filter(Boolean);
            const fileName = pathParts[pathParts.length - 1] || parsed.hostname;
            srcName = `${parsed.hostname} (${fileName.replace(/\.(xml|gz)+$/i, '')})`;
          } catch {
            srcName = srcUrl;
          }
        }

        if (srcUrl.startsWith('http://') || srcUrl.startsWith('https://')) {
          sourcesToCreate.push({
            name: srcName || srcUrl,
            url: srcUrl,
            refresh_interval_hours: Number(bulkInterval) || 24,
          });
        }
      }

      if (sourcesToCreate.length === 0) {
        throw new Error('No valid HTTP/HTTPS URLs found in input.');
      }

      await api.bulkCreateEPGSources(sourcesToCreate);
      setBulkText('');
      setAddMode('single');
      loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to add multiple EPG sources');
    } finally {
      setSubmitting(false);
    }
  };

  const handleManualRefresh = async (src: EPGSource) => {
    try {
      await api.refreshEPGSource(src.id);
      loadSourcesQuietly();
    } catch (err: any) {
      alert(err.message || 'Failed to trigger refresh');
    }
  };

  const handleDelete = async (src: EPGSource) => {
    if (!confirm(`Delete EPG source "${src.name}"?`)) return;
    try {
      await api.deleteEPGSource(src.id);
      loadData();
    } catch (err: any) {
      alert(err.message || 'Failed to delete source');
    }
  };

  const handleEdit = (src: EPGSource) => {
    setEditingSource(src);
    setName(src.name);
    setUrl(src.url);
    setRefreshInterval(src.refresh_interval_hours);
  };

  const serverBase = systemStatus?.base_url || window.location.origin;
  const epgXmlUrl = `${serverBase}/epg.xml`;
  const epgGzUrl = `${serverBase}/epg.xml.gz`;

  const copyUrl = async (u: string, type: 'xml' | 'gz') => {
    await copyToClipboard(u);
    if (type === 'xml') {
      setCopiedXml(true);
      setTimeout(() => setCopiedXml(false), 2000);
    } else {
      setCopiedGz(true);
      setTimeout(() => setCopiedGz(false), 2000);
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center space-x-2">
            <Calendar className="w-6 h-6 text-indigo-400" />
            <span>EPG TV Guide Sources</span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Manage multiple XMLTV EPG data providers, sync intervals, and fallback guide data.
          </p>
        </div>
      </div>

      {/* Generated Output Card */}
      <div className="bg-gradient-to-r from-purple-950/40 via-indigo-950/40 to-slate-900 border border-purple-500/20 rounded-3xl p-6 shadow-xl mb-8">
        <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
          <div>
            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-purple-500/20 text-purple-300 border border-purple-500/30 mb-2">
              Combined EPG Aggregator Output
            </span>
            <h3 className="text-lg font-bold text-white">Aggregated & Translated XMLTV Feed</h3>
            <p className="text-xs text-slate-400 max-w-xl mt-1">
              Stream to IPTV merges and translates guides from all active sources into a single XMLTV feed, filtered strictly to your configured streams.
            </p>
          </div>

          <div className="flex flex-col sm:flex-row gap-3 w-full md:w-auto">
            <div className="flex items-center bg-slate-900 border border-slate-800 rounded-xl p-1.5 pl-3 text-xs">
              <span className="text-purple-400 font-semibold mr-2 font-mono">XML:</span>
              <span className="text-slate-300 font-mono truncate max-w-[200px] mr-2">{epgXmlUrl}</span>
              <button
                onClick={() => copyUrl(epgXmlUrl, 'xml')}
                className="flex items-center space-x-1 bg-purple-600 hover:bg-purple-500 text-white px-2.5 py-1 rounded-lg text-xs font-semibold"
              >
                {copiedXml ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                <span>{copiedXml ? 'Copied' : 'Copy'}</span>
              </button>
            </div>

            <div className="flex items-center bg-slate-900 border border-slate-800 rounded-xl p-1.5 pl-3 text-xs">
              <span className="text-indigo-400 font-semibold mr-2 font-mono">GZIP:</span>
              <span className="text-slate-300 font-mono truncate max-w-[200px] mr-2">{epgGzUrl}</span>
              <button
                onClick={() => copyUrl(epgGzUrl, 'gz')}
                className="flex items-center space-x-1 bg-indigo-600 hover:bg-indigo-500 text-white px-2.5 py-1 rounded-lg text-xs font-semibold"
              >
                {copiedGz ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                <span>{copiedGz ? 'Copied' : 'Copy'}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Source Addition Form */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl h-fit">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-base font-bold text-white flex items-center space-x-2">
              <Plus className="w-4 h-4 text-indigo-400" />
              <span>{editingSource ? 'Edit EPG Source' : 'Add EPG Source'}</span>
            </h3>
            {!editingSource && (
              <div className="flex bg-slate-950 p-1 rounded-xl border border-slate-800">
                <button
                  type="button"
                  onClick={() => setAddMode('single')}
                  className={`px-2.5 py-1 text-[11px] rounded-lg font-medium transition-all ${
                    addMode === 'single' ? 'bg-indigo-600 text-white shadow' : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  Single
                </button>
                <button
                  type="button"
                  onClick={() => setAddMode('bulk')}
                  className={`px-2.5 py-1 text-[11px] rounded-lg font-medium transition-all ${
                    addMode === 'bulk' ? 'bg-indigo-600 text-white shadow' : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  Add Multiple
                </button>
              </div>
            )}
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-red-400 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {addMode === 'bulk' && !editingSource ? (
            <form onSubmit={handleBulkSave} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  Multiple EPG URLs (one per line) <span className="text-red-400">*</span>
                </label>
                <textarea
                  rows={5}
                  required
                  placeholder={`https://avkb.short.gy/epg.xml.gz\nUS Sports | https://example.com/sports.xml.gz\nUK Guide | https://example.com/uk.xml`}
                  value={bulkText}
                  onChange={(e) => setBulkText(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
                <span className="text-[11px] text-slate-500 mt-1 block">
                  Format: <code>URL</code> or <code>Provider Name | URL</code>. Blank lines and lines starting with # are ignored.
                </span>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center space-x-1.5">
                  <Clock className="w-3.5 h-3.5 text-indigo-400" />
                  <span>Refresh Interval for All (Hours)</span>
                </label>
                <input
                  type="number"
                  min="1"
                  max="168"
                  value={bulkInterval}
                  onChange={(e) => setBulkInterval(Number(e.target.value))}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={submitting || !bulkText.trim()}
                  className="w-full px-5 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-xl text-xs font-semibold shadow-lg shadow-indigo-600/20 transition-all"
                >
                  {submitting ? 'Adding Multiple Sources...' : 'Add All EPG Sources'}
                </button>
              </div>
            </form>
          ) : (
            <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                Provider Name <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                required
                placeholder="e.g. US Cable EPG / Sports Guide"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                XMLTV URL (.xml or .xml.gz) <span className="text-red-400">*</span>
              </label>
              <input
                type="url"
                required
                placeholder="https://example.com/epg.xml.gz"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
              />
              <span className="text-[11px] text-slate-500 mt-1 block">
                Supports raw XML and gzip compressed XMLTV feeds.
              </span>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center space-x-1.5">
                <Clock className="w-3.5 h-3.5 text-indigo-400" />
                <span>Refresh Interval (Hours)</span>
              </label>
              <input
                type="number"
                min="1"
                max="168"
                value={refreshInterval}
                onChange={(e) => setRefreshInterval(Number(e.target.value))}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500"
              />
              <span className="text-[11px] text-slate-500 mt-1 block">
                How frequently to auto-download and update program schedules (e.g. 12 or 24 hours).
              </span>
            </div>

            <div className="flex items-center space-x-3 pt-2">
              {editingSource && (
                <button
                  type="button"
                  onClick={() => {
                    setEditingSource(null);
                    setName('');
                    setUrl('');
                    setRefreshInterval(24);
                  }}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs font-semibold"
                >
                  Cancel
                </button>
              )}
              <button
                type="submit"
                disabled={submitting}
                className="flex-1 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-2 px-4 rounded-xl text-xs shadow-lg shadow-indigo-600/20 transition-all flex items-center justify-center space-x-2"
              >
                <Plus className="w-4 h-4" />
                <span>{submitting ? 'Saving...' : editingSource ? 'Update Source' : 'Add Source'}</span>
              </button>
            </div>
          </form>
          )}
        </div>

        {/* Sources List Table */}
        <div className="lg:col-span-2">
          <div className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-xl">
            <div className="p-4 border-b border-slate-800 bg-slate-900/80">
              <span className="text-xs font-semibold text-slate-400">Configured Sources: {sources.length}</span>
            </div>

            {loading && sources.length === 0 ? (
              <div className="p-12 text-center text-slate-500 text-sm">Loading EPG sources...</div>
            ) : sources.length === 0 ? (
              <div className="p-12 text-center text-slate-500 text-sm italic">
                No EPG sources added yet. Add one using the form on the left.
              </div>
            ) : (
              <div className="divide-y divide-slate-800/60">
                {sources.map((src) => {
                  const isUpdating = src.status === 'updating';
                  const isOk = src.status === 'ok';
                  const isError = src.status === 'error';

                  return (
                    <div key={src.id} className="p-5 hover:bg-slate-800/30 transition-colors space-y-3">
                      <div className="flex items-start justify-between">
                        <div>
                          <div className="flex items-center space-x-2">
                            <h4 className="text-sm font-semibold text-slate-200">{src.name}</h4>
                            <span className={`text-[10px] uppercase font-semibold px-2 py-0.5 rounded-full border ${
                              isUpdating
                                ? 'bg-amber-500/10 text-amber-400 border-amber-500/20 animate-pulse'
                                : isOk
                                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                : isError
                                ? 'bg-red-500/10 text-red-400 border-red-500/20'
                                : 'bg-slate-800 text-slate-400 border-slate-700'
                            }`}>
                              {src.status}
                            </span>
                          </div>
                          <span className="text-xs text-slate-500 font-mono block mt-1 truncate max-w-md" title={src.url}>
                            {src.url}
                          </span>
                        </div>

                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => handleManualRefresh(src)}
                            disabled={isUpdating}
                            title="Refresh EPG now"
                            className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg transition-colors disabled:opacity-50"
                          >
                            <RefreshCw className={`w-4 h-4 ${isUpdating ? 'animate-spin' : ''}`} />
                          </button>

                          <button
                            onClick={() => handleEdit(src)}
                            title="Edit source"
                            className="p-1.5 text-slate-400 hover:text-indigo-400 hover:bg-indigo-500/10 rounded-lg transition-colors"
                          >
                            <Edit className="w-4 h-4" />
                          </button>

                          <button
                            onClick={() => handleDelete(src)}
                            title="Delete source"
                            className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>

                      <div className="flex flex-wrap items-center gap-4 text-xs text-slate-400 pt-2 border-t border-slate-800/40">
                        <span>Channels Parsed: <strong className="text-slate-200">{src.channel_count}</strong></span>
                        <span>Interval: Every <strong className="text-slate-200">{src.refresh_interval_hours}h</strong></span>
                        {src.last_refreshed_at && (
                          <span>Last Sync: <strong className="text-slate-200">{new Date(src.last_refreshed_at).toLocaleString()}</strong></span>
                        )}
                        {src.status_message && (
                          <span className="text-red-400 font-medium">{src.status_message}</span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
