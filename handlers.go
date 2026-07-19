package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/mailru/easyjson"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Compile regex once at package level for better performance
var unicodePattern = regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`)

// setCORSHeaders sets common CORS headers
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// setJSONHeaders sets common headers for JSON responses
func setJSONHeaders(w http.ResponseWriter) {
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
}

// encodeJSONWithCompression streams JSON directly to the response writer.
// Note: Actual gzip compression is now handled globally by the gziphandler middleware.
func encodeJSONWithCompression(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// Try to use easyjson for supported types
	if marshaler, ok := data.(easyjson.Marshaler); ok {
		_, err := easyjson.MarshalToWriter(marshaler, w)
		return err
	}
	
	// Fallback to standard json
	encoder := json.NewEncoder(w)
	return encoder.Encode(data)
}

// getResponseTimeData retrieves response time history for a monitor within a time range
func getResponseTimeData(monitorID string, timeRange string) []ResponseTimeData {
	id, err := strconv.ParseUint(monitorID, 10, 32)
	if err != nil {
		return []ResponseTimeData{}
	}

	// Calculate time cutoff based on range
	var cutoffTime time.Time
	now := time.Now()
	
	switch timeRange {
	case "1h":
		cutoffTime = now.Add(-1 * time.Hour)
	case "12h":
		cutoffTime = now.Add(-12 * time.Hour)
	case "1w":
		cutoffTime = now.Add(-7 * 24 * time.Hour)
	case "1y":
		cutoffTime = now.Add(-365 * 24 * time.Hour)
	default:
		// Default to 24 hours
		cutoffTime = now.Add(-24 * time.Hour)
	}

	// Get checks within time range, ordered by creation time
	var checks []CheckHistory
	query := db.Where("monitor_id = ? AND created_at > ?", id, cutoffTime).
		Order("created_at ASC")
	
	// Limit results based on time range to avoid too much data
	switch timeRange {
	case "1h":
		query = query.Limit(60) // Max 60 points for 1 hour
	case "12h":
		query = query.Limit(144) // Max 144 points for 12 hours
	case "24h":
		query = query.Limit(288) // Max 288 points for 24 hours
	case "1w":
		query = query.Limit(168) // Max 168 points for 1 week
	case "1y":
		query = query.Limit(365) // Max 365 points for 1 year
	default:
		query = query.Limit(50)
	}
	
	query.Find(&checks)

	// If no data, return empty array
	if len(checks) == 0 {
		return []ResponseTimeData{}
	}

	// Convert to response time data format
	// Send ISO 8601 timestamps and let the frontend format them in the user's timezone
	data := make([]ResponseTimeData, len(checks))
	for i, check := range checks {
		// Send ISO 8601 timestamp (UTC) - frontend will format in user's timezone
		isoTimestamp := check.CreatedAt.Format(time.RFC3339)
		
		// Also provide a fallback formatted string (UTC) for backwards compatibility
		var timeStr string
		switch timeRange {
		case "1h", "12h", "24h":
			timeStr = check.CreatedAt.Format("03:04 PM")
		case "1w":
			timeStr = check.CreatedAt.Format("Mon 03:04 PM")
		case "1y":
			timeStr = check.CreatedAt.Format("Jan 2")
		default:
			timeStr = check.CreatedAt.Format("03:04 PM")
		}
		
		data[i] = ResponseTimeData{
			Time:         timeStr,      // Fallback (will be overridden by frontend)
			Timestamp:    isoTimestamp, // ISO 8601 timestamp for client-side formatting
			ResponseTime: float64(check.ResponseTime),
		}
	}

	return data
}

// apiMonitors handles GET requests to list all monitors
func apiMonitors(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("[API] Request")
	
	setJSONHeaders(w)

	if r.Method == http.MethodGet {
		var monitors []Monitor
		if err := db.Find(&monitors).Error; err != nil {
			log.Error().Err(err).Msg("[API] ERROR GET /api/monitors")
			http.Error(w, "Failed to fetch monitors", http.StatusInternalServerError)
			return
		}
		log.Info().Int("count", len(monitors)).Msg("[API] GET /api/monitors")
		if err := encodeJSONWithCompression(w, r, monitors); err != nil {
			log.Error().Err(err).Msg("[API] ERROR encoding monitors")
		}
		return
	}

	log.Warn().Str("method", r.Method).Msg("[API] ERROR Method not allowed")
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// apiCreateMonitor handles POST requests to create a new monitor
func apiCreateMonitor(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("[API] Request")
	
	setJSONHeaders(w)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		log.Debug().Msg("[API] OPTIONS /api/monitors/create: CORS preflight")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Warn().Str("method", r.Method).Msg("[API] ERROR Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateMonitorRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("[API] ERROR POST /api/monitors/create: Failed to read body")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	if err := easyjson.Unmarshal(bodyBytes, &req); err != nil {
		log.Error().Err(err).Msg("[API] ERROR POST /api/monitors/create: Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" {
		log.Warn().Str("name", req.Name).Str("url", req.URL).Msg("[API] ERROR POST /api/monitors/create: Missing required fields")
		http.Error(w, "Name and URL are required", http.StatusBadRequest)
		return
	}

	// Set default check interval to 60 seconds if not provided
	checkInterval := req.CheckInterval
	if checkInterval <= 0 {
		checkInterval = 60
	}

	monitor := Monitor{
		Name:         req.Name,
		URL:          req.URL,
		IsThirdParty: req.IsThirdParty,
		Icon:         req.Icon,
		Status:       "unknown",
		Uptime:       0,
		ResponseTime: 0,
		LastCheck:    "never",
		CheckInterval: checkInterval,
	}

	if err := db.Create(&monitor).Error; err != nil {
		log.Error().Err(err).Msg("[API] ERROR POST /api/monitors/create: Failed to create monitor")
		http.Error(w, "Failed to create monitor", http.StatusInternalServerError)
		return
	}

	log.Info().Uint("id", monitor.ID).Str("name", monitor.Name).Str("url", monitor.URL).
		Int("check_interval", monitor.CheckInterval).Msg("[API] POST /api/monitors/create: Created monitor")

	// Trigger immediate scheduler refresh to start ticker for new monitor
	go func() {
		time.Sleep(100 * time.Millisecond)
		monitorScheduler.refreshScheduler()
	}()
	
	// Immediately check the new monitor
	go checkService(&monitor)
	
	// Broadcast new monitor via SSE
	broadcastUpdate("monitor_added", monitor)
	
	// Schedule stats update (debounced)
	broadcastStatsIfChanged()

	w.WriteHeader(http.StatusCreated)
	if err := encodeJSONWithCompression(w, r, monitor); err != nil {
		log.Error().Err(err).Msg("[API] ERROR encoding monitor")
	}
}

// apiStats handles GET requests to retrieve overall statistics
func apiStats(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("[API] Request")
	
	setJSONHeaders(w)

	if r.Method != http.MethodGet {
		log.Warn().Str("method", r.Method).Msg("[API] ERROR Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := getStats()
	log.Info().Float64("uptime", stats.OverallUptime).Int("up", stats.ServicesUp).
		Int("down", stats.ServicesDown).Int("avg_ms", stats.AvgResponseTime).Msg("[API] GET /api/stats")
	if err := encodeJSONWithCompression(w, r, stats); err != nil {
		log.Error().Err(err).Msg("[API] ERROR encoding stats")
	}
}

// apiResponseTime handles GET requests to retrieve response time history
func apiResponseTime(w http.ResponseWriter, r *http.Request) {
	monitorID := r.URL.Query().Get("id")
	timeRange := r.URL.Query().Get("range")
	if monitorID == "" {
		monitorID = "1"
	}
	if timeRange == "" {
		timeRange = "24h" // Default to 24 hours
	}
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Str("id", monitorID).Str("range", timeRange).Msg("[API] Request")
	
	setJSONHeaders(w)

	if r.Method != http.MethodGet {
		log.Warn().Str("method", r.Method).Msg("[API] ERROR Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := getResponseTimeData(monitorID, timeRange)
	log.Info().Str("id", monitorID).Str("range", timeRange).Int("points", len(data)).Msg("[API] GET /api/response-time")
	if err := encodeJSONWithCompression(w, r, data); err != nil {
		log.Error().Err(err).Msg("[API] ERROR encoding response time data")
	}
}

// apiSSE handles Server-Sent Events connections
func apiSSE(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("remote_addr", r.RemoteAddr).Str("user_agent", r.UserAgent()).Msg("[SSE] New connection request")
	
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create client
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
	client := sseBroadcaster.addClient(clientID)
	defer func() {
		sseBroadcaster.removeClient(clientID)
		log.Debug().Str("client_id", clientID).Msg("[SSE] Cleanup completed")
	}()

	// Send initial connection message
	connectMsg := `{"type":"connected"}`
	fmt.Fprintf(w, "data: %s\n\n", connectMsg)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
		log.Debug().Str("client_id", clientID).Msg("[SSE] Sent connection confirmation")
	} else {
		log.Warn().Str("client_id", clientID).Msg("[SSE] ResponseWriter does not support flushing")
	}

	// Keep connection alive and send updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	messageCount := 0
	keepaliveCount := 0
	startTime := time.Now()

	for {
		select {
		case message := <-client.Send:
			messageCount++
			fmt.Fprintf(w, "data: %s\n\n", string(message))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
				log.Debug().Int("message_num", messageCount).Str("client_id", clientID).
					Int("bytes", len(message)).Msg("[SSE] Sent message")
			} else {
				log.Error().Str("client_id", clientID).Msg("[SSE] ERROR: Cannot flush message")
			}
		case <-ticker.C:
			// Send keepalive
			keepaliveCount++
			fmt.Fprintf(w, ": keepalive\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
				log.Debug().Int("keepalive_num", keepaliveCount).Str("client_id", clientID).Msg("[SSE] Sent keepalive")
			}
		case <-r.Context().Done():
			duration := time.Since(startTime)
			log.Info().Str("client_id", clientID).Dur("duration", duration).
				Int("messages", messageCount).Int("keepalives", keepaliveCount).
				Msg("[SSE] Client disconnected")
			return
		}
	}
}

// apiMonitor handles GET, PUT, and DELETE requests for individual monitors
func apiMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Str("id", id).Msg("[API] Request")
	
	setCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		log.Debug().Msg("[API] OPTIONS /api/monitor: CORS preflight")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		setJSONHeaders(w)
		if id == "" {
			log.Warn().Msg("[API] ERROR GET /api/monitor: Missing id parameter")
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		var monitor Monitor
		monitorID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR GET /api/monitor: Invalid id parameter")
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		if err := db.First(&monitor, monitorID).Error; err != nil {
			log.Warn().Str("id", id).Msg("[API] ERROR GET /api/monitor: Monitor not found")
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		log.Info().Str("id", id).Str("name", monitor.Name).Msg("[API] GET /api/monitor")
		if err := encodeJSONWithCompression(w, r, monitor); err != nil {
			log.Error().Err(err).Msg("[API] ERROR encoding monitor")
		}
		return
	}

	if r.Method == http.MethodPut {
		setJSONHeaders(w)
		if id == "" {
			log.Warn().Msg("[API] ERROR PUT /api/monitor: Missing id parameter")
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		monitorID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR PUT /api/monitor: Invalid id parameter")
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		var monitor Monitor
		if err := db.First(&monitor, monitorID).Error; err != nil {
			log.Warn().Str("id", id).Msg("[API] ERROR PUT /api/monitor: Monitor not found")
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Read body once
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR PUT /api/monitor: Failed to read body")
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Check if this is a pause/unpause request (has only "paused" field)
		// Use standard json for anonymous structs
		var pauseReq struct {
			Paused *bool `json:"paused"`
		}
		if err := json.Unmarshal(bodyBytes, &pauseReq); err == nil && pauseReq.Paused != nil {
			// This is a pause/unpause request
			monitor.Paused = *pauseReq.Paused
			if err := db.Save(&monitor).Error; err != nil {
				log.Error().Err(err).Str("id", id).Msg("[API] ERROR PUT /api/monitor: Failed to update paused state")
				http.Error(w, "Failed to update paused state", http.StatusInternalServerError)
				return
			}
			log.Info().Str("id", id).Bool("paused", monitor.Paused).Msg("[API] PUT /api/monitor: Updated paused state")
			
			// Trigger immediate scheduler refresh to pick up pause state changes
			go func() {
				time.Sleep(100 * time.Millisecond)
				monitorScheduler.refreshScheduler()
			}()
			
			// Broadcast update via SSE
			broadcastUpdate("monitor_update", monitor)
			broadcastStatsIfChanged()
			
			if err := encodeJSONWithCompression(w, r, monitor); err != nil {
				log.Error().Err(err).Msg("[API] ERROR encoding monitor")
			}
			return
		}

		// Regular update request
		var req CreateMonitorRequest
		if err := easyjson.Unmarshal(bodyBytes, &req); err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR PUT /api/monitor: Invalid request body")
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.URL == "" {
			log.Warn().Str("id", id).Msg("[API] ERROR PUT /api/monitor: Missing required fields")
			http.Error(w, "Name and URL are required", http.StatusBadRequest)
			return
		}

		// Update monitor fields
		monitor.Name = req.Name
		monitor.URL = req.URL
		monitor.IsThirdParty = req.IsThirdParty
		monitor.Icon = req.Icon
		
		// Only update CheckInterval if explicitly provided (non-zero)
		// This allows updating other fields without resetting the interval
		if req.CheckInterval > 0 {
			monitor.CheckInterval = req.CheckInterval
		}

		if err := db.Save(&monitor).Error; err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR PUT /api/monitor: Failed to update monitor")
			http.Error(w, "Failed to update monitor", http.StatusInternalServerError)
			return
		}
		
		// Reload monitor from database to ensure we have the latest data
		if err := db.First(&monitor, monitorID).Error; err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR PUT /api/monitor: Failed to reload monitor")
			http.Error(w, "Failed to reload monitor", http.StatusInternalServerError)
			return
		}

		log.Info().Str("id", id).Str("name", monitor.Name).Str("url", monitor.URL).
			Int("check_interval", monitor.CheckInterval).Msg("[API] PUT /api/monitor: Updated monitor")
		
		// Trigger immediate scheduler refresh to pick up interval changes
		// Use a delay to ensure database transaction is committed
		go func() {
			// Wait longer to ensure DB write is fully committed
			time.Sleep(500 * time.Millisecond)
			log.Info().Uint64("monitor_id", monitorID).
				Int("check_interval", monitor.CheckInterval).
				Msg("[API] Triggering scheduler refresh after monitor update")
			monitorScheduler.refreshScheduler()
		}()
		
		// Broadcast update via SSE
		broadcastUpdate("monitor_update", monitor)
		broadcastStatsIfChanged()
		
		if err := encodeJSONWithCompression(w, r, monitor); err != nil {
			log.Error().Err(err).Msg("[API] ERROR encoding monitor")
		}
		return
	}

	if r.Method == http.MethodDelete {
		if id == "" {
			log.Warn().Msg("[API] ERROR DELETE /api/monitor: Missing id parameter")
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		monitorID, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR DELETE /api/monitor: Invalid id parameter")
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		// Check if monitor exists first
		var monitor Monitor
		if err := db.First(&monitor, monitorID).Error; err != nil {
			log.Warn().Str("id", id).Msg("[API] ERROR DELETE /api/monitor: Monitor not found")
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		if err := db.Delete(&Monitor{}, monitorID).Error; err != nil {
			log.Error().Err(err).Str("id", id).Msg("[API] ERROR DELETE /api/monitor: Failed to delete")
			http.Error(w, "Failed to delete monitor", http.StatusInternalServerError)
			return
		}

		log.Info().Str("id", id).Str("name", monitor.Name).Msg("[API] DELETE /api/monitor: Successfully deleted monitor")
		
		// Broadcast deletion via SSE
		broadcastUpdate("monitor_deleted", map[string]interface{}{"id": monitorID})
		
		// Schedule stats update (debounced)
		broadcastStatsIfChanged()
		
		w.WriteHeader(http.StatusNoContent)
		return
	}

	log.Warn().Str("method", r.Method).Msg("[API] ERROR Method not allowed")
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// apiExportMonitors handles GET requests to export all monitors as YAML
func apiExportMonitors(w http.ResponseWriter, r *http.Request) {
	log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("[API] Request")

	setCORSHeaders(w)

	if r.Method != http.MethodGet {
		log.Warn().Str("method", r.Method).Msg("[API] ERROR Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch all monitors from database
	var monitors []Monitor
	if err := db.Find(&monitors).Error; err != nil {
		log.Error().Err(err).Msg("[API] ERROR GET /api/monitors/export: Failed to fetch monitors")
		http.Error(w, "Failed to fetch monitors", http.StatusInternalServerError)
		return
	}

	// Convert monitors to YAML format
	config := ConfigFile{
		Monitors: make([]MonitorConfig, 0, len(monitors)),
	}

	for _, monitor := range monitors {
		monitorConfig := MonitorConfig{
			Name:         monitor.Name,
			URL:          monitor.URL,
			Icon:         monitor.Icon,
			CheckInterval: monitor.CheckInterval,
			IsThirdParty: monitor.IsThirdParty,
			Paused:       monitor.Paused,
		}
		config.Monitors = append(config.Monitors, monitorConfig)
	}

	// Marshal to YAML using encoder with custom options
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2) // Use 2-space indentation
	
	if err := encoder.Encode(&config); err != nil {
		log.Error().Err(err).Msg("[API] ERROR GET /api/monitors/export: Failed to marshal YAML")
		http.Error(w, "Failed to generate YAML", http.StatusInternalServerError)
		return
	}
	encoder.Close()
	
	yamlData := buf.Bytes()
	
	// Convert Unicode escape sequences back to actual emojis
	// The yaml.v3 library escapes Unicode like "\U0001F4BB" - we need to convert these back
	yamlData = convertUnicodeEscapes(yamlData)
	
	// Set headers for file download
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=monitors.yaml")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(yamlData)))

	// Write YAML data
	if _, err := w.Write(yamlData); err != nil {
		log.Error().Err(err).Msg("[API] ERROR GET /api/monitors/export: Failed to write response")
		return
	}

	log.Info().Int("count", len(monitors)).Msg("[API] GET /api/monitors/export: Exported monitors as YAML")
}

// convertUnicodeEscapes converts YAML Unicode escape sequences like "\U0001F4BB" back to actual emojis
func convertUnicodeEscapes(data []byte) []byte {
	result := unicodePattern.ReplaceAllFunc(data, func(match []byte) []byte {
		// Extract the hex code (8 digits after \U)
		// match is "\U0001F4BB", so we skip the first 2 bytes (\U) and take the next 8
		hexStr := string(match[2:10])
		
		// Parse the hex string to a rune (Unicode code point)
		codePoint, err := strconv.ParseUint(hexStr, 16, 32)
		if err != nil {
			// If parsing fails, return the original match
			return match
		}
		
		// Convert the code point to a UTF-8 encoded string
		r := rune(codePoint)
		if !utf8.ValidRune(r) {
			return match
		}
		
		// Encode the rune to UTF-8 bytes
		utf8Bytes := make([]byte, utf8.RuneLen(r))
		utf8.EncodeRune(utf8Bytes, r)
		
		return utf8Bytes
	})
	
	return result
}

