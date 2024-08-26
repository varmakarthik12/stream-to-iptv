package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"stream-to-iptv/pkg/stream"
	"stream-to-iptv/pkg/utils"
	"strings"

	"github.com/go-chi/render"
	"github.com/grafov/m3u8"
)

func streamHandler(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/stream/")
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext != ".m3u8" && ext != ".ts" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	http.StripPrefix("/stream", http.FileServer(http.Dir(utils.GetBaseFolder()))).ServeHTTP(w, r)
}

func playlistHandler(w http.ResponseWriter, r *http.Request) {
	streams, err := stream.GetStreamConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stream configuration: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-mpegURL")

	playlist, err := m3u8.NewMediaPlaylist(0, uint(len(streams)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create playlist: %v", err), http.StatusInternalServerError)
		return
	}

	for _, s := range streams {
		streamURL := fmt.Sprintf("http://%s/stream/%s/%s", r.Host, s.Name, utils.GetStreamFileName(s.Name))
		groupTitles := strings.Join(s.Groups, ", ")
		entry := &m3u8.MediaSegment{
			URI:   streamURL,
			Title: fmt.Sprintf("#EXTINF:-1 tvg-id=\"%s\" tvg-logo=\"%s\" group-title=\"%s\",%s", s.TVGId, s.Logo, groupTitles, s.Name),
		}
		playlist.AppendSegment(entry)
	}

	// Manually construct the M3U8 content with the x-tvg-url attribute
	playlistContent := strings.Replace(playlist.Encode().String(), "#EXTM3U\n", "", 1)
	m3u8Content := fmt.Sprintf("#EXTM3U x-tvg-url=\"%s\"\n%s", utils.GetEPGURL(), playlistContent)
	render.PlainText(w, r, m3u8Content)
}
