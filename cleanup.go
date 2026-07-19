package main

import (
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// cleanOldCheckHistory removes check history records older than 1 year
func cleanOldCheckHistory() {
	oneYearAgo := time.Now().Add(-365 * 24 * time.Hour)
	
	log.Info().Time("cutoff", oneYearAgo).Msg("[Cleanup] Starting cleanup of check history")
	
	var deletedCount int64
	result := db.Where("created_at < ?", oneYearAgo).Delete(&CheckHistory{})
	
	if result.Error != nil {
		log.Error().Err(result.Error).Msg("[Cleanup] Failed to clean old check history")
		return
	}
	
	deletedCount = result.RowsAffected
	log.Info().Int64("deleted", deletedCount).Msg("[Cleanup] Successfully deleted check history records")
}

// bucketOldCheckHistory aggregates CheckHistory records older than 24 hours into hourly buckets
// Uses SQL aggregation for maximum efficiency instead of loading data into Go
func bucketOldCheckHistory() {
	cutoffTime := time.Now().Add(-24 * time.Hour)
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	
	log.Info().Time("cutoff", cutoffTime).Msg("[Bucketing] Starting check history bucketing")
	
	// Use SQL to aggregate all checks into buckets in a single query
	// This is much more efficient than loading all checks into memory
	// SQLite's strftime can truncate datetime to hour: strftime('%Y-%m-%d %H:00:00', datetime)
	// Then convert to unix timestamp: unixepoch(strftime('%Y-%m-%d %H:00:00', created_at))
	
	// Aggregate checks into buckets using SQL
	// This single query replaces all the Go-based grouping and aggregation
	var aggregatedBuckets []struct {
		MonitorID       uint
		BucketHour      int64
		TotalChecks     int
		UpChecks        int
		AvgResponseTime float64
		MinResponseTime int
		MaxResponseTime int
	}
	
	// GORM stores datetime as text with format: "2026-01-12 05:29:47.500629789 +0000 UTC..."
	// Extract date and hour (first 13 chars: "2026-01-12 05"), then convert to unix timestamp
	// Use substr to get "YYYY-MM-DD HH" format, then use datetime() to parse and convert
	err := db.Raw(`
		SELECT 
			monitor_id,
			CAST(unixepoch(substr(created_at, 1, 13) || ':00:00') AS INTEGER) as bucket_hour,
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_checks,
			AVG(CASE WHEN response_time > 0 THEN response_time ELSE NULL END) as avg_response_time,
			MIN(CASE WHEN response_time > 0 THEN response_time ELSE NULL END) as min_response_time,
			MAX(CASE WHEN response_time > 0 THEN response_time ELSE NULL END) as max_response_time
		FROM check_histories
		WHERE created_at < ? AND created_at >= ?
		GROUP BY monitor_id, bucket_hour
		ORDER BY monitor_id, bucket_hour
	`, cutoffTime, sevenDaysAgo).Scan(&aggregatedBuckets).Error
	
	if err != nil {
		log.Error().Err(err).Msg("[Bucketing] Failed to aggregate checks into buckets")
		return
	}
	
	if len(aggregatedBuckets) == 0 {
		log.Info().Msg("[Bucketing] No old checks to bucket")
		// Still try to delete old raw records
		result := db.Where("created_at < ?", sevenDaysAgo).Delete(&CheckHistory{})
		if result.Error != nil {
			log.Error().Err(result.Error).Msg("[Bucketing] Failed to delete old raw records")
		} else {
			log.Debug().Int64("deleted", result.RowsAffected).Msg("[Bucketing] Deleted old raw records")
		}
		return
	}
	
	log.Info().Int("buckets", len(aggregatedBuckets)).Msg("[Bucketing] Aggregated checks into buckets")
	
	// Batch upsert buckets in transactions (500 per transaction for better SQLite performance)
	const batchSize = 500
	totalBucketed := 0
	
	for i := 0; i < len(aggregatedBuckets); i += batchSize {
		end := i + batchSize
		if end > len(aggregatedBuckets) {
			end = len(aggregatedBuckets)
		}
		batch := aggregatedBuckets[i:end]
		
		// Upsert batch in transaction using INSERT ... ON CONFLICT
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, agg := range batch {
				// Handle NULL values from SQL (MIN/MAX can be NULL if no response_time > 0)
				minRT := agg.MinResponseTime
				maxRT := agg.MaxResponseTime
				if minRT == 0 {
					minRT = 0 // Keep as 0 if NULL
				}
				if maxRT == 0 {
					maxRT = 0 // Keep as 0 if NULL
				}
				
				// Use INSERT ... ON CONFLICT UPDATE (UPSERT) for SQLite
				// This is more efficient than separate SELECT + UPDATE/INSERT
				upsertSQL := `
					INSERT INTO check_history_buckets 
						(monitor_id, bucket_hour, total_checks, up_checks, avg_response_time, min_response_time, max_response_time, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT(monitor_id, bucket_hour) DO UPDATE SET
						total_checks = excluded.total_checks,
						up_checks = excluded.up_checks,
						avg_response_time = excluded.avg_response_time,
						min_response_time = excluded.min_response_time,
						max_response_time = excluded.max_response_time
				`
				
				if err := tx.Exec(upsertSQL,
					agg.MonitorID,
					agg.BucketHour,
					agg.TotalChecks,
					agg.UpChecks,
					agg.AvgResponseTime,
					minRT,
					maxRT,
					time.Now(),
				).Error; err != nil {
					return err
				}
			}
			return nil
		})
		
		if err != nil {
			log.Error().Err(err).Int("batch_start", i).Int("batch_end", end).
				Msg("[Bucketing] Failed to upsert bucket batch")
			continue
		}
		
		totalBucketed += len(batch)
	}
	
	// Delete old raw records after bucketing (keep last 7 days raw for detailed charts)
	result := db.Where("created_at < ?", sevenDaysAgo).Delete(&CheckHistory{})
	
	if result.Error != nil {
		log.Error().Err(result.Error).Msg("[Bucketing] Failed to delete old raw records")
	} else {
		log.Debug().Int64("deleted", result.RowsAffected).Msg("[Bucketing] Deleted old raw records")
	}
	
	log.Info().Int("buckets_created", totalBucketed).Msg("[Bucketing] Completed check history bucketing")
}

var cleanupScheduler gocron.Scheduler

// startCleanupScheduler starts a background job that runs cleanup daily at midnight using gocron
func startCleanupScheduler() {
	// Create a new scheduler for cleanup jobs
	sched, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal().Err(err).Msg("[Cleanup] Failed to create cleanup scheduler")
	}
	
	cleanupScheduler = sched
	
	// Schedule cleanup job to run daily at midnight (00:00)
	// Cron expression: "0 0 * * *" means: minute=0, hour=0, every day, every month, every weekday
	_, err = cleanupScheduler.NewJob(
		gocron.CronJob("0 0 * * *", false),
		gocron.NewTask(func() {
			log.Info().Msg("[Cleanup] Running scheduled cleanup and bucketing")
			cleanOldCheckHistory()
			bucketOldCheckHistory()
		}),
		gocron.WithName("daily-cleanup"),
	)
	
	if err != nil {
		log.Fatal().Err(err).Msg("[Cleanup] Failed to schedule cleanup job")
	}
	
	// Start the scheduler
	cleanupScheduler.Start()
	log.Info().Msg("[Cleanup] Cleanup scheduler started - will run daily at midnight")
}

