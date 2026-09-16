package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/iptv"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"
	"stream-to-iptv/internal/stream"

	"github.com/go-chi/chi/v5"
)

func setupTestApp(t *testing.T) (chi.Router, *repository.Repository, *stream.Manager, string) {
	tempDir := t.TempDir()
	os.Setenv("DATA_DIR", tempDir)

	database, err := db.InitDB(tempDir)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	repo := repository.NewRepository(database)
	_ = stream.EnsureLoadingBumper(tempDir)

	streamMgr := stream.NewManager(repo)
	epgService := iptv.NewEPGService(repo)
	m3uService := iptv.NewM3UService(repo)

	r := chi.NewRouter()
	RegisterRoutes(r, repo, streamMgr, epgService, m3uService)

	return r, repo, streamMgr, tempDir
}

func TestColdStartHLSAndLoadingBumper(t *testing.T) {
	router, repo, _, tempDir := setupTestApp(t)

	// Create a test stream in repo
	st := &models.Stream{
		Name:      "Test Channel",
		Slug:      "test-channel",
		MediaURL:  "http://example.com/live.ts",
		Mode:      "on_demand",
		Enabled:   true,
		AutoRecover: true,
	}
	if _, err := repo.CreateStream(st); err != nil {
		t.Fatalf("CreateStream failed: %v", err)
	}

	// 1. Request stream manifest during cold-start (FFmpeg hasn't produced segments yet)
	req := httptest.NewRequest("GET", "/stream/test-channel/test-channel.m3u8", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 during cold start, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "loading.ts") {
		t.Fatalf("Expected cold-start manifest to reference loading.ts, got:\n%s", body)
	}
	if !strings.Contains(body, "#EXT-X-TARGETDURATION:2") {
		t.Fatalf("Expected target duration 2 in loading manifest, got:\n%s", body)
	}

	// 2. Request the loading bumper segment
	reqBumper := httptest.NewRequest("GET", "/stream/test-channel/loading.ts", nil)
	recBumper := httptest.NewRecorder()
	router.ServeHTTP(recBumper, reqBumper)

	if recBumper.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 for loading.ts, got %d", recBumper.Code)
	}
	ct := recBumper.Header().Get("Content-Type")
	if ct != "video/mp2t" {
		t.Fatalf("Expected video/mp2t content type, got %s", ct)
	}
	if recBumper.Body.Len() == 0 {
		t.Fatalf("Expected non-empty loading.ts bumper body")
	}

	// 3. Simulate FFmpeg producing media segments and manifest on disk
	streamDir := filepath.Join(tempDir, "streams", "test-channel")
	_ = os.MkdirAll(streamDir, 0755)
	_ = os.WriteFile(filepath.Join(streamDir, "segment_000.ts"), []byte("dummy-ts-segment"), 0644)
	realManifest := "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:6\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:6.000000,\nsegment_000.ts\n"
	_ = os.WriteFile(filepath.Join(streamDir, "test-channel.m3u8"), []byte(realManifest), 0644)

	// Next poll of test-channel.m3u8 should detect ready segments and inject #EXT-X-DISCONTINUITY
	reqLive := httptest.NewRequest("GET", "/stream/test-channel/test-channel.m3u8", nil)
	recLive := httptest.NewRecorder()
	router.ServeHTTP(recLive, reqLive)

	if recLive.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 for live manifest, got %d", recLive.Code)
	}
	liveBody := recLive.Body.String()
	if !strings.Contains(liveBody, "#EXT-X-DISCONTINUITY") {
		t.Fatalf("Expected #EXT-X-DISCONTINUITY during cold-start transition, got:\n%s", liveBody)
	}
	if !strings.Contains(liveBody, "segment_000.ts") {
		t.Fatalf("Expected segment_000.ts in live manifest, got:\n%s", liveBody)
	}

	// 4. Request for non-existent channel must return 404
	reqNotFound := httptest.NewRequest("GET", "/stream/non-existent/stream.m3u8", nil)
	recNotFound := httptest.NewRecorder()
	router.ServeHTTP(recNotFound, reqNotFound)

	if recNotFound.Code != http.StatusNotFound {
		t.Fatalf("Expected HTTP 404 for non-existent stream, got %d", recNotFound.Code)
	}
}

func TestHostSystemStatusEndpoint(t *testing.T) {
	router, repo, _, _ := setupTestApp(t)

	// Create user and token for protected auth
	user, err := repo.CreateUser("admin", "password123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	token, err := createToken(user.ID, user.Username)
	if err != nil {
		t.Fatalf("createToken failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/system/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 from /api/system/status, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var status map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to parse JSON status: %v", err)
	}

	// Verify telemetry and system fields exist
	if _, ok := status["cpu_cores"]; !ok {
		t.Fatalf("Missing cpu_cores in status response")
	}
	if _, ok := status["memory_alloc_mb"]; !ok {
		t.Fatalf("Missing memory_alloc_mb in status response")
	}
	if _, ok := status["data_dir"]; !ok {
		t.Fatalf("Missing data_dir in status response")
	}
	if _, ok := status["ffmpeg_path"]; !ok {
		t.Fatalf("Missing ffmpeg_path in status response")
	}
}

func TestStreamLogoResolution(t *testing.T) {
	router, repo, _, _ := setupTestApp(t)

	// Create a local logo
	logo, err := repo.CreateLogo("Test Logo", "test-logo.png", "/logos/test-logo.png", true)
	if err != nil {
		t.Fatalf("CreateLogo failed: %v", err)
	}

	// Create a stream with logo_id but empty logo_url (simulating library selection)
	st := &models.Stream{
		Name:     "Logo Test Channel",
		Slug:     "logo-test-channel",
		MediaURL: "http://example.com/test.ts",
		LogoID:   logo.ID,
		LogoURL:  "", // empty!
		Enabled:  true,
	}
	if _, err := repo.CreateStream(st); err != nil {
		t.Fatalf("CreateStream failed: %v", err)
	}

	// Create auth token
	user, err := repo.CreateUser("logo-admin", "password123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	token, err := createToken(user.ID, user.Username)
	if err != nil {
		t.Fatalf("createToken failed: %v", err)
	}

	// Fetch /api/streams
	req := httptest.NewRequest("GET", "/api/streams", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200 from /api/streams, got %d", rec.Code)
	}

	var streams []models.Stream
	if err := json.NewDecoder(rec.Body).Decode(&streams); err != nil {
		t.Fatalf("Failed to decode streams: %v", err)
	}

	found := false
	for _, s := range streams {
		if s.Slug == "logo-test-channel" {
			found = true
			if s.LogoURL == "" {
				t.Fatalf("Expected resolved LogoURL for stream with logo_id, got empty string")
			}
			if !strings.Contains(s.LogoURL, "test-logo.png") {
				t.Fatalf("Expected LogoURL to contain test-logo.png, got: %s", s.LogoURL)
			}
		}
	}
	if !found {
		t.Fatalf("Stream logo-test-channel not found in response")
	}
}

