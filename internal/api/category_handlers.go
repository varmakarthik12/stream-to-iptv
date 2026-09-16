package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"stream-to-iptv/internal/stream"

	"github.com/go-chi/chi/v5"
)

func (h *APIHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.GetAllCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func (h *APIHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	slug := stream.Slugify(req.Name)
	cat, err := h.repo.CreateCategory(strings.TrimSpace(req.Name), slug, req.SortOrder)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create category: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cat)
}

func (h *APIHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	slug := stream.Slugify(req.Name)
	if err := h.repo.UpdateCategory(id, strings.TrimSpace(req.Name), slug, req.SortOrder); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *APIHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteCategory(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
