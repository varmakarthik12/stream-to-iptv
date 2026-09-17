package repository

import (
	"database/sql"
	"stream-to-iptv/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// User Operations
func (r *Repository) CountUsers() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (r *Repository) CreateUser(username, passwordHash string) (*models.User, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.db.Exec("INSERT INTO users (id, username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, username, passwordHash, now, now)
	if err != nil {
		return nil, err
	}
	return &models.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow("SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow("SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Settings Operations
func (r *Repository) GetAllSettings() (map[string]string, error) {
	rows, err := r.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, nil
}

func (r *Repository) GetSetting(key, defaultValue string) (string, error) {
	var val string
	err := r.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return val, nil
}

func (r *Repository) SetSetting(key, value string) error {
	_, err := r.db.Exec(`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`, key, value)
	return err
}

// Category Operations
func (r *Repository) GetAllCategories() ([]models.Category, error) {
	rows, err := r.db.Query(`SELECT c.id, c.name, c.slug, c.sort_order, c.created_at, COUNT(sc.stream_id) as stream_count
		FROM categories c
		LEFT JOIN stream_categories sc ON c.id = sc.category_id
		GROUP BY c.id
		ORDER BY c.sort_order ASC, c.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt, &c.StreamCount); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repository) CreateCategory(name, slug string, sortOrder int) (*models.Category, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.db.Exec("INSERT INTO categories (id, name, slug, sort_order, created_at) VALUES (?, ?, ?, ?, ?)",
		id, name, slug, sortOrder, now)
	if err != nil {
		return nil, err
	}
	return &models.Category{
		ID:        id,
		Name:      name,
		Slug:      slug,
		SortOrder: sortOrder,
		CreatedAt: now,
	}, nil
}

func (r *Repository) UpdateCategory(id, name, slug string, sortOrder int) error {
	_, err := r.db.Exec("UPDATE categories SET name = ?, slug = ?, sort_order = ? WHERE id = ?",
		name, slug, sortOrder, id)
	return err
}

func (r *Repository) DeleteCategory(id string) error {
	_, err := r.db.Exec("DELETE FROM categories WHERE id = ?", id)
	return err
}

// Logo Operations
func (r *Repository) GetAllLogos() ([]models.Logo, error) {
	rows, err := r.db.Query("SELECT id, name, file_name, url, is_local, created_at FROM logos ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Logo
	for rows.Next() {
		var l models.Logo
		var isLocalInt int
		if err := rows.Scan(&l.ID, &l.Name, &l.FileName, &l.URL, &isLocalInt, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.IsLocal = isLocalInt == 1
		list = append(list, l)
	}
	return list, nil
}

func (r *Repository) CreateLogo(name, fileName, url string, isLocal bool) (*models.Logo, error) {
	id := uuid.New().String()
	now := time.Now()
	isLocalInt := 0
	if isLocal {
		isLocalInt = 1
	}
	_, err := r.db.Exec("INSERT INTO logos (id, name, file_name, url, is_local, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, name, fileName, url, isLocalInt, now)
	if err != nil {
		return nil, err
	}
	return &models.Logo{
		ID:        id,
		Name:      name,
		FileName:  fileName,
		URL:       url,
		IsLocal:   isLocal,
		CreatedAt: now,
	}, nil
}

func (r *Repository) GetLogoByID(id string) (*models.Logo, error) {
	var l models.Logo
	var isLocalInt int
	err := r.db.QueryRow("SELECT id, name, file_name, url, is_local, created_at FROM logos WHERE id = ?", id).
		Scan(&l.ID, &l.Name, &l.FileName, &l.URL, &isLocalInt, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	l.IsLocal = isLocalInt == 1
	return &l, nil
}

func (r *Repository) DeleteLogo(id string) error {
	_, err := r.db.Exec("DELETE FROM logos WHERE id = ?", id)
	return err
}

// EPG Sources Operations
func (r *Repository) GetAllEPGSources() ([]models.EPGSource, error) {
	rows, err := r.db.Query(`SELECT id, name, url, refresh_interval_hours, last_refreshed_at, status, status_message, channel_count, created_at
		FROM epg_sources ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.EPGSource
	for rows.Next() {
		var s models.EPGSource
		var lastRefreshed sql.NullTime
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.RefreshIntervalHours, &lastRefreshed, &s.Status, &s.StatusMessage, &s.ChannelCount, &s.CreatedAt); err != nil {
			return nil, err
		}
		if lastRefreshed.Valid {
			s.LastRefreshedAt = &lastRefreshed.Time
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *Repository) GetEPGSourceByID(id string) (*models.EPGSource, error) {
	var s models.EPGSource
	var lastRefreshed sql.NullTime
	err := r.db.QueryRow(`SELECT id, name, url, refresh_interval_hours, last_refreshed_at, status, status_message, channel_count, created_at
		FROM epg_sources WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.URL, &s.RefreshIntervalHours, &lastRefreshed, &s.Status, &s.StatusMessage, &s.ChannelCount, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	if lastRefreshed.Valid {
		s.LastRefreshedAt = &lastRefreshed.Time
	}
	return &s, nil
}

func (r *Repository) CreateEPGSource(name, url string, refreshIntervalHours int) (*models.EPGSource, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.db.Exec(`INSERT INTO epg_sources (id, name, url, refresh_interval_hours, status, created_at)
		VALUES (?, ?, ?, ?, 'pending', ?)`, id, name, url, refreshIntervalHours, now)
	if err != nil {
		return nil, err
	}
	return &models.EPGSource{
		ID:                   id,
		Name:                 name,
		URL:                  url,
		RefreshIntervalHours: refreshIntervalHours,
		Status:               "pending",
		CreatedAt:            now,
	}, nil
}

func (r *Repository) UpdateEPGSource(id, name, url string, refreshIntervalHours int) error {
	_, err := r.db.Exec(`UPDATE epg_sources SET name = ?, url = ?, refresh_interval_hours = ? WHERE id = ?`,
		name, url, refreshIntervalHours, id)
	return err
}

func (r *Repository) UpdateEPGSourceStatus(id, status, statusMessage string, channelCount int) error {
	now := time.Now()
	_, err := r.db.Exec(`UPDATE epg_sources SET status = ?, status_message = ?, channel_count = ?, last_refreshed_at = ? WHERE id = ?`,
		status, statusMessage, channelCount, now, id)
	return err
}

func (r *Repository) DeleteEPGSource(id string) error {
	_, err := r.db.Exec("DELETE FROM epg_sources WHERE id = ?", id)
	return err
}

// EPG Channels Operations
func (r *Repository) SaveEPGChannels(sourceID string, channels []models.EPGChannel) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM epg_channels WHERE epg_source_id = ?", sourceID); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO epg_channels (id, epg_source_id, channel_id, display_name, icon_url) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ch := range channels {
		id := uuid.New().String()
		if _, err := stmt.Exec(id, sourceID, ch.ChannelID, ch.DisplayName, ch.IconURL); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) SearchEPGChannels(sourceID, query string, limit int) ([]models.EPGChannel, error) {
	if limit <= 0 {
		limit = 50
	}
	sqlQuery := `SELECT id, epg_source_id, channel_id, display_name, icon_url FROM epg_channels WHERE 1=1`
	var args []interface{}

	if sourceID != "" {
		sqlQuery += " AND epg_source_id = ?"
		args = append(args, sourceID)
	}

	if query != "" {
		sqlQuery += " AND (display_name LIKE ? OR channel_id LIKE ?)"
		pattern := "%" + query + "%"
		args = append(args, pattern, pattern)
	}

	sqlQuery += " ORDER BY display_name ASC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.EPGChannel
	for rows.Next() {
		var c models.EPGChannel
		if err := rows.Scan(&c.ID, &c.EPGSourceID, &c.ChannelID, &c.DisplayName, &c.IconURL); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// Stream Operations
func (r *Repository) GetAllStreams() ([]models.Stream, error) {
	rows, err := r.db.Query(`SELECT id, name, slug, tvg_id, tvg_name, tvg_chno, media_url, logo_id, logo_url,
		program_id, COALESCE(local_addr, ''), mode, idle_timeout_sec, buffer_size, fifo_size, use_gpu, overrun_nonfatal, enabled,
		COALESCE(auto_recover, 1), COALESCE(recover_timeout_sec, 30), created_at, updated_at
		FROM streams ORDER BY 
			CASE WHEN tvg_chno IS NOT NULL AND tvg_chno != '' AND CAST(tvg_chno AS INTEGER) > 0 
				THEN CAST(tvg_chno AS INTEGER) 
				ELSE 999999 
			END ASC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Stream
	for rows.Next() {
		var s models.Stream
		var useGPUInt, overrunInt, enabledInt, autoRecoverInt int
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.TVGId, &s.TVGName, &s.TVGChno, &s.MediaURL, &s.LogoID, &s.LogoURL,
			&s.ProgramID, &s.LocalAddr, &s.Mode, &s.IdleTimeoutSec, &s.BufferSize, &s.FifoSize, &useGPUInt, &overrunInt, &enabledInt,
			&autoRecoverInt, &s.RecoverTimeoutSec, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.UseGPU = useGPUInt == 1
		s.OverrunNonfatal = overrunInt == 1
		s.Enabled = enabledInt == 1
		s.AutoRecover = autoRecoverInt == 1

		// Fetch Categories
		categories, err := r.GetStreamCategories(s.ID)
		if err == nil {
			s.Categories = categories
			for _, cat := range categories {
				s.CategoryIDs = append(s.CategoryIDs, cat.ID)
			}
		}

		// Fetch EPG Mapping
		mapping, err := r.GetStreamEPGMapping(s.ID)
		if err == nil {
			s.EPGMapping = mapping
		}

		// Resolve logo URL from logo library if empty
		if s.LogoID != "" && s.LogoURL == "" {
			if logo, err := r.GetLogoByID(s.LogoID); err == nil && logo != nil {
				s.LogoURL = logo.URL
			}
		}

		list = append(list, s)
	}
	return list, nil
}

func (r *Repository) GetStreamByID(id string) (*models.Stream, error) {
	var s models.Stream
	var useGPUInt, overrunInt, enabledInt, autoRecoverInt int
	err := r.db.QueryRow(`SELECT id, name, slug, tvg_id, tvg_name, tvg_chno, media_url, logo_id, logo_url,
		program_id, COALESCE(local_addr, ''), mode, idle_timeout_sec, buffer_size, fifo_size, use_gpu, overrun_nonfatal, enabled,
		COALESCE(auto_recover, 1), COALESCE(recover_timeout_sec, 30), created_at, updated_at
		FROM streams WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Slug, &s.TVGId, &s.TVGName, &s.TVGChno, &s.MediaURL, &s.LogoID, &s.LogoURL,
			&s.ProgramID, &s.LocalAddr, &s.Mode, &s.IdleTimeoutSec, &s.BufferSize, &s.FifoSize, &useGPUInt, &overrunInt, &enabledInt,
			&autoRecoverInt, &s.RecoverTimeoutSec, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.UseGPU = useGPUInt == 1
	s.OverrunNonfatal = overrunInt == 1
	s.Enabled = enabledInt == 1
	s.AutoRecover = autoRecoverInt == 1

	categories, _ := r.GetStreamCategories(s.ID)
	s.Categories = categories
	for _, cat := range categories {
		s.CategoryIDs = append(s.CategoryIDs, cat.ID)
	}

	mapping, _ := r.GetStreamEPGMapping(s.ID)
	s.EPGMapping = mapping

	if s.LogoID != "" && s.LogoURL == "" {
		if logo, err := r.GetLogoByID(s.LogoID); err == nil && logo != nil {
			s.LogoURL = logo.URL
		}
	}

	return &s, nil
}

func (r *Repository) GetStreamBySlug(slug string) (*models.Stream, error) {
	var s models.Stream
	var useGPUInt, overrunInt, enabledInt, autoRecoverInt int
	err := r.db.QueryRow(`SELECT id, name, slug, tvg_id, tvg_name, tvg_chno, media_url, logo_id, logo_url,
		program_id, COALESCE(local_addr, ''), mode, idle_timeout_sec, buffer_size, fifo_size, use_gpu, overrun_nonfatal, enabled,
		COALESCE(auto_recover, 1), COALESCE(recover_timeout_sec, 30), created_at, updated_at
		FROM streams WHERE slug = ?`, slug).
		Scan(&s.ID, &s.Name, &s.Slug, &s.TVGId, &s.TVGName, &s.TVGChno, &s.MediaURL, &s.LogoID, &s.LogoURL,
			&s.ProgramID, &s.LocalAddr, &s.Mode, &s.IdleTimeoutSec, &s.BufferSize, &s.FifoSize, &useGPUInt, &overrunInt, &enabledInt,
			&autoRecoverInt, &s.RecoverTimeoutSec, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.UseGPU = useGPUInt == 1
	s.OverrunNonfatal = overrunInt == 1
	s.Enabled = enabledInt == 1
	s.AutoRecover = autoRecoverInt == 1

	categories, _ := r.GetStreamCategories(s.ID)
	s.Categories = categories

	mapping, _ := r.GetStreamEPGMapping(s.ID)
	s.EPGMapping = mapping

	if s.LogoID != "" && s.LogoURL == "" {
		if logo, err := r.GetLogoByID(s.LogoID); err == nil && logo != nil {
			s.LogoURL = logo.URL
		}
	}

	return &s, nil
}

func (r *Repository) CreateStream(s *models.Stream) (*models.Stream, error) {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	useGPUInt := 0
	if s.UseGPU {
		useGPUInt = 1
	}
	overrunInt := 0
	if s.OverrunNonfatal {
		overrunInt = 1
	}
	enabledInt := 0
	if s.Enabled {
		enabledInt = 1
	}
	autoRecoverInt := 0
	if s.AutoRecover {
		autoRecoverInt = 1
	}
	if s.RecoverTimeoutSec <= 0 {
		s.RecoverTimeoutSec = 30
	}

	_, err := r.db.Exec(`INSERT INTO streams (id, name, slug, tvg_id, tvg_name, tvg_chno, media_url, logo_id, logo_url,
		program_id, local_addr, mode, idle_timeout_sec, buffer_size, fifo_size, use_gpu, overrun_nonfatal, enabled,
		auto_recover, recover_timeout_sec, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Slug, s.TVGId, s.TVGName, s.TVGChno, s.MediaURL, s.LogoID, s.LogoURL,
		s.ProgramID, s.LocalAddr, s.Mode, s.IdleTimeoutSec, s.BufferSize, s.FifoSize, useGPUInt, overrunInt, enabledInt,
		autoRecoverInt, s.RecoverTimeoutSec, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if len(s.CategoryIDs) > 0 {
		_ = r.SetStreamCategories(s.ID, s.CategoryIDs)
	}

	if s.EPGMapping != nil {
		_ = r.SetStreamEPGMapping(s.ID, s.EPGMapping)
	}

	return s, nil
}

func (r *Repository) UpdateStream(s *models.Stream) error {
	now := time.Now()
	s.UpdatedAt = now

	useGPUInt := 0
	if s.UseGPU {
		useGPUInt = 1
	}
	overrunInt := 0
	if s.OverrunNonfatal {
		overrunInt = 1
	}
	enabledInt := 0
	if s.Enabled {
		enabledInt = 1
	}
	autoRecoverInt := 0
	if s.AutoRecover {
		autoRecoverInt = 1
	}
	if s.RecoverTimeoutSec <= 0 {
		s.RecoverTimeoutSec = 30
	}

	_, err := r.db.Exec(`UPDATE streams SET name = ?, slug = ?, tvg_id = ?, tvg_name = ?, tvg_chno = ?, media_url = ?,
		logo_id = ?, logo_url = ?, program_id = ?, local_addr = ?, mode = ?, idle_timeout_sec = ?, buffer_size = ?, fifo_size = ?,
		use_gpu = ?, overrun_nonfatal = ?, enabled = ?, auto_recover = ?, recover_timeout_sec = ?, updated_at = ? WHERE id = ?`,
		s.Name, s.Slug, s.TVGId, s.TVGName, s.TVGChno, s.MediaURL, s.LogoID, s.LogoURL,
		s.ProgramID, s.LocalAddr, s.Mode, s.IdleTimeoutSec, s.BufferSize, s.FifoSize, useGPUInt, overrunInt, enabledInt,
		autoRecoverInt, s.RecoverTimeoutSec, s.UpdatedAt, s.ID)
	if err != nil {
		return err
	}

	_ = r.SetStreamCategories(s.ID, s.CategoryIDs)
	if s.EPGMapping != nil {
		_ = r.SetStreamEPGMapping(s.ID, s.EPGMapping)
	}

	return nil
}

func (r *Repository) DeleteStream(id string) error {
	_, err := r.db.Exec("DELETE FROM streams WHERE id = ?", id)
	return err
}

func (r *Repository) GetStreamCategories(streamID string) ([]models.Category, error) {
	rows, err := r.db.Query(`SELECT c.id, c.name, c.slug, c.sort_order, c.created_at
		FROM categories c
		INNER JOIN stream_categories sc ON c.id = sc.category_id
		WHERE sc.stream_id = ?
		ORDER BY c.sort_order ASC`, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repository) SetStreamCategories(streamID string, categoryIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM stream_categories WHERE stream_id = ?", streamID); err != nil {
		return err
	}

	if len(categoryIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO stream_categories (stream_id, category_id) VALUES (?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, catID := range categoryIDs {
			catID = strings.TrimSpace(catID)
			if catID != "" {
				if _, err := stmt.Exec(streamID, catID); err != nil {
					return err
				}
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) GetStreamEPGMapping(streamID string) (*models.StreamEPGMapping, error) {
	var m models.StreamEPGMapping
	err := r.db.QueryRow(`SELECT stream_id, primary_epg_source_id, primary_channel_id, fallback_epg_source_id, fallback_channel_id
		FROM stream_epg_mappings WHERE stream_id = ?`, streamID).
		Scan(&m.StreamID, &m.PrimaryEPGSourceID, &m.PrimaryChannelID, &m.FallbackEPGSourceID, &m.FallbackChannelID)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) SetStreamEPGMapping(streamID string, mapping *models.StreamEPGMapping) error {
	if mapping == nil {
		_, err := r.db.Exec("DELETE FROM stream_epg_mappings WHERE stream_id = ?", streamID)
		return err
	}

	_, err := r.db.Exec(`INSERT INTO stream_epg_mappings (stream_id, primary_epg_source_id, primary_channel_id, fallback_epg_source_id, fallback_channel_id)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(stream_id) DO UPDATE SET
			primary_epg_source_id = excluded.primary_epg_source_id,
			primary_channel_id = excluded.primary_channel_id,
			fallback_epg_source_id = excluded.fallback_epg_source_id,
			fallback_channel_id = excluded.fallback_channel_id`,
		streamID, mapping.PrimaryEPGSourceID, mapping.PrimaryChannelID, mapping.FallbackEPGSourceID, mapping.FallbackChannelID)
	return err
}

func (r *Repository) CountStreams() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM streams").Scan(&count)
	return count, err
}
