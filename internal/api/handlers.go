package api

import (
	"fmt"
	"strings"
	"time"

	"stream-to-iptv/internal/iptv"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"
	"stream-to-iptv/internal/stream"
)

type APIHandler struct {
	repo       *repository.Repository
	streamMgr  *stream.Manager
	epgService *iptv.EPGService
	m3uService *iptv.M3UService
}

func NewAPIHandler(
	repo *repository.Repository,
	streamMgr *stream.Manager,
	epgService *iptv.EPGService,
	m3uService *iptv.M3UService,
) *APIHandler {
	return &APIHandler{
		repo:       repo,
		streamMgr:  streamMgr,
		epgService: epgService,
		m3uService: m3uService,
	}
}

// resolveStreamLogo formats and attaches the full logo URL for a stream
func (h *APIHandler) resolveStreamLogo(st *models.Stream, baseURL string) {
	if st.LogoID != "" {
		if logo, err := h.repo.GetLogoByID(st.LogoID); err == nil && logo != nil {
			if logo.IsLocal {
				st.LogoURL = fmt.Sprintf("%s/logos/%s", baseURL, logo.FileName)
			} else {
				st.LogoURL = logo.URL
			}
			return
		}
	}
	if st.LogoURL != "" {
		if strings.HasPrefix(st.LogoURL, "/logos/") {
			st.LogoURL = fmt.Sprintf("%s%s", baseURL, st.LogoURL)
		} else if !strings.HasPrefix(st.LogoURL, "http://") && !strings.HasPrefix(st.LogoURL, "https://") {
			st.LogoURL = fmt.Sprintf("%s/logos/%s", baseURL, st.LogoURL)
		}
	}
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}
