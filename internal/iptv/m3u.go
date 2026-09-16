package iptv

import (
	"fmt"
	"net/url"
	"strings"

	"stream-to-iptv/internal/repository"
)

type M3UService struct {
	repo *repository.Repository
}

func NewM3UService(repo *repository.Repository) *M3UService {
	return &M3UService{repo: repo}
}

// GeneratePlaylist builds the M3U content for enabled channels
func (s *M3UService) GeneratePlaylist(host string, scheme string, queryToken string) (string, error) {
	streams, err := s.repo.GetAllStreams()
	if err != nil {
		return "", err
	}

	settings, _ := s.repo.GetAllSettings()
	iptvToken := settings["iptv_token"]

	// If token authentication is required
	if iptvToken != "" && iptvToken != queryToken {
		return "", fmt.Errorf("unauthorized: valid token required")
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	if customURL := settings["server_url"]; customURL != "" {
		baseURL = strings.TrimRight(customURL, "/")
	}

	epgURL := fmt.Sprintf("%s/epg.xml", baseURL)
	if iptvToken != "" {
		epgURL = fmt.Sprintf("%s?token=%s", epgURL, url.QueryEscape(iptvToken))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#EXTM3U x-tvg-url=\"%s\"\n\n", epgURL))

	for _, st := range streams {
		if !st.Enabled {
			continue
		}

		// Channel name & TVG metadata
		name := st.Name
		tvgID := st.TVGId
		if tvgID == "" {
			if st.EPGMapping != nil && st.EPGMapping.PrimaryChannelID != "" {
				tvgID = st.EPGMapping.PrimaryChannelID
			} else {
				tvgID = st.Slug
			}
		}
		tvgName := st.TVGName
		if tvgName == "" {
			tvgName = st.Name
		}
		tvgChno := st.TVGChno

		// Logo resolution
		logoURL := st.LogoURL
		if st.LogoID != "" {
			logo, err := s.repo.GetLogoByID(st.LogoID)
			if err == nil && logo != nil {
				if logo.IsLocal {
					logoURL = fmt.Sprintf("%s/logos/%s", baseURL, logo.FileName)
				} else {
					logoURL = logo.URL
				}
			}
		}

		// Categories
		var categoryNames []string
		for _, cat := range st.Categories {
			categoryNames = append(categoryNames, cat.Name)
		}
		groupTitle := strings.Join(categoryNames, ";")
		if groupTitle == "" {
			groupTitle = "General"
		}

		// Playback URL
		streamURL := fmt.Sprintf("%s/stream/%s/%s.m3u8", baseURL, st.Slug, st.Slug)
		if iptvToken != "" {
			streamURL = fmt.Sprintf("%s?token=%s", streamURL, url.QueryEscape(iptvToken))
		}

		// Build EXTINF line
		var extinf strings.Builder
		extinf.WriteString(fmt.Sprintf("#EXTINF:-1 tvg-id=\"%s\" tvg-name=\"%s\"", escapeQuotes(tvgID), escapeQuotes(tvgName)))

		if logoURL != "" {
			extinf.WriteString(fmt.Sprintf(" tvg-logo=\"%s\"", escapeQuotes(logoURL)))
		}
		if tvgChno != "" {
			extinf.WriteString(fmt.Sprintf(" tvg-chno=\"%s\"", escapeQuotes(tvgChno)))
		}
		extinf.WriteString(fmt.Sprintf(" group-title=\"%s\",%s\n", escapeQuotes(groupTitle), name))
		extinf.WriteString(streamURL)
		extinf.WriteString("\n\n")

		sb.WriteString(extinf.String())
	}

	return sb.String(), nil
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `'`)
}

// GetPlaybackURL constructs the full URL for a stream
func GetPlaybackURL(baseURL string, slug string, token string) string {
	u := fmt.Sprintf("%s/stream/%s/%s.m3u8", strings.TrimRight(baseURL, "/"), slug, slug)
	if token != "" {
		u = fmt.Sprintf("%s?token=%s", u, url.QueryEscape(token))
	}
	return u
}
