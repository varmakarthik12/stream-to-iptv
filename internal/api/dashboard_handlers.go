package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"stream-to-iptv/internal/db"
	"stream-to-iptv/pkg/ip"
)

func (h *APIHandler) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	ips, _ := ip.GetLocalIP()
	settings, _ := h.repo.GetAllSettings()
	port := settings["port"]
	if port == "" {
		port = "8068"
	}

	baseURL := fmt.Sprintf("http://%s:%s", r.Host, port)
	if len(ips) > 0 {
		baseURL = fmt.Sprintf("http://%s:%s", ips[0], port)
	}
	if custom := settings["server_url"]; custom != "" {
		baseURL = strings.TrimRight(custom, "/")
	}

	token := settings["iptv_token"]
	playlistURL := fmt.Sprintf("%s/playlist.m3u", baseURL)
	epgURL := fmt.Sprintf("%s/epg.xml", baseURL)
	if token != "" {
		playlistURL = fmt.Sprintf("%s?token=%s", playlistURL, token)
		epgURL = fmt.Sprintf("%s?token=%s", epgURL, token)
	}

	streamCount, _ := h.repo.CountStreams()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	activeCount := h.streamMgr.GetActiveCount()
	problemCount := h.streamMgr.GetProblemCount()

	allStreams, _ := h.repo.GetAllStreams()
	var problemStreams []map[string]interface{}
	for _, st := range allStreams {
		status := h.streamMgr.GetStreamStatus(st.Slug)
		if status == "error" {
			problemStreams = append(problemStreams, map[string]interface{}{
				"id":                  st.ID,
				"name":                st.Name,
				"slug":                st.Slug,
				"status":              status,
				"auto_recover":        st.AutoRecover,
				"recover_timeout_sec": st.RecoverTimeoutSec,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"local_ips":             ips,
		"base_url":              baseURL,
		"playlist_url":          playlistURL,
		"epg_url":               epgURL,
		"stream_count":          streamCount,
		"active_streams_count":  activeCount,
		"problem_streams_count": problemCount,
		"problem_streams":       problemStreams,
		"data_dir":              db.GetDataDir(),
		"ffmpeg_path":           h.streamMgr.ResolveFFmpegBinary(),
		"cpu_cores":             runtime.NumCPU(),
		"goroutines":            runtime.NumGoroutine(),
		"memory_alloc_mb":       float64(mem.Alloc) / 1024 / 1024,
		"memory_sys_mb":         float64(mem.Sys) / 1024 / 1024,
	})
}
