package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	cachedStats StatsResponse
	statsMu     sync.RWMutex
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
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	
	var avgResult sql.NullFloat64
	var countResult int64
	
	db.Model(&CheckHistory{}).
		Where("created_at > ? AND response_time > 0 AND status = ?", twentyFourHoursAgo, "up").
		Count(&countResult)
	
	if countResult > 0 {
		err := db.Raw(`
			SELECT AVG(response_time) as avg_response_time 
			FROM check_histories 
			WHERE created_at > ? AND response_time > 0 AND status = ?
		`, twentyFourHoursAgo, "up").Row().Scan(&avgResult)
		
		if err == nil && avgResult.Valid {
			avgResponseTime = int(avgResult.Float64)
		}
	}
	
	if countResult == 0 || avgResponseTime == 0 {
		var fallbackStats struct {
			TotalResponseTime int64
			ResponseCount      int64
		}
		db.Model(&Monitor{}).
			Select(`SUM(response_time) as total_response_time, COUNT(*) as response_count`).
			Where("paused = ? AND response_time > 0 AND status = ?", false, "up").
			Scan(&fallbackStats)
		
		if fallbackStats.ResponseCount > 0 {
			avgResponseTime = int(fallbackStats.TotalResponseTime / fallbackStats.ResponseCount)
		}
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

	statsMu.Lock()
	cachedStats = newStats
	statsMu.Unlock()
}

// getStats returns the in-memory cached statistics
func getStats() StatsResponse {
	statsMu.RLock()
	defer statsMu.RUnlock()
	
	return cachedStats
}

