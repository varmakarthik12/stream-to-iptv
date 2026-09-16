import React, { useState, useEffect } from 'react';
import { Copy, Check, Info, Radio, ExternalLink, HelpCircle } from 'lucide-react';
import { api, SystemStatus } from '../api';

export const IPTVBanner: React.FC = () => {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [copiedM3U, setCopiedM3U] = useState(false);
  const [copiedEPG, setCopiedEPG] = useState(false);
  const [showHelp, setShowHelp] = useState(false);

  useEffect(() => {
    loadStatus();
  }, []);

  const loadStatus = async () => {
    try {
      const data = await api.getSystemStatus();
      setStatus(data);
    } catch (err) {
      console.error('Failed to load system status', err);
    }
  };

  const copyToClipboard = (text: string, type: 'm3u' | 'epg') => {
    navigator.clipboard.writeText(text);
    if (type === 'm3u') {
      setCopiedM3U(true);
      setTimeout(() => setCopiedM3U(false), 2000);
    } else {
      setCopiedEPG(true);
      setTimeout(() => setCopiedEPG(false), 2000);
    }
  };

  if (!status) return null;

  return (
    <div className="bg-gradient-to-r from-slate-900 via-indigo-950/40 to-slate-900 border-b border-indigo-500/20 py-4 px-4 sm:px-6 lg:px-8 mb-6">
      <div className="max-w-7xl mx-auto flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4">
        <div>
          <div className="flex items-center space-x-2">
            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
              IPTV Feeds Active
            </span>
            <span className="text-xs text-slate-400">
              {status.stream_count} channels configured
            </span>
          </div>
          <p className="text-sm font-semibold text-slate-200 mt-1">
            Connect your IPTV player using the auto-generated feeds below
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3 w-full lg:w-auto">
          {/* M3U Playlist Pill */}
          <div className="flex items-center bg-slate-800/90 border border-slate-700/80 rounded-lg p-1 pl-3 text-xs w-full sm:w-auto">
            <span className="font-semibold text-indigo-400 mr-2 uppercase tracking-wide">M3U Playlist:</span>
            <span className="text-slate-300 truncate max-w-[220px] font-mono mr-2" title={status.playlist_url}>
              {status.playlist_url}
            </span>
            <button
              onClick={() => copyToClipboard(status.playlist_url, 'm3u')}
              className="flex items-center space-x-1 bg-indigo-600 hover:bg-indigo-500 text-white px-2.5 py-1.5 rounded-md font-medium transition-colors"
            >
              {copiedM3U ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedM3U ? 'Copied' : 'Copy'}</span>
            </button>
          </div>

          {/* EPG URL Pill */}
          <div className="flex items-center bg-slate-800/90 border border-slate-700/80 rounded-lg p-1 pl-3 text-xs w-full sm:w-auto">
            <span className="font-semibold text-purple-400 mr-2 uppercase tracking-wide">XMLTV EPG:</span>
            <span className="text-slate-300 truncate max-w-[220px] font-mono mr-2" title={status.epg_url}>
              {status.epg_url}
            </span>
            <button
              onClick={() => copyToClipboard(status.epg_url, 'epg')}
              className="flex items-center space-x-1 bg-purple-600 hover:bg-purple-500 text-white px-2.5 py-1.5 rounded-md font-medium transition-colors"
            >
              {copiedEPG ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedEPG ? 'Copied' : 'Copy'}</span>
            </button>
          </div>

          {/* Help Button */}
          <button
            onClick={() => setShowHelp(true)}
            title="IPTV Player Setup Guide"
            className="p-2 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg border border-slate-700/60 transition-colors"
          >
            <HelpCircle className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Setup Guide Modal */}
      {showHelp && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-lg w-full p-6 shadow-2xl relative">
            <h3 className="text-lg font-bold text-white mb-2 flex items-center space-x-2">
              <Radio className="w-5 h-5 text-indigo-400" />
              <span>IPTV Player Setup Guide</span>
            </h3>
            <p className="text-xs text-slate-400 mb-4">
              Add your generated M3U playlist and EPG XML to your preferred IPTV player:
            </p>

            <div className="space-y-3 text-xs text-slate-300">
              <div className="p-3 bg-slate-800/60 rounded-lg border border-slate-700/50">
                <h4 className="font-semibold text-indigo-300 mb-1">TiviMate / OTT Navigator</h4>
                <ol className="list-decimal list-inside space-y-1 text-slate-400">
                  <li>Go to <strong>Settings &gt; Playlists &gt; Add Playlist</strong>.</li>
                  <li>Select <strong>M3U Playlist</strong> and paste the M3U Playlist URL.</li>
                  <li>In <strong>TV Guide (EPG)</strong> settings, add the XMLTV EPG URL.</li>
                </ol>
              </div>

              <div className="p-3 bg-slate-800/60 rounded-lg border border-slate-700/50">
                <h4 className="font-semibold text-indigo-300 mb-1">VLC Media Player</h4>
                <p className="text-slate-400">
                  Click <strong>Media &gt; Open Network Stream</strong> (Ctrl+N), paste the M3U Playlist URL, and click Play.
                </p>
              </div>

              <div className="p-3 bg-slate-800/60 rounded-lg border border-slate-700/50">
                <h4 className="font-semibold text-indigo-300 mb-1">Kodi (IPTV Simple Client)</h4>
                <p className="text-slate-400">
                  In PVR IPTV Simple Client configure <strong>M3U Play List URL</strong> and <strong>XMLTV URL</strong>.
                </p>
              </div>
            </div>

            <div className="mt-6 flex justify-end">
              <button
                onClick={() => setShowHelp(false)}
                className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-semibold"
              >
                Close Guide
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
