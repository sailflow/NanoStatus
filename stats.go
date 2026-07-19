package main

import (
	"sync"
	"time"
)

var (
	cachedStats   StatsResponse
	cachedStatsMu sync.RWMutex
)

func init() {
	// Start the background updater for stats
	go func() {
		// Wait for db to be initialized
		time.Sleep(2 * time.Second)
		updateCachedStats()
		ticker := time.NewTicker(15 * time.Second)
		for range ticker.C {
			updateCachedStats()
		}
	}()
}

// updateCachedStats recalculates and caches the global stats
func updateCachedStats() {
	if db == nil {
		return
	}

	var stats struct {
		UnpausedCount int64
		UpCount       int64
		DownCount     int64
		TotalUptime   float64
	}
	
	db.Model(&Monitor{}).
		Select(`
			COUNT(*) as unpaused_count,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) as down_count,
			SUM(uptime) as total_uptime
		`).
		Where("paused = ?", false).
		Scan(&stats)
	
	upCount := int(stats.UpCount)
	downCount := int(stats.DownCount)
	unpausedCount := int(stats.UnpausedCount)
	totalUptime := stats.TotalUptime

	var avgResponseTime int
	
	// Calculate global average response time using the current monitor states.
	// This is O(monitors) and extremely fast, replacing the O(millions) scan over check_histories.
	var responseStats struct {
		TotalResponseTime int64
		ResponseCount      int64
	}
	db.Model(&Monitor{}).
		Select(`SUM(response_time) as total_response_time, COUNT(*) as response_count`).
		Where("paused = ? AND response_time > 0 AND status = ?", false, "up").
		Scan(&responseStats)
	
	if responseStats.ResponseCount > 0 {
		avgResponseTime = int(responseStats.TotalResponseTime / responseStats.ResponseCount)
	}

	overallUptime := 0.0
	if unpausedCount > 0 {
		overallUptime = totalUptime / float64(unpausedCount)
	}

	newStats := StatsResponse{
		OverallUptime:   overallUptime,
		ServicesUp:      upCount,
		ServicesDown:    downCount,
		AvgResponseTime: avgResponseTime,
	}

	cachedStatsMu.Lock()
	cachedStats = newStats
	cachedStatsMu.Unlock()
}

// getStats returns the in-memory cached statistics
func getStats() StatsResponse {
	cachedStatsMu.RLock()
	defer cachedStatsMu.RUnlock()
	
	return cachedStats
}

