import React, { useState, useEffect } from 'react';
import { X, Copy, Check, Upload, Search, Film, Layers, Calendar, Image as ImageIcon, Cpu, AlertCircle } from 'lucide-react';
import { api, Stream, Category, Logo, EPGSource, EPGChannel } from '../api';

interface StreamEditorModalProps {
  stream?: Stream | null;
  onClose: () => void;
  onSave: () => void;
}

export const StreamEditorModal: React.FC<StreamEditorModalProps> = ({ stream, onClose, onSave }) => {
  // Form state
  const [name, setName] = useState(stream?.name || '');
  const [slug, setSlug] = useState(stream?.slug || '');
  const [mediaUrl, setMediaUrl] = useState(stream?.media_url || '');
  const [tvgId, setTvgId] = useState(stream?.tvg_id || '');
  const [tvgName, setTvgName] = useState(stream?.tvg_name || '');
  const [tvgChno, setTvgChno] = useState(stream?.tvg_chno || '');
  const [mode, setMode] = useState<'ondemand' | 'always_on'>(stream?.mode || 'ondemand');
  const [idleTimeoutSec, setIdleTimeoutSec] = useState(stream?.idle_timeout_sec || 180);
  const [autoRecover, setAutoRecover] = useState(stream ? (stream.auto_recover !== false) : true);
  const [recoverTimeoutSec, setRecoverTimeoutSec] = useState(stream?.recover_timeout_sec || 30);
  const [enabled, setEnabled] = useState(stream ? stream.enabled : true);

  // Inline Quick Creation
  const [newCategoryName, setNewCategoryName] = useState('');
  const [creatingCategory, setCreatingCategory] = useState(false);

  // Advanced FFmpeg
  const [bufferSize, setBufferSize] = useState(stream?.buffer_size || '1000000');
  const [fifoSize, setFifoSize] = useState(stream?.fifo_size || '');
  const [programId, setProgramId] = useState(stream?.program_id || '1');
  const [useGpu, setUseGpu] = useState(stream ? stream.use_gpu : false);
  const [overrunNonfatal, setOverrunNonfatal] = useState(stream ? stream.overrun_nonfatal : false);
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Logo state
  const [logoTab, setLogoTab] = useState<'library' | 'upload' | 'url'>('library');
  const [selectedLogoId, setSelectedLogoId] = useState(stream?.logo_id || '');
  const [customLogoUrl, setCustomLogoUrl] = useState(stream?.logo_url || '');
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploadName, setUploadName] = useState('');

  // Categories state
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<string[]>(
    stream?.category_ids || stream?.categories?.map((c) => c.id) || []
  );

  // EPG Mapping state
  const [primarySourceId, setPrimarySourceId] = useState(stream?.epg_mapping?.primary_epg_source_id || '');
  const [primaryChannelId, setPrimaryChannelId] = useState(stream?.epg_mapping?.primary_channel_id || '');
  const [fallbackSourceId, setFallbackSourceId] = useState(stream?.epg_mapping?.fallback_epg_source_id || '');
  const [fallbackChannelId, setFallbackChannelId] = useState(stream?.epg_mapping?.fallback_channel_id || '');

  // Autocomplete state
  const [primarySearch, setPrimarySearch] = useState('');
  const [primarySearchResults, setPrimarySearchResults] = useState<EPGChannel[]>([]);
  const [fallbackSearch, setFallbackSearch] = useState('');
  const [fallbackSearchResults, setFallbackSearchResults] = useState<EPGChannel[]>([]);

  // Auxiliary data
  const [categories, setCategories] = useState<Category[]>([]);
  const [logos, setLogos] = useState<Logo[]>([]);
  const [epgSources, setEPGSources] = useState<EPGSource[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedUrl, setCopiedUrl] = useState(false);

  useEffect(() => {
    loadAuxData();
  }, []);

  const loadAuxData = async () => {
    try {
      const [cats, lgs, srcs] = await Promise.all([
        api.getCategories(),
        api.getLogos(),
        api.getEPGSources(),
      ]);
      setCategories(cats || []);
      setLogos(lgs || []);
      setEPGSources(srcs || []);
    } catch (err) {
      console.error('Failed to load auxiliary modal data', err);
    }
  };

  // Auto-generate slug from name if new
  useEffect(() => {
    if (!stream && name) {
      const generated = name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
      setSlug(generated);
      if (!tvgId) setTvgId(generated);
      if (!tvgName) setTvgName(name);
    }
  }, [name, stream]);

  // Search EPG channels
  useEffect(() => {
    if (!primarySourceId || primarySearch.trim().length === 0) {
      setPrimarySearchResults([]);
      return;
    }
    const timer = setTimeout(async () => {
      try {
        const results = await api.searchEPGChannels(primarySourceId, primarySearch);
        setPrimarySearchResults(results || []);
      } catch (err) {
        console.error(err);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [primarySourceId, primarySearch]);

  useEffect(() => {
    if (!fallbackSourceId || fallbackSearch.trim().length === 0) {
      setFallbackSearchResults([]);
      return;
    }
    const timer = setTimeout(async () => {
      try {
        const results = await api.searchEPGChannels(fallbackSourceId, fallbackSearch);
        setFallbackSearchResults(results || []);
      } catch (err) {
        console.error(err);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [fallbackSourceId, fallbackSearch]);

  const toggleCategory = (id: string) => {
    setSelectedCategoryIds((prev) =>
      prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id]
    );
  };

  // Auto-generated non-editable playback URL preview
  const serverBase = window.location.origin;
  const currentSlug = slug || 'channel-slug';
  const autoPlaybackUrl = `${serverBase}/stream/${currentSlug}/${currentSlug}.m3u8`;

  const copyPlaybackUrl = () => {
    navigator.clipboard.writeText(autoPlaybackUrl);
    setCopiedUrl(true);
    setTimeout(() => setCopiedUrl(false), 2000);
  };

  const handleQuickCreateCategory = async () => {
    if (!newCategoryName.trim()) return;
    setCreatingCategory(true);
    try {
      const cat = await api.createCategory({ name: newCategoryName.trim(), sort_order: 0 });
      setCategories((prev) => [...prev, cat]);
      setSelectedCategoryIds((prev) => [...prev, cat.id]);
      setNewCategoryName('');
    } catch (err: any) {
      alert(err.message || 'Failed to create category');
    } finally {
      setCreatingCategory(false);
    }
  };

  const handleQuickImportLogoUrl = async () => {
    if (!customLogoUrl.trim()) return;
    try {
      const lg = await api.importLogoUrl({ name: name || 'Channel Logo', url: customLogoUrl.trim() });
      setLogos((prev) => [lg, ...prev]);
      setSelectedLogoId(lg.id);
      setCustomLogoUrl('');
      setLogoTab('library');
    } catch (err: any) {
      alert(err.message || 'Failed to import logo');
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !mediaUrl.trim()) {
      setError('Please provide channel name and media stream source URL');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      let finalLogoId = selectedLogoId;
      let finalLogoUrl = customLogoUrl;

      // If user uploaded a new logo file inline
      if (logoTab === 'upload' && uploadFile) {
        const uploaded = await api.uploadLogo(uploadFile, uploadName || name);
        finalLogoId = uploaded.id;
        finalLogoUrl = uploaded.url;
      } else if (selectedLogoId && !finalLogoUrl) {
        const found = logos.find((l) => l.id === selectedLogoId);
        if (found) {
          finalLogoUrl = found.url;
        }
      }

      const payload: Partial<Stream> = {
        name: name.trim(),
        slug: slug.trim(),
        media_url: mediaUrl.trim(),
        tvg_id: tvgId.trim(),
        tvg_name: tvgName.trim(),
        tvg_chno: tvgChno.trim(),
        mode,
        idle_timeout_sec: Number(idleTimeoutSec) || 180,
        auto_recover: autoRecover,
        recover_timeout_sec: Number(recoverTimeoutSec) || 30,
        enabled,
        buffer_size: bufferSize.trim(),
        fifo_size: fifoSize.trim(),
        program_id: programId.trim(),
        use_gpu: useGpu,
        overrun_nonfatal: overrunNonfatal,
        logo_id: finalLogoId,
        logo_url: finalLogoUrl,
        category_ids: selectedCategoryIds,
        epg_mapping: {
          stream_id: stream?.id || '',
          primary_epg_source_id: primarySourceId,
          primary_channel_id: primaryChannelId,
          fallback_epg_source_id: fallbackSourceId,
          fallback_channel_id: fallbackChannelId,
        },
      };

      if (stream) {
        await api.updateStream(stream.id, payload);
      } else {
        await api.createStream(payload);
      }

      onSave();
    } catch (err: any) {
      setError(err.message || 'Failed to save stream');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in overflow-y-auto">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-3xl w-full my-8 shadow-2xl overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/90">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center border border-indigo-500/20">
              <Film className="w-4 h-4" />
            </div>
            <h3 className="text-lg font-bold text-white">
              {stream ? 'Edit Stream Channel' : 'Add New Stream Channel'}
            </h3>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSave} className="p-6 space-y-6 overflow-y-auto max-h-[75vh]">
          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-red-400 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Section 1: Basic Information */}
          <div className="space-y-4">
            <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-400 flex items-center space-x-2">
              <Film className="w-3.5 h-3.5" />
              <span>Channel Information</span>
            </h4>

            <div className="grid grid-cols-1 sm:grid-cols-12 gap-4">
              {/* Channel Number prominently at top */}
              <div className="sm:col-span-3">
                <label className="block text-xs font-semibold text-slate-300 mb-1.5 flex items-center justify-between">
                  <span>Channel Number</span>
                  <span className="text-[10px] text-indigo-400 font-mono">tvg-chno</span>
                </label>
                <input
                  type="number"
                  min="1"
                  placeholder="e.g. 1"
                  value={tvgChno}
                  onChange={(e) => setTvgChno(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm font-mono font-bold text-white placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
              </div>

              {/* Channel Name */}
              <div className="sm:col-span-5">
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  Channel Name <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. ESPN HD"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
              </div>

              {/* URL Slug */}
              <div className="sm:col-span-4">
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  URL Slug (Identifier)
                </label>
                <input
                  type="text"
                  placeholder="e.g. espn-hd"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                Media Source URL (Input) <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                required
                placeholder="e.g. udp://@239.255.255.250:1234 or rtsp://... or https://.../stream.m3u8"
                value={mediaUrl}
                onChange={(e) => setMediaUrl(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
              />
              <span className="text-[11px] text-slate-500 mt-1 block">
                Supports UDP, RTSP, RTMP, HTTP/HTTPS HLS, or any FFmpeg-compatible source stream.
              </span>
            </div>

            {/* Read-only Auto-Generated Playback URL */}
            <div className="bg-slate-950/70 border border-indigo-500/20 rounded-xl p-3.5">
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-semibold text-indigo-300 flex items-center space-x-1.5">
                  <span>Auto-Generated IPTV Playback URL</span>
                  <span className="text-[10px] bg-indigo-500/20 text-indigo-400 px-1.5 py-0.5 rounded font-mono">
                    Read-Only
                  </span>
                </label>
                <button
                  type="button"
                  onClick={copyPlaybackUrl}
                  className="flex items-center space-x-1 text-xs text-indigo-400 hover:text-indigo-300 transition-colors font-medium"
                >
                  {copiedUrl ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copiedUrl ? 'Copied' : 'Copy URL'}</span>
                </button>
              </div>
              <input
                type="text"
                readOnly
                value={autoPlaybackUrl}
                className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-1.5 text-xs font-mono text-slate-300 select-all focus:outline-none cursor-default"
              />
              <p className="text-[11px] text-slate-500 mt-1">
                This exact playback endpoint is injected into your generated <code className="text-slate-400">/playlist.m3u</code> feed.
              </p>
            </div>
          </div>

          <hr className="border-slate-800" />

          {/* Section 2: Mode & Lifecycle */}
          <div className="space-y-4">
            <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-400 flex items-center space-x-2">
              <Cpu className="w-3.5 h-3.5" />
              <span>Transcoding Mode & Lifecycle</span>
            </h4>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <label className={`flex items-start space-x-3 p-3.5 rounded-xl border cursor-pointer transition-all ${
                mode === 'ondemand'
                  ? 'bg-indigo-600/10 border-indigo-500/50 text-slate-100'
                  : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'
              }`}>
                <input
                  type="radio"
                  name="streamMode"
                  value="ondemand"
                  checked={mode === 'ondemand'}
                  onChange={() => setMode('ondemand')}
                  className="mt-1 text-indigo-600 focus:ring-0"
                />
                <div>
                  <span className="text-sm font-semibold block text-slate-200">On-Demand (Recommended)</span>
                  <span className="text-xs text-slate-400">
                    FFmpeg starts when an IPTV client tunes in and stops automatically when idle.
                  </span>
                </div>
              </label>

              <label className={`flex items-start space-x-3 p-3.5 rounded-xl border cursor-pointer transition-all ${
                mode === 'always_on'
                  ? 'bg-indigo-600/10 border-indigo-500/50 text-slate-100'
                  : 'bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700'
              }`}>
                <input
                  type="radio"
                  name="streamMode"
                  value="always_on"
                  checked={mode === 'always_on'}
                  onChange={() => setMode('always_on')}
                  className="mt-1 text-indigo-600 focus:ring-0"
                />
                <div>
                  <span className="text-sm font-semibold block text-slate-200">Always-On</span>
                  <span className="text-xs text-slate-400">
                    Runs FFmpeg 24/7 in the background with automatic reconnect on failure.
                  </span>
                </div>
              </label>
            </div>

            {mode === 'ondemand' && (
              <div className="max-w-xs">
                <label className="block text-xs font-semibold text-slate-300 mb-1">
                  Inactivity Timeout (seconds)
                </label>
                <input
                  type="number"
                  min="30"
                  max="3600"
                  value={idleTimeoutSec}
                  onChange={(e) => setIdleTimeoutSec(Number(e.target.value))}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-1.5 text-sm text-slate-200 focus:outline-none focus:border-indigo-500"
                />
                <span className="text-[11px] text-slate-500">Stops transcoding after this duration of no client requests.</span>
              </div>
            )}

            {/* Auto-Recovery Watchdog */}
            <div className="bg-slate-950 p-4 rounded-xl border border-slate-800 space-y-3">
              <label className="flex items-center space-x-2.5 text-xs text-slate-200 cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoRecover}
                  onChange={(e) => setAutoRecover(e.target.checked)}
                  className="rounded bg-slate-900 border-slate-700 text-indigo-600 focus:ring-0"
                />
                <span className="font-semibold">Automated Failure Recovery Watchdog (Self-Healing)</span>
              </label>
              {autoRecover && (
                <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400 pl-6">
                  <span>Auto-recover and restart stream if it stays in error for more than:</span>
                  <input
                    type="number"
                    min="5"
                    max="600"
                    value={recoverTimeoutSec}
                    onChange={(e) => setRecoverTimeoutSec(Number(e.target.value))}
                    className="w-20 bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1 text-xs text-slate-200 text-center focus:outline-none focus:border-indigo-500 font-mono"
                  />
                  <span>seconds</span>
                </div>
              )}
            </div>
          </div>

          <hr className="border-slate-800" />

          {/* Section 3: Categories */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-400 flex items-center space-x-2">
                <Layers className="w-3.5 h-3.5" />
                <span>IPTV Categories (Multi-Category Support)</span>
              </h4>

              {/* Seamless Inline Category Creation */}
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  placeholder="+ New Category..."
                  value={newCategoryName}
                  onChange={(e) => setNewCategoryName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      handleQuickCreateCategory();
                    }
                  }}
                  className="bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 w-36"
                />
                <button
                  type="button"
                  onClick={handleQuickCreateCategory}
                  disabled={creatingCategory || !newCategoryName.trim()}
                  className="px-2.5 py-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-semibold disabled:opacity-40"
                >
                  Add
                </button>
              </div>
            </div>

            <p className="text-xs text-slate-400">
              Select one or multiple categories for IPTV group-title mapping:
            </p>

            {categories.length === 0 ? (
              <p className="text-xs text-slate-600 italic">No categories created yet. Type a name above and click Add to create one instantly.</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {categories.map((cat) => {
                  const isSelected = selectedCategoryIds.includes(cat.id);
                  return (
                    <button
                      type="button"
                      key={cat.id}
                      onClick={() => toggleCategory(cat.id)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        isSelected
                          ? 'bg-indigo-600 text-white shadow-md shadow-indigo-500/20'
                          : 'bg-slate-950 border border-slate-800 text-slate-400 hover:border-slate-700'
                      }`}
                    >
                      {cat.name}
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <hr className="border-slate-800" />

          {/* Section 4: Logo Selection */}
          <div className="space-y-3">
            <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-400 flex items-center space-x-2">
              <ImageIcon className="w-3.5 h-3.5" />
              <span>Channel Logo</span>
            </h4>

            <div className="flex space-x-2 border-b border-slate-800 pb-2">
              <button
                type="button"
                onClick={() => setLogoTab('library')}
                className={`px-3 py-1 text-xs rounded-md font-medium ${
                  logoTab === 'library' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Choose from Library
              </button>
              <button
                type="button"
                onClick={() => setLogoTab('upload')}
                className={`px-3 py-1 text-xs rounded-md font-medium ${
                  logoTab === 'upload' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Upload New Logo
              </button>
              <button
                type="button"
                onClick={() => setLogoTab('url')}
                className={`px-3 py-1 text-xs rounded-md font-medium ${
                  logoTab === 'url' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                External Image URL
              </button>
            </div>

            {logoTab === 'library' && (
              <div className="space-y-2">
                {logos.length === 0 ? (
                  <p className="text-xs text-slate-600 italic">No saved logos in the library yet. Upload one via the Upload tab.</p>
                ) : (
                  <div className="grid grid-cols-4 sm:grid-cols-6 gap-3 max-h-40 overflow-y-auto p-1">
                    {logos.map((lg) => {
                      const isSelected = selectedLogoId === lg.id;
                      return (
                        <div
                          key={lg.id}
                          onClick={() => {
                            if (isSelected) {
                              setSelectedLogoId('');
                              setCustomLogoUrl('');
                            } else {
                              setSelectedLogoId(lg.id);
                              setCustomLogoUrl(lg.url);
                            }
                          }}
                          className={`flex flex-col items-center p-2 rounded-xl border cursor-pointer transition-all ${
                            isSelected
                              ? 'border-indigo-500 bg-indigo-500/10 ring-2 ring-indigo-500/30'
                              : 'border-slate-800 bg-slate-950 hover:border-slate-700'
                          }`}
                        >
                          <img src={lg.url} alt={lg.name} className="w-10 h-10 object-contain mb-1 rounded" />
                          <span className="text-[10px] text-slate-300 truncate w-full text-center">{lg.name}</span>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}

            {logoTab === 'upload' && (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 bg-slate-950 p-4 rounded-xl border border-slate-800">
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Logo Name</label>
                  <input
                    type="text"
                    placeholder="e.g. ESPN Logo"
                    value={uploadName}
                    onChange={(e) => setUploadName(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-200"
                  />
                </div>
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Select File (PNG/JPG/SVG/WebP)</label>
                  <input
                    type="file"
                    accept="image/*"
                    onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
                    className="w-full text-xs text-slate-400 file:mr-2 file:py-1 file:px-3 file:rounded-md file:border-0 file:text-xs file:bg-indigo-600 file:text-white hover:file:bg-indigo-500"
                  />
                </div>
              </div>
            )}

            {logoTab === 'url' && (
              <div className="space-y-2">
                <input
                  type="text"
                  placeholder="https://example.com/logo.png"
                  value={customLogoUrl}
                  onChange={(e) => {
                    setCustomLogoUrl(e.target.value);
                    setSelectedLogoId('');
                  }}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
                {customLogoUrl.startsWith('http') && (
                  <button
                    type="button"
                    onClick={handleQuickImportLogoUrl}
                    className="text-xs text-indigo-400 hover:text-indigo-300 font-medium flex items-center space-x-1"
                  >
                    <span>Download and save to local logo library</span>
                  </button>
                )}
              </div>
            )}
          </div>

          <hr className="border-slate-800" />

          {/* Section 5: EPG Guide & Fallback Mapping */}
          <div className="space-y-4">
            <h4 className="text-xs font-bold uppercase tracking-wider text-indigo-400 flex items-center space-x-2">
              <Calendar className="w-3.5 h-3.5" />
              <span>EPG Guide Mapping & Fallback</span>
            </h4>

            {/* Primary EPG */}
            <div className="bg-slate-950 p-4 rounded-xl border border-slate-800 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-slate-200">Primary EPG Source</span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <select
                    value={primarySourceId}
                    onChange={(e) => {
                      setPrimarySourceId(e.target.value);
                      setPrimaryChannelId('');
                      setPrimarySearch('');
                    }}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200"
                  >
                    <option value="">-- Select EPG Source --</option>
                    {epgSources.map((s) => (
                      <option key={s.id} value={s.id}>{s.name} ({s.channel_count} channels)</option>
                    ))}
                  </select>
                </div>

                <div className="relative">
                  <input
                    type="text"
                    disabled={!primarySourceId}
                    placeholder="Search channel in EPG..."
                    value={primarySearch}
                    onChange={(e) => setPrimarySearch(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 disabled:opacity-50"
                  />
                  {primarySearchResults.length > 0 && (
                    <div className="absolute top-full left-0 right-0 z-30 mt-1 bg-slate-900 border border-slate-800 rounded-lg shadow-xl max-h-48 overflow-y-auto">
                      {primarySearchResults.map((ch) => (
                        <div
                          key={ch.id}
                          onClick={() => {
                            setPrimaryChannelId(ch.channel_id);
                            setPrimarySearch(`${ch.display_name} (${ch.channel_id})`);
                            setPrimarySearchResults([]);
                            if (!tvgId || tvgId === slug) {
                              setTvgId(ch.channel_id);
                            }
                            if (!tvgName || tvgName === name) {
                              setTvgName(ch.display_name || name);
                            }
                          }}
                          className="px-3 py-2 hover:bg-slate-800 cursor-pointer text-xs border-b border-slate-800/50 flex justify-between items-center"
                        >
                          <span className="text-slate-200">{ch.display_name}</span>
                          <span className="text-slate-500 font-mono text-[10px]">{ch.channel_id}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              {primaryChannelId && (
                <div className="text-[11px] text-indigo-400 font-mono bg-indigo-500/10 px-2.5 py-1 rounded border border-indigo-500/20">
                  Mapped Primary Channel ID: <strong>{primaryChannelId}</strong>
                </div>
              )}
            </div>

            {/* Fallback EPG */}
            <div className="bg-slate-950 p-4 rounded-xl border border-slate-800 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-slate-200">Fallback EPG Source (Optional)</span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <select
                    value={fallbackSourceId}
                    onChange={(e) => {
                      setFallbackSourceId(e.target.value);
                      setFallbackChannelId('');
                      setFallbackSearch('');
                    }}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200"
                  >
                    <option value="">-- Select Fallback EPG Source --</option>
                    {epgSources.map((s) => (
                      <option key={s.id} value={s.id}>{s.name} ({s.channel_count} channels)</option>
                    ))}
                  </select>
                </div>

                <div className="relative">
                  <input
                    type="text"
                    disabled={!fallbackSourceId}
                    placeholder="Search fallback channel..."
                    value={fallbackSearch}
                    onChange={(e) => setFallbackSearch(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 disabled:opacity-50"
                  />
                  {fallbackSearchResults.length > 0 && (
                    <div className="absolute top-full left-0 right-0 z-30 mt-1 bg-slate-900 border border-slate-800 rounded-lg shadow-xl max-h-48 overflow-y-auto">
                      {fallbackSearchResults.map((ch) => (
                        <div
                          key={ch.id}
                          onClick={() => {
                            setFallbackChannelId(ch.channel_id);
                            setFallbackSearch(`${ch.display_name} (${ch.channel_id})`);
                            setFallbackSearchResults([]);
                          }}
                          className="px-3 py-2 hover:bg-slate-800 cursor-pointer text-xs border-b border-slate-800/50 flex justify-between items-center"
                        >
                          <span className="text-slate-200">{ch.display_name}</span>
                          <span className="text-slate-500 font-mono text-[10px]">{ch.channel_id}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              {fallbackChannelId && (
                <div className="text-[11px] text-purple-400 font-mono bg-purple-500/10 px-2.5 py-1 rounded border border-purple-500/20">
                  Mapped Fallback Channel ID: <strong>{fallbackChannelId}</strong>
                </div>
              )}
            </div>

            {/* Optional TVG / XMLTV Overrides */}
            <div className="bg-slate-950/70 p-4 rounded-xl border border-slate-800/80 space-y-3">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                <span className="text-xs font-semibold text-slate-300 flex items-center space-x-2">
                  <span>XMLTV & TVG Metadata Overrides</span>
                  <span className="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">Optional</span>
                </span>
                <span className="text-[11px] text-slate-500">Auto-inherited from EPG mapping & channel name</span>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
                <div>
                  <label className="block text-[11px] text-slate-400 mb-1 font-medium">
                    TVG ID (XMLTV Identifier)
                  </label>
                  <input
                    type="text"
                    placeholder={primaryChannelId || slug || 'Auto-inherited from EPG mapping or slug'}
                    value={tvgId}
                    onChange={(e) => setTvgId(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3 py-1.5 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                  />
                  <span className="text-[10px] text-slate-500 mt-1 block">
                    Matches <code className="text-slate-400">&lt;channel id="..."&gt;</code> in XMLTV guide. Defaults to mapped EPG channel or slug.
                  </span>
                </div>

                <div>
                  <label className="block text-[11px] text-slate-400 mb-1 font-medium">
                    TVG Display Name (tvg-name)
                  </label>
                  <input
                    type="text"
                    placeholder={name || 'Auto-inherited from channel name'}
                    value={tvgName}
                    onChange={(e) => setTvgName(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-xl px-3 py-1.5 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                  />
                  <span className="text-[10px] text-slate-500 mt-1 block">
                    Secondary display name for IPTV players. Defaults to channel name.
                  </span>
                </div>
              </div>
            </div>
          </div>

          <hr className="border-slate-800" />

          {/* Section 6: Advanced FFmpeg settings */}
          <div>
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-xs font-semibold text-slate-400 hover:text-slate-200 flex items-center space-x-1"
            >
              <span>{showAdvanced ? 'Hide Advanced FFmpeg Parameters' : 'Show Advanced FFmpeg Parameters'}</span>
            </button>

            {showAdvanced && (
              <div className="mt-3 grid grid-cols-1 sm:grid-cols-3 gap-3 bg-slate-950 p-4 rounded-xl border border-slate-800">
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Buffer Size</label>
                  <input
                    type="text"
                    value={bufferSize}
                    onChange={(e) => setBufferSize(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-200"
                  />
                </div>
                <div>
                  <label className="block text-xs text-slate-400 mb-1">FIFO Size</label>
                  <input
                    type="text"
                    value={fifoSize}
                    onChange={(e) => setFifoSize(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-200"
                  />
                </div>
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Program ID</label>
                  <input
                    type="text"
                    value={programId}
                    onChange={(e) => setProgramId(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-200"
                  />
                </div>
                <div className="sm:col-span-3 flex items-center space-x-6 pt-2">
                  <label className="flex items-center space-x-2 text-xs text-slate-300 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={useGpu}
                      onChange={(e) => setUseGpu(e.target.checked)}
                      className="rounded bg-slate-900 border-slate-800 text-indigo-600 focus:ring-0"
                    />
                    <span>Enable CUDA GPU Acceleration (-hwaccel cuda)</span>
                  </label>
                  <label className="flex items-center space-x-2 text-xs text-slate-300 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={overrunNonfatal}
                      onChange={(e) => setOverrunNonfatal(e.target.checked)}
                      className="rounded bg-slate-900 border-slate-800 text-indigo-600 focus:ring-0"
                    />
                    <span>Overrun Non-Fatal</span>
                  </label>
                </div>
              </div>
            )}
          </div>

          <div className="flex items-center justify-between pt-4 border-t border-slate-800">
            <label className="flex items-center space-x-2 text-xs text-slate-300 cursor-pointer">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
                className="rounded bg-slate-900 border-slate-800 text-indigo-600 focus:ring-0"
              />
              <span className="font-medium">Channel Enabled in IPTV Playlist</span>
            </label>

            <div className="flex items-center space-x-3">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl text-xs font-semibold transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={saving}
                className="px-5 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-xl text-xs font-semibold shadow-lg shadow-indigo-600/20 transition-all"
              >
                {saving ? 'Saving...' : stream ? 'Update Channel' : 'Create Channel'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
};
