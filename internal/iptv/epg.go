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

type XMLTVProgrammeElement struct {
	XMLName xml.Name   `xml:"programme"`
	Attrs   []xml.Attr `xml:",any,attr"`
	Inner   []byte     `xml:",innerxml"`
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
	decoder.Entity = xml.HTMLEntity
	var channels []models.EPGChannel
	consecutiveProgrammes := 0

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
				consecutiveProgrammes = 0
				var ch XMLTVChannel
				if err := decoder.DecodeElement(&ch, &se); err == nil {
					channels = append(channels, models.EPGChannel{
						ChannelID:   ch.ID,
						DisplayName: strings.TrimSpace(ch.DisplayName),
						IconURL:     ch.Icon.Src,
					})
				}
			} else if se.Name.Local == "programme" {
				consecutiveProgrammes++
				// Once programmes start and we have seen a solid batch with no channels,
				// stop scanning for fast channel list indexing
				if consecutiveProgrammes > 100 && len(channels) > 0 {
					return channels, nil
				}
				_ = decoder.Skip()
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

		// TVGId resolution consistent with playlist.m3u
		tvgID := st.TVGId
		if tvgID == "" {
			if st.EPGMapping != nil && st.EPGMapping.PrimaryChannelID != "" {
				tvgID = st.EPGMapping.PrimaryChannelID
			} else {
				tvgID = st.Slug
			}
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

	// Retrieve all active EPG sources
	allSources, _ := s.repo.GetAllEPGSources()
	allSourceIDs := make([]string, 0, len(allSources))
	for _, src := range allSources {
		allSourceIDs = append(allSourceIDs, src.ID)
	}

	// For streams that do not have an explicit primary source assigned, allow auto-discovery across all active sources
	for _, mc := range mapped {
		if mc.PrimarySourceID == "" || mc.PrimaryChannelID == "" {
			for _, srcID := range allSourceIDs {
				sourcesNeeded[srcID] = true
			}
			break
		}
	}

	// Output paths
	destPath := filepath.Join(db.GetEPGDir(), "generated_epg.xml")
	destGzPath := filepath.Join(db.GetEPGDir(), "generated_epg.xml.gz")
	tmpPath := destPath + ".tmp"
	tmpGzPath := destGzPath + ".tmp"

	outFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

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

		// Build map of sourceChannelID -> slice of target mappedChannel
		// This supports multiple streams mapped to the same channel without overwriting
		channelMap := make(map[string][]mappedChannel)
		for _, mc := range mapped {
			if mc.PrimarySourceID == sourceID && mc.PrimaryChannelID != "" {
				channelMap[mc.PrimaryChannelID] = append(channelMap[mc.PrimaryChannelID], mc)
			} else if (mc.PrimarySourceID == "" || mc.PrimarySourceID == sourceID) && mc.TVGId != "" {
				channelMap[mc.TVGId] = append(channelMap[mc.TVGId], mc)
			}
		}

		decoder := xml.NewDecoder(f)
		decoder.Entity = xml.HTMLEntity

		for {
			t, err := decoder.Token()
			if err != nil {
				break
			}

			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "programme" {
					var prog XMLTVProgrammeElement
					if err := decoder.DecodeElement(&prog, &se); err == nil {
						var chID string
						for _, a := range prog.Attrs {
							if a.Name.Local == "channel" {
								chID = a.Value
								break
							}
						}
						if targetMCs, ok := channelMap[chID]; ok {
							for _, targetMC := range targetMCs {
								programmesFound[targetMC.TVGId]++
								writeProgrammeElement(outFile, prog, targetMC.TVGId, targetMC.LogoURL)
							}
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
			decoder.Entity = xml.HTMLEntity

			for {
				t, err := decoder.Token()
				if err != nil {
					break
				}
				switch se := t.(type) {
				case xml.StartElement:
					if se.Name.Local == "programme" {
						var prog XMLTVProgrammeElement
						if err := decoder.DecodeElement(&prog, &se); err == nil {
							var chID string
							for _, a := range prog.Attrs {
								if a.Name.Local == "channel" {
									chID = a.Value
									break
								}
							}
							if chID == mc.FallbackChannelID {
								programmesFound[mc.TVGId]++
								writeProgrammeElement(outFile, prog, mc.TVGId, mc.LogoURL)
							}
						}
					}
				}
			}
			f.Close()
		}
	}

	// 4. Secondary fallback: check any other available active EPG source for matching TVGId
	for _, mc := range mapped {
		if programmesFound[mc.TVGId] == 0 && mc.TVGId != "" {
			for _, srcID := range allSourceIDs {
				if srcID == mc.PrimarySourceID || srcID == mc.FallbackSourceID {
					continue
				}
				cacheFile := filepath.Join(db.GetEPGDir(), fmt.Sprintf("source_%s.xml", srcID))
				f, err := os.Open(cacheFile)
				if err != nil {
					continue
				}

				decoder := xml.NewDecoder(f)
				decoder.Entity = xml.HTMLEntity

				for {
					t, err := decoder.Token()
					if err != nil {
						break
					}
					switch se := t.(type) {
					case xml.StartElement:
						if se.Name.Local == "programme" {
							var prog XMLTVProgrammeElement
							if err := decoder.DecodeElement(&prog, &se); err == nil {
								var chID string
								for _, a := range prog.Attrs {
									if a.Name.Local == "channel" {
										chID = a.Value
										break
									}
								}
								if chID == mc.TVGId {
									programmesFound[mc.TVGId]++
									writeProgrammeElement(outFile, prog, mc.TVGId, mc.LogoURL)
								}
							}
						}
					}
				}
				f.Close()
				if programmesFound[mc.TVGId] > 0 {
					break
				}
			}
		}
	}

	outFile.WriteString("</tv>\n")
	outFile.Sync()
	outFile.Close()

	// Atomic replace for uncompressed XML
	_ = os.Rename(tmpPath, destPath)

	// Compress to .gz atomically
	if err := compressFile(destPath, tmpGzPath); err == nil {
		_ = os.Rename(tmpGzPath, destGzPath)
	}

	logrus.Infof("Merged XMLTV EPG generated successfully (%d streams mapped)", len(mapped))
	return nil
}

func writeProgrammeElement(w io.Writer, p XMLTVProgrammeElement, targetChannelID, channelLogoURL string) {
	fmt.Fprintf(w, "  <programme")
	for _, a := range p.Attrs {
		if a.Name.Local == "channel" {
			fmt.Fprintf(w, " channel=\"%s\"", xmlEscape(targetChannelID))
		} else {
			fmt.Fprintf(w, " %s=\"%s\"", a.Name.Local, xmlEscape(a.Value))
		}
	}
	fmt.Fprintf(w, ">\n")

	innerStr := string(p.Inner)
	// If program has no poster/icon, inject channel's icon as fallback
	hasIcon := strings.Contains(innerStr, "<icon")
	if !hasIcon && channelLogoURL != "" {
		fmt.Fprintf(w, "    <icon src=\"%s\" />\n", xmlEscape(channelLogoURL))
	}
	// Sanitize control characters that are forbidden in XML 1.0
	cleanInner := cleanXMLText(innerStr)
	w.Write([]byte("    "))
	w.Write([]byte(cleanInner))
	w.Write([]byte("\n  </programme>\n"))
}

func cleanXMLText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0x09 || r == 0x0A || r == 0x0D ||
			(r >= 0x20 && r <= 0xD7FF) ||
			(r >= 0xE000 && r <= 0xFFFD) ||
			(r >= 0x10000 && r <= 0x10FFFF) {
			return r
		}
		return -1
	}, s)
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
