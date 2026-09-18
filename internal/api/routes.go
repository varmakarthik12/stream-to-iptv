package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/iptv"
	"stream-to-iptv/internal/repository"
	"stream-to-iptv/internal/stream"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func RegisterRoutes(
	r chi.Router,
	repo *repository.Repository,
	streamMgr *stream.Manager,
	epgService *iptv.EPGService,
	m3uService *iptv.M3UService,
) {
	authHandler := NewAuthHandler(repo)
	apiHandler := NewAPIHandler(repo, streamMgr, epgService, m3uService)

	// Global Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Universal CORS Setup for Web UI and IPTV Players
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link", "Content-Length", "Content-Range"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// IPTV Public Endpoints
	r.Get("/playlist.m3u", func(w http.ResponseWriter, req *http.Request) {
		token := req.URL.Query().Get("token")
		scheme := "http"
		if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		m3uContent, err := m3uService.GeneratePlaylist(req.Host, scheme, token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Write([]byte(m3uContent))
	})

	r.Get("/epg.xml", func(w http.ResponseWriter, req *http.Request) {
		epgPath := filepath.Join(db.GetEPGDir(), "generated_epg.xml")
		if fi, err := os.Stat(epgPath); os.IsNotExist(err) || fi.Size() == 0 {
			_ = epgService.GenerateMergedEPG()
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		http.ServeFile(w, req, epgPath)
	})

	r.Get("/epg.xml.gz", func(w http.ResponseWriter, req *http.Request) {
		epgGzPath := filepath.Join(db.GetEPGDir(), "generated_epg.xml.gz")
		if fi, err := os.Stat(epgGzPath); os.IsNotExist(err) || fi.Size() == 0 {
			_ = epgService.GenerateMergedEPG()
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/x-gzip")
		http.ServeFile(w, req, epgGzPath)
	})

	// Stream Files: /stream/{slug}/{filename}
	r.HandleFunc("/stream/*", func(w http.ResponseWriter, req *http.Request) {
		// Set universal streaming CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		subPath := strings.TrimPrefix(req.URL.Path, "/stream/")
		parts := strings.Split(strings.Trim(subPath, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		slug := parts[0]
		filename := ""
		if len(parts) > 1 {
			filename = filepath.Base(parts[1])
		} else {
			filename = slug + ".m3u8"
		}

		if filename == "." || filename == "/" || filename == "\\" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".m3u8" && ext != ".ts" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Verify stream exists and is enabled
		st, err := repo.GetStreamBySlug(slug)
		if err != nil || !st.Enabled {
			http.Error(w, "Stream not found or disabled", http.StatusNotFound)
			return
		}

		// Touch stream to wake FFmpeg if idle/on-demand
		streamMgr.Touch(slug)

		// Serve loading bumper segment
		if filename == "loading.ts" {
			w.Header().Set("Content-Type", "video/mp2t")
			w.Header().Set("Cache-Control", "public, max-age=10")
			target := filepath.Join(db.GetDataDir(), "loading.ts")
			if _, err := os.Stat(target); os.IsNotExist(err) {
				_ = stream.EnsureLoadingBumper(db.GetDataDir())
			}
			http.ServeFile(w, req, target)
			return
		}

		// Serve HLS manifest with cold-start transition handling
		if ext == ".m3u8" {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			// Wait up to 8 seconds for the stream to become ready during cold-start.
			// On fast sources FFmpeg produces the first segment well within this window,
			// giving a fully seamless transition. On slower sources the player falls back
			// to the loading bumper and re-polls every 2s until FFmpeg catches up.
			ready, shouldDiscontinuity := streamMgr.WaitForStreamReady(slug, 8*time.Second)
			if !ready {
				// FFmpeg is still cold-starting: serve dynamic loading bumper playlist with sliding buffer
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(streamMgr.GenerateDynamicLoadingPlaylist(slug)))
				return
			}

			// Stream is ready on disk
			playlistPath := filepath.Join(db.GetStreamsDir(), slug, filename)
			if shouldDiscontinuity {
				data, err := os.ReadFile(playlistPath)
				if err == nil {
					content := string(data)
					if !strings.Contains(content, "#EXT-X-DISCONTINUITY") {
						idx := strings.Index(content, "#EXTINF")
						if idx != -1 {
							content = content[:idx] + "#EXT-X-DISCONTINUITY\n" + content[idx:]
						}
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(content))
					return
				}
			}

			http.ServeFile(w, req, playlistPath)
			return
		}

		// Serve TS media segments
		segmentPath := filepath.Join(db.GetStreamsDir(), slug, filename)
		w.Header().Set("Content-Type", "video/mp2t")
		http.ServeFile(w, req, segmentPath)
	})

	// Logos serving: /logos/{filename}
	r.Handle("/logos/*", http.StripPrefix("/logos", http.FileServer(http.Dir(db.GetLogosDir()))))

	// REST API Routes
	r.Route("/api", func(api chi.Router) {
		// Public Auth & Setup
		api.Get("/setup/status", authHandler.SetupStatus)
		api.Post("/setup", authHandler.Setup)
		api.Post("/auth/login", authHandler.Login)
		api.Post("/auth/logout", authHandler.Logout)

		// Protected Admin Routes
		api.Group(func(protected chi.Router) {
			protected.Use(authHandler.RequireAuth)

			protected.Get("/auth/me", authHandler.Me)

			// Streams
			protected.Get("/streams", apiHandler.ListStreams)
			protected.Post("/streams", apiHandler.CreateStream)
			protected.Get("/streams/{id}", apiHandler.GetStream)
			protected.Put("/streams/{id}", apiHandler.UpdateStream)
			protected.Delete("/streams/{id}", apiHandler.DeleteStream)
			protected.Post("/streams/{id}/start", apiHandler.StartStreamAction)
			protected.Post("/streams/{id}/stop", apiHandler.StopStreamAction)
			protected.Get("/streams/{id}/logs", apiHandler.GetStreamLogs)

			// Categories
			protected.Get("/categories", apiHandler.ListCategories)
			protected.Post("/categories", apiHandler.CreateCategory)
			protected.Put("/categories/{id}", apiHandler.UpdateCategory)
			protected.Delete("/categories/{id}", apiHandler.DeleteCategory)

			// Logos
			protected.Get("/logos", apiHandler.ListLogos)
			protected.Post("/logos/upload", apiHandler.UploadLogo)
			protected.Post("/logos/import-url", apiHandler.ImportLogoURL)
			protected.Delete("/logos/{id}", apiHandler.DeleteLogo)

			// EPG
			protected.Get("/epg-sources", apiHandler.ListEPGSources)
			protected.Post("/epg-sources", apiHandler.CreateEPGSource)
			protected.Post("/epg-sources/bulk", apiHandler.BulkCreateEPGSources)
			protected.Put("/epg-sources/{id}", apiHandler.UpdateEPGSource)
			protected.Delete("/epg-sources/{id}", apiHandler.DeleteEPGSource)
			protected.Post("/epg-sources/{id}/refresh", apiHandler.RefreshEPGSource)
			protected.Get("/epg-sources/channels", apiHandler.SearchEPGChannels)

			// Settings & System
			protected.Get("/settings", apiHandler.GetSettings)
			protected.Put("/settings", apiHandler.UpdateSettings)
			protected.Get("/system/status", apiHandler.GetSystemStatus)
		})
	})
}
