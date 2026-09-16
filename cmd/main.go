package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stream-to-iptv/internal/api"
	"stream-to-iptv/internal/db"
	"stream-to-iptv/internal/iptv"
	"stream-to-iptv/internal/repository"
	"stream-to-iptv/internal/stream"
	"stream-to-iptv/web"
	"stream-to-iptv/pkg/ip"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func main() {
	portFlag := flag.String("port", "", "Server listening port (defaults to PORT env or 8068)")
	dataDirFlag := flag.String("data-dir", "", "Path to data directory for sqlite and assets (defaults to DATA_DIR or ./data)")
	flag.Parse()

	// 1. Initialize SQLite database & migrations
	dataDir := *dataDirFlag
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "./data"
	}

	database, err := db.InitDB(dataDir)
	if err != nil {
		logrus.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	repo := repository.NewRepository(database)

	// 2. Resolve port
	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		dbPort, _ := repo.GetSetting("port", "8068")
		port = dbPort
	}
	if port == "" {
		port = "8068"
	}

	// 3. Initialize Services and Assets
	if err := stream.EnsureLoadingBumper(db.GetDataDir()); err != nil {
		logrus.Warnf("Failed to initialize loading bumper: %v", err)
	}
	streamMgr := stream.NewManager(repo)
	epgService := iptv.NewEPGService(repo)
	defer epgService.Stop()

	m3uService := iptv.NewM3UService(repo)

	// 4. Start Always-On streams
	streamMgr.StartAllAlwaysOnStreams()

	// 5. Setup Chi Router
	r := chi.NewRouter()

	// Register API and public IPTV endpoints
	api.RegisterRoutes(r, repo, streamMgr, epgService, m3uService)

	// Register embedded React SPA web routes
	web.RegisterWebRoutes(r)

	// Print access banner
	ips, _ := ip.GetLocalIP()
	logrus.Infof("==================================================")
	logrus.Infof("Stream to IPTV Server running on port :%s", port)
	logrus.Infof("Web Management UI: http://localhost:%s", port)
	for _, localIP := range ips {
		logrus.Infof("Network Access:    http://%s:%s", localIP, port)
		logrus.Infof("IPTV M3U Playlist: http://%s:%s/playlist.m3u", localIP, port)
		logrus.Infof("XMLTV EPG Guide:   http://%s:%s/epg.xml", localIP, port)
	}
	logrus.Infof("Data Directory:    %s", db.GetDataDir())
	logrus.Infof("==================================================")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server listen failed: %v", err)
		}
	}()

	<-stopChan
	logrus.Info("Shutting down server gracefully...")

	// Stop all active stream FFmpeg processes
	streamMgr.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("Server shutdown error: %v", err)
	}

	logrus.Info("Server stopped cleanly")
}
