package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

//go:embed dist
var staticFiles embed.FS

var db *gorm.DB

func init() {
	// Configure zerolog for console output with colors
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	
	// Use console writer for pretty output in development
	// In production, you can set ZEROLOG_LOG_LEVEL env var to control log level
	if os.Getenv("ZEROLOG_LOG_LEVEL") == "" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	
	// Set log level from environment variable if provided
	if level := os.Getenv("ZEROLOG_LOG_LEVEL"); level != "" {
		switch level {
		case "debug":
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		case "info":
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		case "warn":
			zerolog.SetGlobalLevel(zerolog.WarnLevel)
		case "error":
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		}
	}
}

func main() {
	// Initialize database
	initDB()

	// Start background checker
	startChecker()
	
	// Start cleanup scheduler (runs daily at midnight)
	go startCleanupScheduler()

	// API routes - Create a dedicated mux for routes that should be compressed
	gzipMux := http.NewServeMux()
	gzipMux.HandleFunc("/api/monitors", apiMonitors)
	gzipMux.HandleFunc("/api/monitors/create", apiCreateMonitor)
	gzipMux.HandleFunc("/api/monitors/export", apiExportMonitors)
	gzipMux.HandleFunc("/api/stats", apiStats)
	gzipMux.HandleFunc("/api/response-time", apiResponseTime)
	gzipMux.HandleFunc("/api/monitor", apiMonitor)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create sub filesystem")
	}

	fileServer := http.FileServer(http.FS(staticFS))

	// Handle SPA routing - serve index.html for all non-API routes
	gzipMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't serve index.html for API routes
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file - if it exists, serve it directly
		file, err := staticFS.Open(path)
		if err == nil {
			// File exists, let FileServer handle it (will close the file)
			file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File doesn't exist, serve index.html for SPA routing
		// Use FileServer to serve index.html efficiently
		r.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r)
	})

	// Initialize main router
	mainMux := http.NewServeMux()
	
	// SSE route MUST NOT be gzipped to prevent buffering delays
	mainMux.HandleFunc("/api/events", apiSSE)
	
	// Wrap the other routes with gzip compression and attach to main mux
	mainMux.Handle("/", gziphandler.GzipHandler(gzipMux))

	port := ":8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = ":" + envPort
	}

	log.Info().Str("port", port).Msg("🚀 Server starting")
	log.Info().Msg("📊 API endpoints:")
	log.Info().Msg("   GET /api/monitors - List all monitors")
	log.Info().Msg("   POST /api/monitors/create - Create a new monitor")
	log.Info().Msg("   GET /api/monitors/export - Export monitors as YAML")
	log.Info().Msg("   GET /api/stats - Get overall statistics")
	log.Info().Msg("   GET /api/response-time?id=<id>&range=<range> - Get response time data")
	log.Info().Msg("   GET /api/monitor?id=<id> - Get specific monitor")
	log.Info().Msg("   PUT /api/monitor?id=<id> - Update monitor")
	log.Info().Msg("   DELETE /api/monitor?id=<id> - Delete a monitor")
	log.Info().Msg("   GET /api/events - Server-Sent Events stream")
	log.Fatal().Err(http.ListenAndServe(port, mainMux)).Msg("Server failed")
}
