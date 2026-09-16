package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stream-to-iptv/internal/db"

	"github.com/go-chi/chi/v5"
)

func (h *APIHandler) ListLogos(w http.ResponseWriter, r *http.Request) {
	logos, err := h.repo.GetAllLogos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseURL := fmt.Sprintf("http://%s", r.Host)
	settings, _ := h.repo.GetAllSettings()
	if custom := settings["server_url"]; custom != "" {
		baseURL = strings.TrimRight(custom, "/")
	}

	for i := range logos {
		if logos[i].IsLocal {
			logos[i].URL = fmt.Sprintf("%s/logos/%s", baseURL, logos[i].FileName)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logos)
}

func (h *APIHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	// 10 MB limit
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large (max 10MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file in form field 'file'", http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".svg" && ext != ".webp" {
		http.Error(w, "Supported formats: PNG, JPG, JPEG, SVG, WebP", http.StatusBadRequest)
		return
	}

	// Read content and compute SHA256 for deterministic unique filename
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	hash := sha256.Sum256(content)
	fileName := fmt.Sprintf("%s%s", hex.EncodeToString(hash[:8]), ext)
	targetPath := filepath.Join(db.GetLogosDir(), fileName)

	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save logo: %v", err), http.StatusInternalServerError)
		return
	}

	baseURL := fmt.Sprintf("http://%s", r.Host)
	publicURL := fmt.Sprintf("%s/logos/%s", baseURL, fileName)

	logo, err := h.repo.CreateLogo(name, fileName, publicURL, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(logo)
}

func (h *APIHandler) ImportLogoURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Imported Logo"
	}

	// Download image with timeout and 10MB limit
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(req.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to download image: %v", err), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Failed to download image: HTTP %d", resp.StatusCode), http.StatusBadRequest)
		return
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(req.URL)
	if ext == "" || len(ext) > 5 {
		ext = ".png"
	}

	hash := sha256.Sum256(content)
	fileName := fmt.Sprintf("%s%s", hex.EncodeToString(hash[:8]), ext)
	targetPath := filepath.Join(db.GetLogosDir(), fileName)

	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	baseURL := fmt.Sprintf("http://%s", r.Host)
	publicURL := fmt.Sprintf("%s/logos/%s", baseURL, fileName)

	logo, err := h.repo.CreateLogo(req.Name, fileName, publicURL, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logo)
}

func (h *APIHandler) DeleteLogo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logo, err := h.repo.GetLogoByID(id)
	if err == nil && logo != nil && logo.IsLocal && logo.FileName != "" {
		_ = os.Remove(filepath.Join(db.GetLogosDir(), logo.FileName))
	}

	_ = h.repo.DeleteLogo(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
