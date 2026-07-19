package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mailru/easyjson"
	"github.com/rs/zerolog/log"
)

// Cached stats for comparison to avoid unnecessary broadcasts
var (
	lastStats      *StatsResponse
	statsMu        sync.RWMutex
	statsUpdateCh  chan struct{}
	statsDebouncer *time.Timer
	debounceMu     sync.Mutex

	monitorBatch   []Monitor
	monitorBatchMu sync.Mutex
	monitorTimer   *time.Timer
)

// queueMonitorUpdate batches rapid monitor updates into a single broadcast
func queueMonitorUpdate(monitor Monitor) {
	monitorBatchMu.Lock()
	defer monitorBatchMu.Unlock()
	
	monitorBatch = append(monitorBatch, monitor)
	
	if monitorTimer != nil {
		monitorTimer.Stop()
	}
	
	monitorTimer = time.AfterFunc(500*time.Millisecond, func() {
		monitorBatchMu.Lock()
		batch := monitorBatch
		monitorBatch = nil
		monitorBatchMu.Unlock()
		
		if len(batch) > 0 {
			broadcastUpdate("monitor_update_batch", batch)
		}
	})
}

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID   string
	Send chan []byte
}

// SSEBroadcaster manages all SSE connections
type SSEBroadcaster struct {
	clients   map[string]*SSEClient
	mu        sync.RWMutex
	broadcast chan []byte
}

var sseBroadcaster = &SSEBroadcaster{
	clients:   make(map[string]*SSEClient),
	broadcast: make(chan []byte, 256),
}

// addClient adds a new SSE client
func (b *SSEBroadcaster) addClient(id string) *SSEClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	client := &SSEClient{
		ID:   id,
		Send: make(chan []byte, 256),
	}
	b.clients[id] = client
	log.Info().Str("client_id", id).Int("total", len(b.clients)).Msg("[SSE] Client connected")
	return client
}

// removeClient removes an SSE client
func (b *SSEBroadcaster) removeClient(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if client, ok := b.clients[id]; ok {
		close(client.Send)
		delete(b.clients, id)
		log.Info().Str("client_id", id).Int("total", len(b.clients)).Msg("[SSE] Client disconnected")
	}
}

// broadcastMessage sends a message to all connected clients
func (b *SSEBroadcaster) broadcastMessage(message []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	clientCount := len(b.clients)
	if clientCount == 0 {
		log.Debug().Int("bytes", len(message)).Msg("[SSE] No clients connected, dropping message")
		return
	}
	
	sentCount := 0
	droppedCount := 0
	for id, client := range b.clients {
		select {
		case client.Send <- message:
			sentCount++
		default:
			droppedCount++
			log.Warn().Str("client_id", id).Msg("[SSE] Client channel full, dropping message")
		}
	}
	
	log.Debug().Int("sent", sentCount).Int("total", clientCount).Int("bytes", len(message)).
		Int("dropped", droppedCount).Msg("[SSE] Broadcast completed")
}

// broadcastUpdate broadcasts an update to all SSE clients
func broadcastUpdate(updateType string, data interface{}) {
	var jsonData []byte
	var err error
	
	// Try to use easyjson for supported types
	if marshaler, ok := data.(easyjson.Marshaler); ok {
		// For easyjson types, marshal the data directly and wrap in JSON object
		dataBytes, marshalErr := easyjson.Marshal(marshaler)
		if marshalErr != nil {
			log.Error().Err(marshalErr).Str("update_type", updateType).Msg("[SSE] Failed to marshal data")
			return
		}
		// Manually construct JSON object: {"type":"...","data":{...}}
		jsonData = []byte(fmt.Sprintf(`{"type":"%s","data":%s}`, updateType, string(dataBytes)))
	} else {
		// Fallback to standard json for unsupported types
		update := map[string]interface{}{
			"type": updateType,
			"data": data,
		}
		jsonData, err = json.Marshal(update)
		if err != nil {
			log.Error().Err(err).Str("update_type", updateType).Msg("[SSE] Failed to marshal update")
			return
		}
	}
	
	log.Debug().Str("update_type", updateType).Int("bytes", len(jsonData)).Msg("[SSE] Broadcasting update")
	go sseBroadcaster.broadcastMessage(jsonData)
}

// broadcastStatsIfChanged broadcasts stats only if they've changed (with debouncing)
func broadcastStatsIfChanged() {
	debounceMu.Lock()
	defer debounceMu.Unlock()
	
	// Reset debounce timer
	if statsDebouncer != nil {
		statsDebouncer.Stop()
	}
	
	// Debounce: wait 500ms after last monitor update before calculating stats
	statsDebouncer = time.AfterFunc(500*time.Millisecond, func() {
		newStats := getStats()
		
		statsMu.Lock()
		var oldUptime float64
		var oldUpInt, oldDownInt, oldAvgInt int
		if lastStats != nil {
			oldUptime = lastStats.OverallUptime
			oldUpInt = lastStats.ServicesUp
			oldDownInt = lastStats.ServicesDown
			oldAvgInt = lastStats.AvgResponseTime
		}
		
		changed := lastStats == nil || 
			lastStats.OverallUptime != newStats.OverallUptime ||
			lastStats.ServicesUp != newStats.ServicesUp ||
			lastStats.ServicesDown != newStats.ServicesDown ||
			lastStats.AvgResponseTime != newStats.AvgResponseTime
		
		if changed {
			log.Info().
				Float64("old_uptime", oldUptime).Float64("new_uptime", newStats.OverallUptime).
				Int("old_up", oldUpInt).Int("new_up", newStats.ServicesUp).
				Int("old_down", oldDownInt).Int("new_down", newStats.ServicesDown).
				Int("old_avg", oldAvgInt).Int("new_avg", newStats.AvgResponseTime).
				Msg("[SSE] Stats changed - broadcasting update")
			lastStats = &newStats
			statsMu.Unlock()
			broadcastUpdate("stats_update", newStats)
		} else {
			statsMu.Unlock()
			log.Debug().Msg("[SSE] Stats unchanged, skipping broadcast")
		}
	})
}

