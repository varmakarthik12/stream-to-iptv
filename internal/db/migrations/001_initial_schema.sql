-- 001_initial_schema.sql

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS logos (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    file_name TEXT,
    url TEXT NOT NULL,
    is_local INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS epg_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    refresh_interval_hours INTEGER DEFAULT 24,
    last_refreshed_at DATETIME,
    status TEXT DEFAULT 'pending',
    status_message TEXT DEFAULT '',
    channel_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS epg_channels (
    id TEXT PRIMARY KEY,
    epg_source_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    icon_url TEXT DEFAULT '',
    FOREIGN KEY(epg_source_id) REFERENCES epg_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_epg_channels_source ON epg_channels(epg_source_id);
CREATE INDEX IF NOT EXISTS idx_epg_channels_lookup ON epg_channels(epg_source_id, channel_id);

CREATE TABLE IF NOT EXISTS streams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    tvg_id TEXT DEFAULT '',
    tvg_name TEXT DEFAULT '',
    tvg_chno TEXT DEFAULT '',
    media_url TEXT NOT NULL,
    logo_id TEXT DEFAULT '',
    logo_url TEXT DEFAULT '',
    program_id TEXT DEFAULT '1',
    mode TEXT DEFAULT 'ondemand',
    idle_timeout_sec INTEGER DEFAULT 180,
    buffer_size TEXT DEFAULT '1000000',
    fifo_size TEXT DEFAULT '',
    use_gpu INTEGER DEFAULT 0,
    overrun_nonfatal INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stream_categories (
    stream_id TEXT NOT NULL,
    category_id TEXT NOT NULL,
    PRIMARY KEY (stream_id, category_id),
    FOREIGN KEY(stream_id) REFERENCES streams(id) ON DELETE CASCADE,
    FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stream_epg_mappings (
    stream_id TEXT PRIMARY KEY,
    primary_epg_source_id TEXT DEFAULT '',
    primary_channel_id TEXT DEFAULT '',
    fallback_epg_source_id TEXT DEFAULT '',
    fallback_channel_id TEXT DEFAULT '',
    FOREIGN KEY(stream_id) REFERENCES streams(id) ON DELETE CASCADE
);
