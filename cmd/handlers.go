package main

import (
	"net/http"
	"path/filepath"
	"stream-to-iptv/pkg/utils"
	"strings"
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
