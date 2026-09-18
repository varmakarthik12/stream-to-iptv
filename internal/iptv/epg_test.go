package iptv

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"
)

func TestParseChannelsWithHTMLEntities(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="ch-1">
    <display-name>Channel &amp; One &nbsp; HD</display-name>
    <icon src="http://example.com/ch1.png" />
  </channel>
  <channel id="ch-2">
    <display-name>Channel Two</display-name>
  </channel>
  <programme start="20260918000000 +0000" stop="20260918010000 +0000" channel="ch-1">
    <title>Sample Show</title>
  </programme>
</tv>`

	s := &EPGService{}
	channels, err := s.parseChannels(strings.NewReader(xmlData))
	if err != nil {
		t.Fatalf("parseChannels failed on valid XML with entities: %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	if channels[0].ChannelID != "ch-1" || !strings.Contains(channels[0].DisplayName, "Channel & One") {
		t.Errorf("unexpected channel 0: %+v", channels[0])
	}
	if channels[0].IconURL != "http://example.com/ch1.png" {
		t.Errorf("unexpected icon: %s", channels[0].IconURL)
	}
}

func TestGenerateMergedEPGRichMetadataAndFallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-epg-*")
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
	s := NewEPGService(repo)
	defer s.Stop()

	// 1. Create EPG Source
	src, err := repo.CreateEPGSource("Test Guide", "http://example.com/epg.xml", 24)
	if err != nil {
		t.Fatalf("Failed to create EPG source: %v", err)
	}

	// Write mock source XML cache file
	epgDir := db.GetEPGDir()
	sourceCachePath := filepath.Join(epgDir, fmt.Sprintf("source_%s.xml", src.ID))

	mockXML := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="1668">
    <display-name>And Xplor HD</display-name>
    <icon src="http://example.com/xplor.png" />
  </channel>
  <programme start="20260918000000 +0000" stop="20260918020000 +0000" channel="1668" catchup-id="12345">
    <title>Blockbuster Movie</title>
    <sub-title>The Grand Premiere</sub-title>
    <desc>An epic adventure with actors &amp; directors.</desc>
    <category>Action</category>
    <icon src="http://example.com/movie_poster.jpg" />
    <credits>
      <director>Christopher Nolan</director>
      <actor>Leonardo DiCaprio</actor>
    </credits>
    <episode-num system="xmltv_ns">1.0.0</episode-num>
  </programme>
  <programme start="20260918020000 +0000" stop="20260918030000 +0000" channel="1668">
    <title>Late Night News</title>
    <desc>Daily news updates &amp; headlines.</desc>
  </programme>
</tv>`
	if err := os.WriteFile(sourceCachePath, []byte(mockXML), 0644); err != nil {
		t.Fatalf("Failed to write mock cache file: %v", err)
	}

	// 2. Create stream 1 mapped to source
	st1 := &models.Stream{
		Name:     "And Xplore",
		Slug:     "and-xplore",
		MediaURL: "http://example.com/stream1.m3u8",
		TVGId:    "1668",
		TVGName:  "And Xplor HD",
		LogoURL:  "http://localhost:8068/logos/channel_logo.png",
		Enabled:  true,
		EPGMapping: &models.StreamEPGMapping{
			PrimaryEPGSourceID: src.ID,
			PrimaryChannelID:   "1668",
		},
	}
	_, err = repo.CreateStream(st1)
	if err != nil {
		t.Fatalf("Failed to create stream 1: %v", err)
	}

	// 3. Create stream 2 mapped to the SAME channel ID (multi-stream test)
	st2 := &models.Stream{
		Name:     "Xplor Backup",
		Slug:     "xplor-backup",
		MediaURL: "http://example.com/stream2.m3u8",
		TVGId:    "xplor-backup-id",
		TVGName:  "Xplor Backup",
		LogoURL:  "http://localhost:8068/logos/backup_logo.png",
		Enabled:  true,
		EPGMapping: &models.StreamEPGMapping{
			PrimaryEPGSourceID: src.ID,
			PrimaryChannelID:   "1668",
		},
	}
	_, err = repo.CreateStream(st2)
	if err != nil {
		t.Fatalf("Failed to create stream 2: %v", err)
	}

	// Generate merged EPG
	if err := s.GenerateMergedEPG(); err != nil {
		t.Fatalf("GenerateMergedEPG failed: %v", err)
	}

	generatedPath := filepath.Join(epgDir, "generated_epg.xml")
	gzPath := filepath.Join(epgDir, "generated_epg.xml.gz")

	// Verify files exist
	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("Failed to read generated_epg.xml: %v", err)
	}
	xmlStr := string(content)

	// Verify channel headers
	if !strings.Contains(xmlStr, `<channel id="1668">`) {
		t.Errorf("Missing channel 1668 definition")
	}
	if !strings.Contains(xmlStr, `<channel id="xplor-backup-id">`) {
		t.Errorf("Missing channel xplor-backup-id definition")
	}

	// Verify rich metadata preservation
	if !strings.Contains(xmlStr, `<director>Christopher Nolan</director>`) {
		t.Errorf("Missing director credits in output")
	}
	if !strings.Contains(xmlStr, `<actor>Leonardo DiCaprio</actor>`) {
		t.Errorf("Missing actor credits in output")
	}
	if !strings.Contains(xmlStr, `<sub-title>The Grand Premiere</sub-title>`) {
		t.Errorf("Missing sub-title in output")
	}
	if !strings.Contains(xmlStr, `catchup-id="12345"`) {
		t.Errorf("Missing catchup-id attribute in output")
	}

	// Verify movie poster was preserved
	if !strings.Contains(xmlStr, `http://example.com/movie_poster.jpg`) {
		t.Errorf("Missing movie poster icon in output")
	}

	// Verify programme without icon falls back to channel's logo
	if !strings.Contains(xmlStr, `http://localhost:8068/logos/channel_logo.png`) {
		t.Errorf("Fallback to channel logo failed for stream 1")
	}
	if !strings.Contains(xmlStr, `http://localhost:8068/logos/backup_logo.png`) {
		t.Errorf("Fallback to channel logo failed for stream 2")
	}

	// Verify both streams received programmes
	if !strings.Contains(xmlStr, `channel="1668"`) || !strings.Contains(xmlStr, `channel="xplor-backup-id"`) {
		t.Errorf("Both mapped streams should have programmes, got:\n%s", xmlStr)
	}

	// Verify GZ archive is valid
	gzFile, err := os.Open(gzPath)
	if err != nil {
		t.Fatalf("Failed to open gz file: %v", err)
	}
	defer gzFile.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		t.Fatalf("Invalid gzip file: %v", err)
	}
	gzContent, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("Failed to decompress gzip: %v", err)
	}
	if len(gzContent) != len(content) {
		t.Errorf("Gzip uncompressed size %d != file size %d", len(gzContent), len(content))
	}
}
