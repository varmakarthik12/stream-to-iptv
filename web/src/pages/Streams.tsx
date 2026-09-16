import React, { useState, useEffect } from 'react';
import { Plus, Search, Filter, Play, Square, Terminal, Edit, Trash2, Copy, Check, Radio, Tv, Layers, ExternalLink, RefreshCw, MonitorPlay } from 'lucide-react';
import { api, Stream, Category } from '../api';
import { IPTVBanner } from '../components/IPTVBanner';
import { StreamEditorModal } from '../components/StreamEditorModal';
import { StreamLogsModal } from '../components/StreamLogsModal';
import { StreamPlayerModal } from '../components/StreamPlayerModal';

const ChannelLogo: React.FC<{ url?: string; name: string }> = ({ url, name }) => {
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [url]);

  if (!url || failed) {
    return <Tv className="w-5 h-5 text-slate-600" />;
  }

  return (
    <img
      src={url}
      alt={name}
      className="w-8 h-8 object-contain"
      onError={() => setFailed(true)}
      loading="lazy"
    />
  );
};

export const Streams: React.FC = () => {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [selectedStatus, setSelectedStatus] = useState('all');

  // Modals
  const [editingStream, setEditingStream] = useState<Stream | null | undefined>(undefined);
  const [viewingLogsStream, setViewingLogsStream] = useState<Stream | null>(null);
  const [playingStream, setPlayingStream] = useState<Stream | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadStreamsQuietly, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [streamsData, categoriesData] = await Promise.all([
        api.getStreams(),
        api.getCategories(),
      ]);
      setStreams(streamsData || []);
      setCategories(categoriesData || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadStreamsQuietly = async () => {
    try {
      const data = await api.getStreams();
      setStreams(data || []);
    } catch (err) {
      console.error(err);
    }
  };

  const handleStartStream = async (st: Stream) => {
    try {
      await api.startStream(st.id);
      loadStreamsQuietly();
    } catch (err: any) {
      alert(err.message || 'Failed to start stream');
    }
  };

  const handleStopStream = async (st: Stream) => {
    try {
      await api.stopStream(st.id);
      loadStreamsQuietly();
    } catch (err: any) {
      alert(err.message || 'Failed to stop stream');
    }
  };

  const handleDeleteStream = async (st: Stream) => {
    if (!confirm(`Are you sure you want to delete stream "${st.name}"?`)) return;
    try {
      await api.deleteStream(st.id);
      loadData();
    } catch (err: any) {
      alert(err.message || 'Failed to delete stream');
    }
  };

  const copyPlaybackUrl = (st: Stream) => {
    const url = st.playback_url || `${window.location.origin}/stream/${st.slug}/${st.slug}.m3u8`;
    navigator.clipboard.writeText(url);
    setCopiedId(st.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  // Filter streams
  const filteredStreams = streams.filter((st) => {
    const matchesSearch =
      (st.name || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (st.slug || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (st.tvg_id || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
      (st.tvg_chno || '').toLowerCase().includes(searchTerm.toLowerCase());

    const matchesCategory =
      selectedCategory === 'all' ||
      st.categories?.some((c) => c.id === selectedCategory);

    const matchesStatus =
      selectedStatus === 'all' ||
      (st.status || 'idle') === selectedStatus;

    return matchesSearch && matchesCategory && matchesStatus;
  });

  // Sort streams by channel number (numerical ascending) by default
  const sortedAndFilteredStreams = [...filteredStreams].sort((a, b) => {
    const chnoA = parseInt(a.tvg_chno || '', 10);
    const chnoB = parseInt(b.tvg_chno || '', 10);
    const hasA = !isNaN(chnoA) && chnoA > 0;
    const hasB = !isNaN(chnoB) && chnoB > 0;

    if (hasA && hasB) {
      if (chnoA !== chnoB) return chnoA - chnoB;
      return a.name.localeCompare(b.name);
    }
    if (hasA) return -1;
    if (hasB) return 1;
    return a.name.localeCompare(b.name);
  });

  return (
    <div>
      <IPTVBanner />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {/* Action Header */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white tracking-tight flex items-center space-x-2">
              <Tv className="w-6 h-6 text-indigo-400" />
              <span>Channels & Streams</span>
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Configure M3U8 IPTV channels, transcode lifecycle, logos, and EPG mappings. Sorted by channel number by default.
            </p>
          </div>

          <div className="flex items-center space-x-3 w-full sm:w-auto">
            <button
              onClick={loadData}
              title="Refresh channels"
              className="p-2.5 rounded-xl bg-slate-900 hover:bg-slate-800 text-slate-400 hover:text-slate-200 border border-slate-800 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={() => setEditingStream(null)}
              className="flex items-center space-x-2 bg-indigo-600 hover:bg-indigo-500 text-white px-4 py-2.5 rounded-xl text-sm font-semibold shadow-lg shadow-indigo-600/20 transition-all w-full sm:w-auto justify-center"
            >
              <Plus className="w-4 h-4" />
              <span>Add Channel</span>
            </button>
          </div>
        </div>

        {/* Filters Bar */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-6">
          <div className="relative">
            <Search className="w-4 h-4 text-slate-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Search by name, slug, TVG ID, or channel #..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-xl pl-9 pr-3.5 py-2 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500"
            />
          </div>

          <div>
            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500"
            >
              <option value="all">All Categories</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <select
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500"
            >
              <option value="all">All Statuses</option>
              <option value="running">Running</option>
              <option value="starting">Starting</option>
              <option value="idle">Idle</option>
              <option value="error">Error</option>
              <option value="stopped">Stopped</option>
            </select>
          </div>
        </div>

        {/* Channels List Table */}
        {loading && streams.length === 0 ? (
          <div className="text-center py-20 text-slate-500 text-sm">
            Loading channels...
          </div>
        ) : sortedAndFilteredStreams.length === 0 ? (
          <div className="bg-slate-900 border border-slate-800 rounded-3xl p-12 text-center">
            <div className="w-12 h-12 rounded-2xl bg-indigo-500/10 text-indigo-400 flex items-center justify-center mx-auto mb-4 border border-indigo-500/20">
              <Tv className="w-6 h-6" />
            </div>
            <h3 className="text-base font-bold text-white mb-1">No channels found</h3>
            <p className="text-xs text-slate-400 max-w-sm mx-auto mb-6">
              {searchTerm || selectedCategory !== 'all' || selectedStatus !== 'all'
                ? 'Try adjusting your filters or search terms.'
                : 'Get started by creating your first IPTV stream channel.'}
            </p>
            <button
              onClick={() => setEditingStream(null)}
              className="inline-flex items-center space-x-2 bg-indigo-600 hover:bg-indigo-500 text-white px-4 py-2 rounded-xl text-xs font-semibold"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Add Channel</span>
            </button>
          </div>
        ) : (
          <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-slate-800 text-[11px] uppercase tracking-wider text-slate-400 font-semibold bg-slate-900/90">
                    <th className="py-3.5 px-3 w-16 text-center">Ch #</th>
                    <th className="py-3.5 px-4">Channel</th>
                    <th className="py-3.5 px-4">Generated Playback URL</th>
                    <th className="py-3.5 px-4">Categories</th>
                    <th className="py-3.5 px-4">Mode</th>
                    <th className="py-3.5 px-4">Status</th>
                    <th className="py-3.5 px-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60 text-xs">
                  {sortedAndFilteredStreams.map((st) => {
                    const status = st.status || 'idle';
                    const isRunning = status === 'running';
                    const isStarting = status === 'starting';
                    const isError = status === 'error';

                    return (
                      <tr key={st.id} className="hover:bg-slate-800/40 transition-colors">
                        {/* Channel Number Column */}
                        <td className="py-3 px-3 text-center">
                          {st.tvg_chno ? (
                            <span className="inline-flex items-center justify-center px-2 py-1 rounded-lg bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 font-mono font-bold text-xs">
                              #{st.tvg_chno}
                            </span>
                          ) : (
                            <span className="text-slate-600 font-mono text-xs">—</span>
                          )}
                        </td>

                        {/* Channel Column */}
                        <td className="py-3 px-4">
                          <div className="flex items-center space-x-3">
                            <div className="w-10 h-10 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-center overflow-hidden shrink-0">
                              <ChannelLogo url={st.logo_url} name={st.name} />
                            </div>
                            <div>
                              <div className="flex items-center space-x-2">
                                <span className="font-semibold text-slate-200">{st.name}</span>
                              </div>
                              <div className="flex items-center space-x-2 text-[11px] text-slate-400 mt-0.5 font-mono">
                                <span>{st.slug}</span>
                                {st.tvg_id && (
                                  <span className="text-slate-500">· {st.tvg_id}</span>
                                )}
                              </div>
                            </div>
                          </div>
                        </td>

                        {/* Generated Playback URL Column */}
                        <td className="py-3 px-4 max-w-xs">
                          <div className="flex items-center space-x-1.5">
                            <input
                              type="text"
                              readOnly
                              value={st.playback_url || `${window.location.origin}/stream/${st.slug}/${st.slug}.m3u8`}
                              className="w-48 bg-slate-950 border border-slate-800 rounded-md px-2 py-1 text-[11px] font-mono text-slate-400 truncate select-all cursor-default focus:outline-none"
                            />
                            <button
                              onClick={() => copyPlaybackUrl(st)}
                              title="Copy playback URL"
                              className="p-1.5 rounded-md bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors shrink-0"
                            >
                              {copiedId === st.id ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                            </button>
                          </div>
                        </td>

                        {/* Categories Column */}
                        <td className="py-3 px-4">
                          <div className="flex flex-wrap gap-1 max-w-[200px]">
                            {st.categories && st.categories.length > 0 ? (
                              st.categories.map((c) => (
                                <span
                                  key={c.id}
                                  className="inline-block text-[10px] px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-300 border border-indigo-500/20"
                                >
                                  {c.name}
                                </span>
                              ))
                            ) : (
                              <span className="text-[11px] text-slate-600 italic">General</span>
                            )}
                          </div>
                        </td>

                        {/* Mode Column */}
                        <td className="py-3 px-4">
                          <span className={`text-[11px] font-medium px-2 py-0.5 rounded-md ${
                            st.mode === 'always_on'
                              ? 'bg-purple-500/15 text-purple-300 border border-purple-500/25'
                              : 'bg-slate-800 text-slate-300'
                          }`}>
                            {st.mode === 'always_on' ? 'Always-On' : 'On-Demand'}
                          </span>
                        </td>

                        {/* Status Column */}
                        <td className="py-3 px-4">
                          <div className="flex items-center space-x-1.5">
                            <span className={`w-2 h-2 rounded-full ${
                              isRunning
                                ? 'bg-emerald-400 animate-pulse'
                                : isStarting
                                ? 'bg-amber-400 animate-ping'
                                : isError
                                ? 'bg-red-500'
                                : 'bg-slate-600'
                            }`} />
                            <span className={`capitalize font-semibold text-[11px] ${
                              isRunning
                                ? 'text-emerald-400'
                                : isStarting
                                ? 'text-amber-400'
                                : isError
                                ? 'text-red-400'
                                : 'text-slate-400'
                            }`}>
                              {status}
                            </span>
                          </div>
                        </td>

                        {/* Actions Column */}
                        <td className="py-3 px-4 text-right">
                          <div className="flex items-center justify-end space-x-1">
                            <button
                              onClick={() => setPlayingStream(st)}
                              title="Preview live channel in browser"
                              className="p-1.5 rounded-lg text-indigo-400 hover:text-white hover:bg-indigo-600/20 transition-colors"
                            >
                              <MonitorPlay className="w-3.5 h-3.5" />
                            </button>

                            {isRunning ? (
                              <button
                                onClick={() => handleStopStream(st)}
                                title="Stop FFmpeg process"
                                className="p-1.5 rounded-lg text-amber-400 hover:bg-amber-500/10 transition-colors"
                              >
                                <Square className="w-3.5 h-3.5" />
                              </button>
                            ) : (
                              <button
                                onClick={() => handleStartStream(st)}
                                title="Start FFmpeg stream"
                                className="p-1.5 rounded-lg text-emerald-400 hover:bg-emerald-500/10 transition-colors"
                              >
                                <Play className="w-3.5 h-3.5" />
                              </button>
                            )}

                            <button
                              onClick={() => setViewingLogsStream(st)}
                              title="View FFmpeg logs"
                              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
                            >
                              <Terminal className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={() => setEditingStream(st)}
                              title="Edit stream settings"
                              className="p-1.5 rounded-lg text-slate-400 hover:text-indigo-400 hover:bg-indigo-500/10 transition-colors"
                            >
                              <Edit className="w-3.5 h-3.5" />
                            </button>

                            <button
                              onClick={() => handleDeleteStream(st)}
                              title="Delete channel"
                              className="p-1.5 rounded-lg text-slate-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Modals */}
      {playingStream && (
        <StreamPlayerModal
          stream={playingStream}
          onClose={() => setPlayingStream(null)}
        />
      )}

      {editingStream !== undefined && (
        <StreamEditorModal
          stream={editingStream}
          onClose={() => setEditingStream(undefined)}
          onSave={() => {
            setEditingStream(undefined);
            loadData();
          }}
        />
      )}

      {viewingLogsStream && (
        <StreamLogsModal
          stream={viewingLogsStream}
          onClose={() => setViewingLogsStream(null)}
        />
      )}
    </div>
  );
};

