// web/src/api.ts

export interface User {
  id: string;
  username: string;
}

export interface SetupStatus {
  setup_required: boolean;
  has_config_to_import: boolean;
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  sort_order: number;
  stream_count?: number;
}

export interface Logo {
  id: string;
  name: string;
  file_name: string;
  url: string;
  is_local: boolean;
}

export interface EPGSource {
  id: string;
  name: string;
  url: string;
  refresh_interval_hours: number;
  last_refreshed_at?: string;
  status: 'pending' | 'updating' | 'ok' | 'error';
  status_message: string;
  channel_count: number;
}

export interface EPGChannel {
  id: string;
  epg_source_id: string;
  channel_id: string;
  display_name: string;
  icon_url: string;
}

export interface StreamEPGMapping {
  stream_id: string;
  primary_epg_source_id: string;
  primary_channel_id: string;
  fallback_epg_source_id: string;
  fallback_channel_id: string;
}

export interface Stream {
  id: string;
  name: string;
  slug: string;
  tvg_id: string;
  tvg_name: string;
  tvg_chno: string;
  media_url: string;
  logo_id: string;
  logo_url: string;
  program_id: string;
  local_addr?: string;
  analyze_duration?: string;
  probe_size?: string;
  mode: 'ondemand' | 'always_on';
  idle_timeout_sec: number;
  auto_recover?: boolean;
  recover_timeout_sec?: number;
  buffer_size: string;
  fifo_size: string;
  use_gpu: boolean;
  overrun_nonfatal: boolean;
  enabled: boolean;
  categories?: Category[];
  category_ids?: string[];
  epg_mapping?: StreamEPGMapping;
  status?: 'idle' | 'starting' | 'running' | 'error' | 'stopped';
  playback_url?: string;
}

export interface StreamLogEntry {
  timestamp: string;
  message: string;
}

export interface SystemStatus {
  local_ips: string[];
  base_url: string;
  playlist_url: string;
  epg_url: string;
  stream_count: number;
  active_streams_count: number;
  problem_streams_count: number;
  problem_streams: Array<{
    id: string;
    name: string;
    slug: string;
    status: string;
    auto_recover: boolean;
    recover_timeout_sec: number;
  }>;
  data_dir: string;
  ffmpeg_path?: string;
  cpu_cores: number;
  goroutines: number;
  memory_alloc_mb: number;
  memory_sys_mb: number;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || `Request failed with status ${res.status}`);
  }

  const contentType = res.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    return res.json();
  }
  return {} as T;
}

export const api = {
  // Auth & Setup
  getSetupStatus: () => request<SetupStatus>('/api/setup/status'),
  setup: (data: { username: string; password: string; import_existing: boolean }) =>
    request<{ success: boolean; user: User; token: string }>('/api/setup', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  login: (data: { username: string; password: string }) =>
    request<{ success: boolean; token: string; user: User }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  logout: () => request<{ success: boolean }>('/api/auth/logout', { method: 'POST' }),
  getMe: () => request<User>('/api/auth/me'),

  // Streams
  getStreams: () => request<Stream[]>('/api/streams'),
  getStream: (id: string) => request<Stream>(`/api/streams/${id}`),
  createStream: (data: Partial<Stream>) =>
    request<Stream>('/api/streams', { method: 'POST', body: JSON.stringify(data) }),
  updateStream: (id: string, data: Partial<Stream>) =>
    request<Stream>(`/api/streams/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteStream: (id: string) =>
    request<{ success: boolean }>(`/api/streams/${id}`, { method: 'DELETE' }),
  startStream: (id: string) =>
    request<{ status: string }>(`/api/streams/${id}/start`, { method: 'POST' }),
  stopStream: (id: string) =>
    request<{ status: string }>(`/api/streams/${id}/stop`, { method: 'POST' }),
  getStreamLogs: (id: string) =>
    request<StreamLogEntry[]>(`/api/streams/${id}/logs`),

  // Categories
  getCategories: () => request<Category[]>('/api/categories'),
  createCategory: (data: { name: string; sort_order: number }) =>
    request<Category>('/api/categories', { method: 'POST', body: JSON.stringify(data) }),
  updateCategory: (id: string, data: { name: string; sort_order: number }) =>
    request<{ success: boolean }>(`/api/categories/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteCategory: (id: string) =>
    request<{ success: boolean }>(`/api/categories/${id}`, { method: 'DELETE' }),

  // Logos
  getLogos: () => request<Logo[]>('/api/logos'),
  uploadLogo: async (file: File, name: string): Promise<Logo> => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('name', name);
    const res = await fetch('/api/logos/upload', {
      method: 'POST',
      body: formData,
      credentials: 'include',
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
  importLogoUrl: (data: { name: string; url: string }) =>
    request<Logo>('/api/logos/import-url', { method: 'POST', body: JSON.stringify(data) }),
  deleteLogo: (id: string) =>
    request<{ success: boolean }>(`/api/logos/${id}`, { method: 'DELETE' }),

  // EPG
  getEPGSources: () => request<EPGSource[]>('/api/epg-sources'),
  createEPGSource: (data: { name: string; url: string; refresh_interval_hours: number }) =>
    request<EPGSource>('/api/epg-sources', { method: 'POST', body: JSON.stringify(data) }),
  bulkCreateEPGSources: (sources: { name: string; url: string; refresh_interval_hours?: number }[]) =>
    request<EPGSource[]>('/api/epg-sources/bulk', { method: 'POST', body: JSON.stringify(sources) }),
  updateEPGSource: (id: string, data: { name: string; url: string; refresh_interval_hours: number }) =>
    request<{ success: boolean }>(`/api/epg-sources/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteEPGSource: (id: string) =>
    request<{ success: boolean }>(`/api/epg-sources/${id}`, { method: 'DELETE' }),
  refreshEPGSource: (id: string) =>
    request<{ status: string }>(`/api/epg-sources/${id}/refresh`, { method: 'POST' }),
  searchEPGChannels: (sourceId: string, q: string) =>
    request<EPGChannel[]>(`/api/epg-sources/channels?source_id=${encodeURIComponent(sourceId)}&q=${encodeURIComponent(q)}`),

  // Settings & System
  getSettings: () => request<Record<string, string>>('/api/settings'),
  updateSettings: (data: Record<string, string>) =>
    request<{ success: boolean }>('/api/settings', { method: 'PUT', body: JSON.stringify(data) }),
  getSystemStatus: () => request<SystemStatus>('/api/system/status'),
};
