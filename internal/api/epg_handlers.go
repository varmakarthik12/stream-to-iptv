package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"

	"github.com/go-chi/chi/v5"
)

func (h *APIHandler) ListEPGSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.repo.GetAllEPGSources()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

func (h *APIHandler) CreateEPGSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                 string `json:"name"`
		URL                  string `json:"url"`
		RefreshIntervalHours int    `json:"refresh_interval_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" {
		http.Error(w, "Name and URL are required", http.StatusBadRequest)
		return
	}
	if req.RefreshIntervalHours <= 0 {
		req.RefreshIntervalHours = 24
	}

	src, err := h.repo.CreateEPGSource(req.Name, req.URL, req.RefreshIntervalHours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger initial background download
	go h.epgService.RefreshSource(src.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(src)
}

func (h *APIHandler) BulkCreateEPGSources(w http.ResponseWriter, r *http.Request) {
	var req []struct {
		Name                 string `json:"name"`
		URL                  string `json:"url"`
		RefreshIntervalHours int    `json:"refresh_interval_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	var createdSources []*models.EPGSource
	for _, item := range req {
		url := strings.TrimSpace(item.URL)
		if url == "" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = url
		}
		interval := item.RefreshIntervalHours
		if interval <= 0 {
			interval = 24
		}
		src, err := h.repo.CreateEPGSource(name, url, interval)
		if err == nil && src != nil {
			createdSources = append(createdSources, src)
			go h.epgService.RefreshSource(src.ID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdSources)
}

func (h *APIHandler) UpdateEPGSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name                 string `json:"name"`
		URL                  string `json:"url"`
		RefreshIntervalHours int    `json:"refresh_interval_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.RefreshIntervalHours <= 0 {
		req.RefreshIntervalHours = 24
	}

	if err := h.repo.UpdateEPGSource(id, req.Name, req.URL, req.RefreshIntervalHours); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *APIHandler) DeleteEPGSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = os.Remove(filepath.Join(db.GetEPGDir(), fmt.Sprintf("source_%s.xml", id)))
	if err := h.repo.DeleteEPGSource(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *APIHandler) RefreshEPGSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	go h.epgService.RefreshSource(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updating"})
}

func (h *APIHandler) SearchEPGChannels(w http.ResponseWriter, r *http.Request) {
	sourceID := r.URL.Query().Get("source_id")
	q := r.URL.Query().Get("q")
	channels, err := h.repo.SearchEPGChannels(sourceID, q, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}
