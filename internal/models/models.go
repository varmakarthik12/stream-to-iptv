package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	SortOrder   int       `json:"sort_order"`
	StreamCount int       `json:"stream_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Logo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FileName  string    `json:"file_name"`
	URL       string    `json:"url"`
	IsLocal   bool      `json:"is_local"`
	CreatedAt time.Time `json:"created_at"`
}

type EPGSource struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	URL                  string     `json:"url"`
	RefreshIntervalHours int        `json:"refresh_interval_hours"`
	LastRefreshedAt      *time.Time `json:"last_refreshed_at"`
	Status               string     `json:"status"` // pending, updating, ok, error
	StatusMessage        string     `json:"status_message"`
	ChannelCount         int        `json:"channel_count"`
	CreatedAt            time.Time  `json:"created_at"`
}

type EPGChannel struct {
	ID          string `json:"id"`
	EPGSourceID string `json:"epg_source_id"`
	ChannelID   string `json:"channel_id"`
	DisplayName string `json:"display_name"`
	IconURL     string `json:"icon_url"`
}

type StreamEPGMapping struct {
	StreamID             string `json:"stream_id"`
	PrimaryEPGSourceID   string `json:"primary_epg_source_id"`
	PrimaryChannelID     string `json:"primary_channel_id"`
	FallbackEPGSourceID  string `json:"fallback_epg_source_id"`
	FallbackChannelID    string `json:"fallback_channel_id"`
}

type Stream struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Slug             string            `json:"slug"`
	TVGId            string            `json:"tvg_id"`
	TVGName          string            `json:"tvg_name"`
	TVGChno          string            `json:"tvg_chno"`
	MediaURL         string            `json:"media_url"`
	LogoID           string            `json:"logo_id"`
	LogoURL          string            `json:"logo_url"`
	ProgramID        string            `json:"program_id"`
	LocalAddr        string            `json:"local_addr"`
	Mode             string            `json:"mode"` // ondemand, always_on
	IdleTimeoutSec   int               `json:"idle_timeout_sec"`
	AutoRecover      bool              `json:"auto_recover"`
	RecoverTimeoutSec int              `json:"recover_timeout_sec"`
	BufferSize       string            `json:"buffer_size"`
	FifoSize         string            `json:"fifo_size"`
	UseGPU           bool              `json:"use_gpu"`
	OverrunNonfatal  bool              `json:"overrun_nonfatal"`
	Enabled          bool              `json:"enabled"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CategoryIDs      []string          `json:"category_ids,omitempty"`
	Categories       []Category        `json:"categories,omitempty"`
	EPGMapping       *StreamEPGMapping `json:"epg_mapping,omitempty"`
	Status           string            `json:"status,omitempty"` // idle, running, starting, error, stopped
	PlaybackURL      string            `json:"playback_url,omitempty"`
}

type StreamLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type SetupRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	ImportExisting bool   `json:"import_existing"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
