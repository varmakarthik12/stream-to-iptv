package api

import (
	"encoding/json"
	"net/http"
)

func (h *APIHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.GetAllSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *APIHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	for k, v := range settings {
		_ = h.repo.SetSetting(k, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
