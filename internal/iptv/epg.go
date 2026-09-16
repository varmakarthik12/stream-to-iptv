package iptv

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/models"
	"stream-to-iptv/internal/repository"

	"github.com/sirupsen/logrus"
)

type XMLTVChannel struct {
	XMLName     xml.Name `xml:"channel"`
	ID          string   `xml:"id,attr"`
	DisplayName string   `xml:"display-name"`
	Icon        struct {
		Src string `xml:"src,attr"`
	} `xml:"icon"`
}

type XMLTVProgramme struct {
	XMLName  xml.Name `xml:"programme"`
	Start    string   `xml:"start,attr"`
	Stop     string   `xml:"stop,attr"`
	Channel  string   `xml:"channel,attr"`
	Title    string   `xml:"title"`
	Desc     string   `xml:"desc"`
	Category string   `xml:"category"`
}

type EPGService struct {
	repo     *repository.Repository
	mu       sync.Mutex
	stopChan chan struct{}
}

func NewEPGService(repo *repository.Repository) *EPGService {
	s := &EPGService{
		repo:     repo,
		stopChan: make(chan struct{}),
	}
	go s.worker()
	return s
}

func (s *EPGService) Stop() {
	close(s.stopChan)
}

// RefreshSource downloads, caches, and parses an EPG source
func (s *EPGService) RefreshSource(sourceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	source, err := s.repo.GetEPGSourceByID(sourceID)
	if err != nil {
		return err
	}

	_ = s.repo.UpdateEPGSourceStatus(sourceID, "updating", "Downloading EPG data...", source.ChannelCount)

	logrus.Infof("Fetching EPG source %s from %s", source.Name, source.URL)
	req, err := http.NewRequestWithContext(context.Background(), "GET", source.URL, nil)
	if err != nil {
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", err.Error(), source.ChannelCount)
		return err
	}
	req.Header.Set("User-Agent", "stream-to-iptv/2.0")

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", fmt.Sprintf("Download failed: %v", err), source.ChannelCount)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", err.Error(), source.ChannelCount)
		return err
	}

	// Read first 2 bytes to check gzip magic header (0x1f, 0x8b)
	headerBytes := make([]byte, 2)
	n, _ := io.ReadFull(resp.Body, headerBytes)
	combinedReader := io.MultiReader(bytes.NewReader(headerBytes[:n]), resp.Body)

	var xmlReader io.Reader = combinedReader
	isGzip := (n == 2 && headerBytes[0] == 0x1f && headerBytes[1] == 0x8b) ||
		strings.HasSuffix(strings.ToLower(source.URL), ".gz")

	if isGzip {
		gz, err := gzip.NewReader(combinedReader)
		if err != nil {
			_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", fmt.Sprintf("Gzip error: %v", err), source.ChannelCount)
			return err
		}
		defer gz.Close()
		xmlReader = gz
	}

	// Save uncompressed XML to local cache file
	cacheFile := filepath.Join(db.GetEPGDir(), fmt.Sprintf("source_%s.xml", sourceID))
	out, err := os.Create(cacheFile)
	if err != nil {
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", fmt.Sprintf("Cache write failed: %v", err), source.ChannelCount)
		return err
	}

	// Stream full uncompressed XML to local cache file first
	if _, err := io.Copy(out, xmlReader); err != nil {
		out.Close()
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", fmt.Sprintf("Cache write failed: %v", err), source.ChannelCount)
		return err
	}
	out.Close()

	// Parse channel list from saved cache file
	cacheRead, err := os.Open(cacheFile)
	if err != nil {
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", fmt.Sprintf("Cache read failed: %v", err), source.ChannelCount)
		return err
	}
	channels, err := s.parseChannels(cacheRead)
	cacheRead.Close()

	if err != nil {
		_ = s.repo.UpdateEPGSourceStatus(sourceID, "error", fmt.Sprintf("XML parse failed: %v", err), 0)
		return err
	}

	// Save channels into SQLite
	if err := s.repo.SaveEPGChannels(sourceID, channels); err != nil {
		logrus.Errorf("Failed to save EPG channels for %s: %v", sourceID, err)
	}

	_ = s.repo.UpdateEPGSourceStatus(sourceID, "ok", "", len(channels))
	logrus.Infof("EPG source %s refreshed successfully with %d channels", source.Name, len(channels))

	// Re-generate merged EPG
	go s.GenerateMergedEPG()
	return nil
}

func (s *EPGService) parseChannels(r io.Reader) ([]models.EPGChannel, error) {
	decoder := xml.NewDecoder(r)
	var channels []models.EPGChannel

	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return channels, err
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "channel" {
				var ch XMLTVChannel
				if err := decoder.DecodeElement(&ch, &se); err == nil {
					channels = append(channels, models.EPGChannel{
						ChannelID:   ch.ID,
						DisplayName: strings.TrimSpace(ch.DisplayName),
						IconURL:     ch.Icon.Src,
					})
				}
			} else if se.Name.Local == "programme" {
				// Once programmes start, we have scanned all channels in standard XMLTV
				// Skip the rest for fast channel list indexing
				return channels, nil
			}
		}
	}

	return channels, nil
}

// GenerateMergedEPG generates a custom tailored XMLTV containing only configured channels
func (s *EPGService) GenerateMergedEPG() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	streams, err := s.repo.GetAllStreams()
	if err != nil {
		return err
	}

	settings, _ := s.repo.GetAllSettings()
	baseURL := "http://localhost:8068"
	if u := settings["server_url"]; u != "" {
		baseURL = strings.TrimRight(u, "/")
	}

	type mappedChannel struct {
		StreamID          string
		TVGId             string
		Name              string
		LogoURL           string
		PrimarySourceID   string
		PrimaryChannelID  string
		FallbackSourceID  string
		FallbackChannelID string
	}

	var mapped []mappedChannel
	sourcesNeeded := make(map[string]bool)

	for _, st := range streams {
		if !st.Enabled {
			continue
		}

		tvgID := st.TVGId
		if tvgID == "" {
			tvgID = st.Slug
		}

		logoURL := st.LogoURL
		if st.LogoID != "" {
			logo, err := s.repo.GetLogoByID(st.LogoID)
			if err == nil && logo != nil {
				if logo.IsLocal {
					logoURL = fmt.Sprintf("%s/logos/%s", baseURL, logo.FileName)
				} else {
					logoURL = logo.URL
				}
			}
		}

		mc := mappedChannel{
			StreamID: st.ID,
			TVGId:    tvgID,
			Name:     st.Name,
			LogoURL:  logoURL,
		}

		if st.EPGMapping != nil {
			mc.PrimarySourceID = st.EPGMapping.PrimaryEPGSourceID
			mc.PrimaryChannelID = st.EPGMapping.PrimaryChannelID
			mc.FallbackSourceID = st.EPGMapping.FallbackEPGSourceID
			mc.FallbackChannelID = st.EPGMapping.FallbackChannelID

			if mc.PrimarySourceID != "" {
				sourcesNeeded[mc.PrimarySourceID] = true
			}
			if mc.FallbackSourceID != "" {
				sourcesNeeded[mc.FallbackSourceID] = true
			}
		}

		mapped = append(mapped, mc)
	}

	// Output paths
	destPath := filepath.Join(db.GetEPGDir(), "generated_epg.xml")
	destGzPath := filepath.Join(db.GetEPGDir(), "generated_epg.xml.gz")

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Write XML Header
	outFile.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	outFile.WriteString("<!DOCTYPE tv SYSTEM \"xmltv.dtd\">\n")
	outFile.WriteString("<tv generator-info-name=\"stream-to-iptv\">\n")

	// 1. Write Channels
	for _, mc := range mapped {
		outFile.WriteString(fmt.Sprintf("  <channel id=\"%s\">\n", xmlEscape(mc.TVGId)))
		outFile.WriteString(fmt.Sprintf("    <display-name>%s</display-name>\n", xmlEscape(mc.Name)))
		if mc.LogoURL != "" {
			outFile.WriteString(fmt.Sprintf("    <icon src=\"%s\" />\n", xmlEscape(mc.LogoURL)))
		}
		outFile.WriteString("  </channel>\n")
	}

	// 2. Extract and translate programmes from cached sources
	programmesFound := make(map[string]int) // targetTVGId -> count

	for sourceID := range sourcesNeeded {
		cacheFile := filepath.Join(db.GetEPGDir(), fmt.Sprintf("source_%s.xml", sourceID))
		f, err := os.Open(cacheFile)
		if err != nil {
			continue
		}

		// Build map of sourceChannelID -> targetTVGId for this source
		channelMap := make(map[string]string)
		for _, mc := range mapped {
			if mc.PrimarySourceID == sourceID && mc.PrimaryChannelID != "" {
				channelMap[mc.PrimaryChannelID] = mc.TVGId
			}
		}

		decoder := xml.NewDecoder(f)
		for {
			t, err := decoder.Token()
			if err != nil {
				break
			}

			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "programme" {
					var prog XMLTVProgramme
					if err := decoder.DecodeElement(&prog, &se); err == nil {
						if targetTVG, ok := channelMap[prog.Channel]; ok {
							programmesFound[targetTVG]++
							writeProgramme(outFile, prog, targetTVG)
						}
					}
				}
			}
		}
		f.Close()
	}

	// 3. Fallback extraction for channels with 0 programmes
	for _, mc := range mapped {
		if programmesFound[mc.TVGId] == 0 && mc.FallbackSourceID != "" && mc.FallbackChannelID != "" {
			cacheFile := filepath.Join(db.GetEPGDir(), fmt.Sprintf("source_%s.xml", mc.FallbackSourceID))
			f, err := os.Open(cacheFile)
			if err != nil {
				continue
			}

			decoder := xml.NewDecoder(f)
			for {
				t, err := decoder.Token()
				if err != nil {
					break
				}
				switch se := t.(type) {
				case xml.StartElement:
					if se.Name.Local == "programme" {
						var prog XMLTVProgramme
						if err := decoder.DecodeElement(&prog, &se); err == nil {
							if prog.Channel == mc.FallbackChannelID {
								writeProgramme(outFile, prog, mc.TVGId)
							}
						}
					}
				}
			}
			f.Close()
		}
	}

	outFile.WriteString("</tv>\n")
	outFile.Sync()

	// Compress to .gz
	_ = compressFile(destPath, destGzPath)

	logrus.Info("Merged XMLTV EPG generated successfully")
	return nil
}

func writeProgramme(w io.Writer, p XMLTVProgramme, targetChannelID string) {
	fmt.Fprintf(w, "  <programme start=\"%s\" stop=\"%s\" channel=\"%s\">\n",
		xmlEscape(p.Start), xmlEscape(p.Stop), xmlEscape(targetChannelID))
	if p.Title != "" {
		fmt.Fprintf(w, "    <title>%s</title>\n", xmlEscape(p.Title))
	}
	if p.Desc != "" {
		fmt.Fprintf(w, "    <desc>%s</desc>\n", xmlEscape(p.Desc))
	}
	if p.Category != "" {
		fmt.Fprintf(w, "    <category>%s</category>\n", xmlEscape(p.Category))
	}
	w.Write([]byte("  </programme>\n"))
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func compressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	_, err = io.Copy(gw, in)
	return err
}

func (s *EPGService) worker() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkPendingRefreshes()
		}
	}
}

func (s *EPGService) checkPendingRefreshes() {
	sources, err := s.repo.GetAllEPGSources()
	if err != nil {
		return
	}

	for _, src := range sources {
		needsRefresh := false
		if src.LastRefreshedAt == nil {
			needsRefresh = true
		} else {
			interval := time.Duration(src.RefreshIntervalHours) * time.Hour
			if time.Since(*src.LastRefreshedAt) >= interval {
				needsRefresh = true
			}
		}

		if needsRefresh {
			logrus.Infof("Auto-refreshing EPG source: %s", src.Name)
			_ = s.RefreshSource(src.ID)
		}
	}
}
