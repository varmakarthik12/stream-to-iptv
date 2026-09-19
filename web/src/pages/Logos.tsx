import React, { useState, useEffect } from 'react';
import { Image as ImageIcon, Upload, Globe, Copy, Check, Trash2, Plus, AlertCircle } from 'lucide-react';
import { api, Logo } from '../api';
import { copyToClipboard } from '../utils/clipboard';

export const Logos: React.FC = () => {
  const [logos, setLogos] = useState<Logo[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'upload' | 'import'>('upload');

  // Upload state
  const [uploadName, setUploadName] = useState('');
  const [uploadFile, setUploadFile] = useState<File | null>(null);

  // Import state
  const [importName, setImportName] = useState('');
  const [importUrl, setImportUrl] = useState('');

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  useEffect(() => {
    loadLogos();
  }, []);

  const loadLogos = async () => {
    setLoading(true);
    try {
      const data = await api.getLogos();
      setLogos(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!uploadFile) return;

    setSubmitting(true);
    setError(null);
    try {
      await api.uploadLogo(uploadFile, uploadName || uploadFile.name);
      setUploadName('');
      setUploadFile(null);
      loadLogos();
    } catch (err: any) {
      setError(err.message || 'Upload failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!importUrl) return;

    setSubmitting(true);
    setError(null);
    try {
      await api.importLogoUrl({
        name: importName || 'Imported Logo',
        url: importUrl,
      });
      setImportName('');
      setImportUrl('');
      loadLogos();
    } catch (err: any) {
      setError(err.message || 'Import failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (l: Logo) => {
    if (!confirm(`Delete logo "${l.name}"?`)) return;
    try {
      await api.deleteLogo(l.id);
      loadLogos();
    } catch (err: any) {
      alert(err.message || 'Failed to delete logo');
    }
  };

  const copyLogoUrl = async (l: Logo) => {
    await copyToClipboard(l.url);
    setCopiedId(l.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center space-x-2">
            <ImageIcon className="w-6 h-6 text-indigo-400" />
            <span>Logo Library</span>
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Store, name, and reuse channel logos stored locally inside the application folder.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-10">
        {/* Logo Upload / Import Card */}
        <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl h-fit">
          <div className="flex space-x-2 border-b border-slate-800 pb-3 mb-4">
            <button
              onClick={() => setTab('upload')}
              className={`flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
                tab === 'upload' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Upload className="w-3.5 h-3.5" />
              <span>Upload Local Logo</span>
            </button>
            <button
              onClick={() => setTab('import')}
              className={`flex items-center space-x-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
                tab === 'import' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <Globe className="w-3.5 h-3.5" />
              <span>Import from URL</span>
            </button>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-red-400 text-xs">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {tab === 'upload' ? (
            <form onSubmit={handleUpload} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  Logo Name
                </label>
                <input
                  type="text"
                  placeholder="e.g. HBO Max HD"
                  value={uploadName}
                  onChange={(e) => setUploadName(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  Image File (PNG, JPG, SVG, WebP) <span className="text-red-400">*</span>
                </label>
                <input
                  type="file"
                  required
                  accept="image/*"
                  onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
                  className="w-full text-xs text-slate-400 file:mr-3 file:py-2 file:px-4 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-indigo-600 file:text-white hover:file:bg-indigo-500 cursor-pointer"
                />
              </div>

              <button
                type="submit"
                disabled={submitting || !uploadFile}
                className="w-full mt-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-2 px-4 rounded-xl text-xs shadow-lg shadow-indigo-600/20 transition-all flex items-center justify-center space-x-2"
              >
                <Upload className="w-4 h-4" />
                <span>{submitting ? 'Uploading...' : 'Save to Library'}</span>
              </button>
            </form>
          ) : (
            <form onSubmit={handleImport} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  Logo Name
                </label>
                <input
                  type="text"
                  placeholder="e.g. ESPN Logo"
                  value={importName}
                  onChange={(e) => setImportName(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                  Image URL <span className="text-red-400">*</span>
                </label>
                <input
                  type="url"
                  required
                  placeholder="https://example.com/logo.png"
                  value={importUrl}
                  onChange={(e) => setImportUrl(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500"
                />
                <span className="text-[11px] text-slate-500 mt-1 block">
                  The image will be downloaded and saved to your local storage folder.
                </span>
              </div>

              <button
                type="submit"
                disabled={submitting || !importUrl}
                className="w-full mt-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-2 px-4 rounded-xl text-xs shadow-lg shadow-indigo-600/20 transition-all flex items-center justify-center space-x-2"
              >
                <Globe className="w-4 h-4" />
                <span>{submitting ? 'Downloading...' : 'Download & Save Logo'}</span>
              </button>
            </form>
          )}
        </div>

        {/* Logos Grid */}
        <div className="lg:col-span-2">
          <div className="bg-slate-900 border border-slate-800 rounded-3xl p-6 shadow-xl">
            <div className="flex items-center justify-between mb-4 border-b border-slate-800 pb-3">
              <span className="text-xs font-semibold text-slate-400">Library: {logos.length} logos saved</span>
            </div>

            {loading ? (
              <div className="text-center py-20 text-slate-500 text-sm">Loading logos...</div>
            ) : logos.length === 0 ? (
              <div className="text-center py-20 text-slate-500 text-sm italic">
                No logos saved in the library yet. Upload or import one using the form on the left.
              </div>
            ) : (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                {logos.map((lg) => (
                  <div
                    key={lg.id}
                    className="bg-slate-950 border border-slate-800 rounded-2xl p-4 flex flex-col items-center justify-between group hover:border-slate-700 transition-all"
                  >
                    <div className="w-20 h-20 flex items-center justify-center p-2 mb-3 bg-slate-900 rounded-xl border border-slate-800/80">
                      <img src={lg.url} alt={lg.name} className="max-w-full max-h-full object-contain" />
                    </div>

                    <div className="text-center w-full mb-3">
                      <h4 className="text-xs font-semibold text-slate-200 truncate w-full" title={lg.name}>
                        {lg.name}
                      </h4>
                      <span className="text-[10px] text-slate-500 block truncate">
                        {lg.is_local ? 'Stored locally' : 'External'}
                      </span>
                    </div>

                    <div className="flex items-center space-x-2 w-full pt-2 border-t border-slate-800/60 justify-center">
                      <button
                        onClick={() => copyLogoUrl(lg)}
                        title="Copy served URL"
                        className="p-1.5 rounded-lg bg-slate-900 hover:bg-slate-800 text-slate-400 hover:text-slate-200 border border-slate-800"
                      >
                        {copiedId === lg.id ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>

                      <button
                        onClick={() => handleDelete(lg)}
                        title="Delete logo"
                        className="p-1.5 rounded-lg bg-slate-900 hover:bg-red-500/10 text-slate-400 hover:text-red-400 border border-slate-800"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
