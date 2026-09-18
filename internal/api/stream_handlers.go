package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"stream-to-iptv/internal/iptv"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/stream"

	"github.com/go-chi/chi/v5"
)

func (h *APIHandler) ListStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := h.repo.GetAllStreams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, _ := h.repo.GetAllSettings()
	baseURL := fmt.Sprintf("http://%s", r.Host)
	if custom := settings["server_url"]; custom != "" {
		baseURL = strings.TrimRight(custom, "/")
	}
	token := settings["iptv_token"]

	for i := range streams {
		streams[i].Status = h.streamMgr.GetStreamStatus(streams[i].Slug)
		streams[i].PlaybackURL = iptv.GetPlaybackURL(baseURL, streams[i].Slug, token)
		h.resolveStreamLogo(&streams[i], baseURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(streams)
}

func (h *APIHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := h.repo.GetStreamByID(id)
	if err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	settings, _ := h.repo.GetAllSettings()
	baseURL := fmt.Sprintf("http://%s", r.Host)
	if custom := settings["server_url"]; custom != "" {
		baseURL = strings.TrimRight(custom, "/")
	}
	token := settings["iptv_token"]

	st.Status = h.streamMgr.GetStreamStatus(st.Slug)
	st.PlaybackURL = iptv.GetPlaybackURL(baseURL, st.Slug, token)
	h.resolveStreamLogo(st, baseURL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func (h *APIHandler) CreateStream(w http.ResponseWriter, r *http.Request) {
	var st models.Stream
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(st.Name) == "" || strings.TrimSpace(st.MediaURL) == "" {
		http.Error(w, "Stream name and media URL are required", http.StatusBadRequest)
		return
	}

	if st.Slug == "" {
		st.Slug = stream.Slugify(st.Name)
	} else {
		st.Slug = stream.Slugify(st.Slug)
	}

	// Ensure unique slug
	existing, _ := h.repo.GetStreamBySlug(st.Slug)
	if existing != nil {
		st.Slug = fmt.Sprintf("%s-%d", st.Slug, timeNowUnix())
	}

	if st.Mode == "" {
		st.Mode = "ondemand"
	}
	if st.IdleTimeoutSec <= 0 {
		st.IdleTimeoutSec = 180
	}
	if st.BufferSize == "" {
		st.BufferSize = "1000000"
	}

	if strings.TrimSpace(st.TVGName) == "" {
		st.TVGName = strings.TrimSpace(st.Name)
	}
	if strings.TrimSpace(st.TVGId) == "" {
		if st.EPGMapping != nil && strings.TrimSpace(st.EPGMapping.PrimaryChannelID) != "" {
			st.TVGId = strings.TrimSpace(st.EPGMapping.PrimaryChannelID)
		} else {
			st.TVGId = st.Slug
		}
	}

	created, err := h.repo.CreateStream(&st)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create stream: %v", err), http.StatusInternalServerError)
		return
	}

	// If always-on and enabled, start it
	if created.Enabled && created.Mode == "always_on" {
		go h.streamMgr.StartStream(created)
	}

	// Re-generate EPG since channels changed
	go h.epgService.GenerateMergedEPG()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *APIHandler) UpdateStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := h.repo.GetStreamByID(id)
	if err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	var st models.Stream
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	st.ID = id
	if strings.TrimSpace(st.Name) == "" || strings.TrimSpace(st.MediaURL) == "" {
		http.Error(w, "Stream name and media URL are required", http.StatusBadRequest)
		return
	}

	if st.Slug == "" {
		st.Slug = existing.Slug
	} else {
		st.Slug = stream.Slugify(st.Slug)
	}

	if strings.TrimSpace(st.TVGName) == "" {
		st.TVGName = strings.TrimSpace(st.Name)
	}
	if strings.TrimSpace(st.TVGId) == "" {
		if st.EPGMapping != nil && strings.TrimSpace(st.EPGMapping.PrimaryChannelID) != "" {
			st.TVGId = strings.TrimSpace(st.EPGMapping.PrimaryChannelID)
		} else {
			st.TVGId = st.Slug
		}
	}

	if err := h.repo.UpdateStream(&st); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update stream: %v", err), http.StatusInternalServerError)
		return
	}

	// Restart stream if it was running or switched to always_on
	if existing.Slug != st.Slug || !st.Enabled {
		h.streamMgr.StopStream(existing.Slug)
	}
	if st.Enabled && st.Mode == "always_on" {
		go h.streamMgr.StartStream(&st)
	}

	go h.epgService.GenerateMergedEPG()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func (h *APIHandler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := h.repo.GetStreamByID(id)
	if err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	h.streamMgr.StopStream(st.Slug)
	if err := h.repo.DeleteStream(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go h.epgService.GenerateMergedEPG()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *APIHandler) StartStreamAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := h.repo.GetStreamByID(id)
	if err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	err = h.streamMgr.StartStream(st)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "starting"})
}

func (h *APIHandler) StopStreamAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := h.repo.GetStreamByID(id)
	if err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	h.streamMgr.StopStream(st.Slug)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (h *APIHandler) GetStreamLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := h.repo.GetStreamByID(id)
	if err != nil {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	logs := h.streamMgr.GetStreamLogs(st.Slug)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
