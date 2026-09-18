import React, { useEffect, useRef, useState } from 'react';
import Hls from 'hls.js';
import { X, Play, Pause, Volume2, VolumeX, Maximize, RotateCcw, AlertCircle, Loader2, Tv } from 'lucide-react';
import { Stream } from '../api';

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

interface StreamPlayerModalProps {
  stream: Stream;
  onClose: () => void;
}

const isAppleMobileDevice = (): boolean => {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') return false;
  const ua = navigator.userAgent || '';
  const isIOS = /iPad|iPhone|iPod/.test(ua);
  const isIPadOS =
    (navigator.platform === 'MacIntel' || /Macintosh/.test(ua)) &&
    (navigator.maxTouchPoints > 1 || (navigator as any).standalone !== undefined);
  const isIOSBrowser = /CriOS|FxiOS|EdgiOS/.test(ua);
  return isIOS || isIPadOS || isIOSBrowser;
};

export const StreamPlayerModal: React.FC<StreamPlayerModalProps> = ({ stream, onClose }) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const cleanupNativeRef = useRef<(() => void) | null>(null);

  const [isPlaying, setIsPlaying] = useState<boolean>(true);
  const [isMuted, setIsMuted] = useState<boolean>(false);
  const [volume, setVolume] = useState<number>(1);
  const [resolution, setResolution] = useState<string>('Detecting...');
  const [bufferLen, setBufferLen] = useState<number>(0);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isStarting, setIsStarting] = useState<boolean>(false);
  const [playerEngine, setPlayerEngine] = useState<string>('Auto');

  const playbackUrl =
    stream.playback_url ||
    `${window.location.origin}/stream/${stream.slug}/${stream.slug}.m3u8`;

  const setupNativePlayer = (video: HTMLVideoElement, url: string): (() => void) => {
    video.src = url;
    video.load();

    const onLoadedMetadata = () => {
      setIsStarting(false);
      setErrorMsg(null);
      if (video.videoWidth && video.videoHeight) {
        setResolution(`${video.videoWidth}x${video.videoHeight}`);
      }
      video.play().catch(() => {
        // Mobile / iOS autoplay policy requires mute first
        video.muted = true;
        setIsMuted(true);
        video.play().catch((err) => {
          console.warn('Muted autoplay also failed:', err);
        });
      });
    };

    const onPlaying = () => {
      setIsStarting(false);
      setIsPlaying(true);
      setErrorMsg(null);
    };

    const onError = () => {
      const err = video.error;
      if (err && err.code !== MediaError.MEDIA_ERR_ABORTED) {
        let msg = 'Failed to load live video stream.';
        if (err.code === MediaError.MEDIA_ERR_NETWORK) {
          msg = 'Network error while loading live stream. Retrying...';
        } else if (err.code === MediaError.MEDIA_ERR_DECODE) {
          msg = 'Video decoding error on live stream.';
        } else if (err.code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED) {
          msg = 'Stream format not supported by browser.';
        }
        setErrorMsg(msg);
        setIsStarting(false);
      }
    };

    video.addEventListener('loadedmetadata', onLoadedMetadata);
    video.addEventListener('playing', onPlaying);
    video.addEventListener('error', onError);

    return () => {
      video.removeEventListener('loadedmetadata', onLoadedMetadata);
      video.removeEventListener('playing', onPlaying);
      video.removeEventListener('error', onError);
      video.removeAttribute('src');
      video.load();
    };
  };

  const setupHlsPlayer = (video: HTMLVideoElement, url: string) => {
    let mediaRecoveryAttempts = 0;
    let lastMediaRecoveryTime = 0;

    const hls = new Hls({
      enableWorker: true,
      lowLatencyMode: false,
      backBufferLength: 30,
      maxBufferLength: 15,
      maxMaxBufferLength: 30,
      liveSyncDurationCount: 3,
      liveMaxLatencyDurationCount: 8,
      liveDurationInfinity: true,
    });

    hlsRef.current = hls;

    hls.loadSource(url);
    hls.attachMedia(video);

    hls.on(Hls.Events.MANIFEST_PARSED, () => {
      setIsStarting(false);
      setErrorMsg(null);
      video.play().catch(() => {
        video.muted = true;
        setIsMuted(true);
        video.play().catch(() => {});
      });
    });

    hls.on(Hls.Events.LEVEL_LOADED, (_event, data) => {
      setIsStarting(false);
      if (data.details.totalduration) {
        if (video.buffered.length > 0) {
          const end = video.buffered.end(video.buffered.length - 1);
          setBufferLen(Math.max(0, Number((end - video.currentTime).toFixed(1))));
        }
      }
    });

    hls.on(Hls.Events.FRAG_BUFFERED, () => {
      setErrorMsg(null);
      mediaRecoveryAttempts = 0;
    });

    hls.on(Hls.Events.FRAG_CHANGED, () => {
      if (video.videoWidth && video.videoHeight) {
        setResolution(`${video.videoWidth}x${video.videoHeight}`);
      }
    });

    hls.on(Hls.Events.ERROR, (_event, data) => {
      if (data.fatal) {
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            setErrorMsg('Network error encountered while loading live stream. Retrying...');
            hls.startLoad();
            break;
          case Hls.ErrorTypes.MEDIA_ERROR: {
            const now = Date.now();
            if (now - lastMediaRecoveryTime > 15000) {
              mediaRecoveryAttempts = 0;
            }
            lastMediaRecoveryTime = now;
            mediaRecoveryAttempts++;

            if (mediaRecoveryAttempts === 1) {
              setErrorMsg('Media error detected. Recovering buffer...');
              hls.recoverMediaError();
            } else if (mediaRecoveryAttempts === 2) {
              setErrorMsg('Media sync error detected. Adjusting audio codec...');
              hls.swapAudioCodec();
              hls.recoverMediaError();
            } else {
              // If native HLS is supported, fallback to native
              if (video.canPlayType('application/vnd.apple.mpegurl')) {
                setErrorMsg('Switching to native browser player...');
                hls.destroy();
                hlsRef.current = null;
                setPlayerEngine('Native (Fallback)');
                cleanupNativeRef.current = setupNativePlayer(video, url);
              } else {
                setErrorMsg('Fatal media error: unable to recover video buffer.');
                hls.destroy();
              }
            }
            break;
          }
          default:
            setErrorMsg(`Streaming error: ${data.details || 'unknown'}`);
            hls.destroy();
            break;
        }
      }
    });
  };

  const initPlayer = () => {
    setErrorMsg(null);
    setIsStarting(true);

    if (cleanupNativeRef.current) {
      cleanupNativeRef.current();
      cleanupNativeRef.current = null;
    }

    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }

    const video = videoRef.current;
    if (!video) return;

    // Apple mobile devices (iPad, iPhone across Chrome, Safari, Firefox) provide native
    // AVFoundation HLS playback which handles MPEG-TS seamlessly without MSE buffer errors.
    const canPlayNative = Boolean(video.canPlayType('application/vnd.apple.mpegurl'));
    const preferNative = isAppleMobileDevice() && canPlayNative;

    if (preferNative) {
      setPlayerEngine('Native (Apple)');
      cleanupNativeRef.current = setupNativePlayer(video, playbackUrl);
    } else if (Hls.isSupported()) {
      setPlayerEngine('HLS.js');
      setupHlsPlayer(video, playbackUrl);
    } else if (canPlayNative) {
      setPlayerEngine('Native');
      cleanupNativeRef.current = setupNativePlayer(video, playbackUrl);
    } else {
      setIsStarting(false);
      setErrorMsg('Your browser does not support HLS video playback.');
    }
  };

  useEffect(() => {
    initPlayer();

    const interval = setInterval(() => {
      const video = videoRef.current;
      if (video && video.buffered.length > 0) {
        const end = video.buffered.end(video.buffered.length - 1);
        setBufferLen(Math.max(0, Number((end - video.currentTime).toFixed(1))));
        if (video.videoWidth && video.videoHeight) {
          setResolution(`${video.videoWidth}x${video.videoHeight}`);
        }
      }
    }, 1000);

    return () => {
      clearInterval(interval);
      if (cleanupNativeRef.current) {
        cleanupNativeRef.current();
        cleanupNativeRef.current = null;
      }
      if (hlsRef.current) {
        hlsRef.current.destroy();
        hlsRef.current = null;
      }
    };
  }, [playbackUrl]);

  const togglePlay = () => {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      video.play();
      setIsPlaying(true);
    } else {
      video.pause();
      setIsPlaying(false);
    }
  };

  const toggleMute = () => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
    setIsMuted(video.muted);
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseFloat(e.target.value);
    setVolume(val);
    const video = videoRef.current;
    if (video) {
      video.volume = val;
      video.muted = val === 0;
      setIsMuted(val === 0);
    }
  };

  const toggleFullscreen = () => {
    const video = videoRef.current;
    if (!video) return;
    if (!document.fullscreenElement) {
      if (video.requestFullscreen) {
        video.requestFullscreen().catch(() => {});
      } else if ((video as any).webkitEnterFullscreen) {
        (video as any).webkitEnterFullscreen();
      }
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen().catch(() => {});
      }
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-4xl overflow-hidden shadow-2xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-900/80">
          <div className="flex items-center space-x-3">
            <div className="w-9 h-9 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-center overflow-hidden shrink-0">
              <ChannelLogo url={stream.logo_url} name={stream.name} />
            </div>
            <div>
              <div className="flex items-center space-x-2">
                <h3 className="text-base font-bold text-white leading-none">{stream.name}</h3>
                {stream.tvg_chno && (
                  <span className="text-[11px] font-mono font-bold px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                    #{stream.tvg_chno}
                  </span>
                )}
                <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center space-x-1">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                  <span>LIVE</span>
                </span>
              </div>
              <p className="text-xs font-mono text-slate-400 mt-0.5 truncate max-w-md">
                {playbackUrl}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-2 rounded-lg hover:bg-slate-800 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Video Canvas */}
        <div className="relative bg-black aspect-video flex items-center justify-center overflow-hidden">
          <video
            ref={videoRef}
            className="w-full h-full object-contain"
            playsInline
            webkit-playsinline="true"
            x5-playsinline="true"
            autoPlay
            onPlay={() => {
              setIsPlaying(true);
              setIsStarting(false);
              setErrorMsg(null);
            }}
            onPlaying={() => {
              setIsPlaying(true);
              setIsStarting(false);
              setErrorMsg(null);
            }}
            onPause={() => setIsPlaying(false)}
          />

          {isStarting && (
            <div className="absolute inset-0 bg-black/60 flex flex-col items-center justify-center space-y-3">
              <Loader2 className="w-10 h-10 text-indigo-400 animate-spin" />
              <div className="text-center">
                <p className="text-sm font-semibold text-white">Starting Live Stream...</p>
                <p className="text-xs text-slate-400 mt-1">Connecting to FFmpeg pipeline and buffering live segments</p>
              </div>
            </div>
          )}

          {errorMsg && (
            <div className="absolute top-4 left-4 right-4 bg-red-950/90 border border-red-800/80 rounded-xl p-3 flex items-start space-x-3 text-red-200">
              <AlertCircle className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
              <div className="text-xs">
                <span className="font-semibold">{errorMsg}</span>
              </div>
            </div>
          )}
        </div>

        {/* Bottom Bar & Stats */}
        <div className="px-6 py-3.5 bg-slate-950/90 border-t border-slate-800 flex flex-wrap items-center justify-between gap-4">
          {/* Controls */}
          <div className="flex items-center space-x-3">
            <button
              onClick={togglePlay}
              className="p-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition-colors"
              title={isPlaying ? 'Pause' : 'Play'}
            >
              {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            </button>

            <button
              onClick={initPlayer}
              className="p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors"
              title="Reload Stream"
            >
              <RotateCcw className="w-4 h-4" />
            </button>

            <div className="flex items-center space-x-2 pl-2 border-l border-slate-800">
              <button
                onClick={toggleMute}
                className="p-1.5 rounded-md text-slate-400 hover:text-white transition-colors"
              >
                {isMuted || volume === 0 ? (
                  <VolumeX className="w-4 h-4 text-red-400" />
                ) : (
                  <Volume2 className="w-4 h-4" />
                )}
              </button>
              <input
                type="range"
                min="0"
                max="1"
                step="0.05"
                value={isMuted ? 0 : volume}
                onChange={handleVolumeChange}
                className="w-20 h-1 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-indigo-500"
              />
            </div>
          </div>

          {/* Real-time Diagnostics */}
          <div className="flex items-center space-x-4 text-xs font-mono text-slate-400">
            <div>
              <span className="text-slate-500 text-[10px] uppercase">Engine:</span>{' '}
              <span className="text-slate-300">{playerEngine}</span>
            </div>
            <div>
              <span className="text-slate-500 text-[10px] uppercase">Resolution:</span>{' '}
              <span className="text-slate-300">{resolution}</span>
            </div>
            <div>
              <span className="text-slate-500 text-[10px] uppercase">Buffer:</span>{' '}
              <span className="text-slate-300">{bufferLen}s</span>
            </div>

            <button
              onClick={toggleFullscreen}
              className="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
              title="Toggle Fullscreen"
            >
              <Maximize className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StreamPlayerModal;
