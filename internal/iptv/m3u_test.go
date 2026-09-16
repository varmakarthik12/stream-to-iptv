package iptv

import (
	"os"
	"strings"
	"testing"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"
)

func TestGeneratePlaylist(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-m3u-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	database, err := db.InitDB(tempDir)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer database.Close()

	repo := repository.NewRepository(database)
	m3uService := NewM3UService(repo)

	cat, _ := repo.CreateCategory("News", "news", 1)
	st := &models.Stream{
		Name:        "CNN HD",
		Slug:        "cnn-hd",
		MediaURL:    "http://example.com/cnn.m3u8",
		TVGId:       "cnn.us",
		TVGName:     "CNN USA",
		TVGChno:     "202",
		Enabled:     true,
		CategoryIDs: []string{cat.ID},
	}
	_, err = repo.CreateStream(st)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	playlist, err := m3uService.GeneratePlaylist("localhost:8068", "http", "")
	if err != nil {
		t.Fatalf("Failed to generate playlist: %v", err)
	}

	if !strings.Contains(playlist, "#EXTM3U x-tvg-url=\"http://localhost:8068/epg.xml\"") {
		t.Errorf("Playlist missing header or EPG URL: %s", playlist)
	}

	if !strings.Contains(playlist, `tvg-id="cnn.us"`) || !strings.Contains(playlist, `tvg-chno="202"`) {
		t.Errorf("Playlist missing TVG tags: %s", playlist)
	}

	if !strings.Contains(playlist, `group-title="News"`) {
		t.Errorf("Playlist missing group-title: %s", playlist)
	}

	if !strings.Contains(playlist, "http://localhost:8068/stream/cnn-hd/cnn-hd.m3u8") {
		t.Errorf("Playlist missing playback URL: %s", playlist)
	}
}
