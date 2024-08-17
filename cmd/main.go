package main

import (
	"fmt"
	"net/http"
	"os"
	"stream-to-iptv/pkg/ffmpeg"
	"stream-to-iptv/pkg/ip"
	"stream-to-iptv/pkg/stream"
	"stream-to-iptv/pkg/utils"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func main() {

	port := utils.GetPort()

	err := initStream(port)
	if err != nil {
		logrus.Fatalf("Failed to initialize stream: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, middleware.RequestID, middleware.RealIP, middleware.Heartbeat("/health"))

	r.HandleFunc("/stream/*", streamHandler)
	r.Route("/", func(c chi.Router) {
		// TODO: implement M3U IPTV playlist
	})

	// Start the HTTP server
	logrus.Infof("Starting server on port :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

func initStream(port string) error {
	streams, err := stream.GetStreamConfig()
	if err != nil {
		return fmt.Errorf("failed to get stream configuration: %v", err)
	}

	ips, err := ip.GetLocalIP()
	if err != nil {
		return fmt.Errorf("failed to get local IP address: %v", err)
	}

	// Create output directories for each stream
	for _, stream := range streams {
		logrus.Infof("Starting stream for %s with Media source %s", stream.Name, stream.Media)

		os.MkdirAll(fmt.Sprintf("%s/%s", utils.GetBaseFolder(), stream.Name), os.ModePerm)
		err := ffmpeg.StartFFmpeg(stream)
		if err != nil {
			return fmt.Errorf("failed to start FFmpeg for stream %s: %v", stream.Name, err)
		}

		for _, ip := range ips {
			logrus.Infof("Stream %s available at http://%s:%s/stream/%s/%s", stream.Name, ip, port, stream.Name, utils.GetStreamFileName(stream.Name))
		}
	}

	return nil
}
