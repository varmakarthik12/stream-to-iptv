import React, { useState, useEffect, useRef } from 'react';
import { Terminal, RefreshCw, X, Copy, Check } from 'lucide-react';
import { api, Stream, StreamLogEntry } from '../api';

interface StreamLogsModalProps {
  stream: Stream;
  onClose: () => void;
}

export const StreamLogsModal: React.FC<StreamLogsModalProps> = ({ stream, onClose }) => {
  const [logs, setLogs] = useState<StreamLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [copied, setCopied] = useState(false);
  const logsEndRef = useRef<HTMLDivElement>(null);

  const fetchLogs = async () => {
    try {
      const data = await api.getStreamLogs(stream.id);
      setLogs(data || []);
      setLoading(false);
    } catch (err) {
      console.error('Failed to fetch logs', err);
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
    if (!autoRefresh) return;
    const interval = setInterval(fetchLogs, 2000);
    return () => clearInterval(interval);
  }, [stream.id, autoRefresh]);

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  const copyLogs = () => {
    const text = logs.map((l) => `[${new Date(l.timestamp).toLocaleTimeString()}] ${l.message}`).join('\n');
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-4xl w-full flex flex-col h-[650px] shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/90">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center border border-indigo-500/20">
              <Terminal className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-100 flex items-center space-x-2">
                <span>FFmpeg Logs: {stream.name}</span>
                <span className="text-xs px-2 py-0.5 rounded font-mono bg-slate-800 text-slate-400">
                  {stream.slug}
                </span>
              </h3>
              <p className="text-xs text-slate-400">Status: <span className="uppercase font-semibold text-indigo-400">{stream.status || 'idle'}</span></p>
            </div>
          </div>

          <div className="flex items-center space-x-3">
            <label className="flex items-center space-x-2 text-xs text-slate-400 cursor-pointer">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="rounded bg-slate-800 border-slate-700 text-indigo-600 focus:ring-0"
              />
              <span>Auto-refresh (2s)</span>
            </label>

            <button
              onClick={fetchLogs}
              title="Refresh now"
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-slate-700/60"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>

            <button
              onClick={copyLogs}
              className="flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-xs bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition-colors"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>

            <button
              onClick={onClose}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Log Viewer Content */}
        <div className="flex-1 bg-slate-950 p-4 font-mono text-xs overflow-y-auto leading-relaxed text-slate-300 space-y-1">
          {logs.length === 0 ? (
            <div className="text-center py-20 text-slate-600 italic">
              No logs recorded yet for this stream. Start the stream or request its URL to view live output.
            </div>
          ) : (
            logs.map((log, idx) => (
              <div key={idx} className="flex space-x-3 hover:bg-slate-900/60 px-2 py-0.5 rounded">
                <span className="text-slate-600 select-none">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className={log.message.includes('ERROR') || log.message.includes('error') ? 'text-red-400 font-semibold' : ''}>
                  {log.message}
                </span>
              </div>
            ))
          )}
          <div ref={logsEndRef} />
        </div>

        {/* Footer */}
        <div className="px-6 py-3 border-t border-slate-800 bg-slate-900/90 text-xs text-slate-500 flex justify-between items-center">
          <span>Displaying last {logs.length} process messages</span>
          <button
            onClick={onClose}
            className="px-4 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg font-medium"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
