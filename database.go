package main

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// initDB initializes the database connection and runs migrations
func initDB() {
	var err error
	// Use pure Go SQLite driver (no CGO required)
	// Database path can be set via DB_PATH env var, defaults to ./nanostatus.db
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./nanostatus.db"
	}
	
	// Ensure the directory exists (for Docker volumes)
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Warn().Err(err).Str("directory", dir).Msg("Could not create database directory")
		}
	}
	
	// Enable WAL mode and configure connection pool for better concurrency
	// WAL mode allows multiple readers and one writer simultaneously
	// _busy_timeout sets how long to wait for locks (in milliseconds)
	// Performance pragmas:
	// - _synchronous=NORMAL: Balance safety and performance (faster than FULL)
	// - _cache_size=-64000: 64MB cache (negative = KB)
	// - _temp_store=MEMORY: Store temp tables in memory
	// - _mmap_size=268435456: 256MB memory-mapped I/O
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=1&_synchronous=NORMAL&_cache_size=-64000&_temp_store=MEMORY&_mmap_size=268435456"
	
	// Open database connection with modernc.org/sqlite
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}

	// Configure connection pool for SQLite (WAL mode supports concurrent readers)
	sqlDB.SetMaxOpenConns(15)
	sqlDB.SetMaxIdleConns(5)

	// Create GORM instance
	db, err = gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Auto-migrate schemas
	if err := db.AutoMigrate(&Monitor{}, &CheckHistory{}, &CheckHistoryBucket{}); err != nil {
		log.Fatal().Err(err).Msg("Failed to migrate database")
	}

	// Create partial index for active monitors (SQLite doesn't support partial indexes in GORM tags)
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_monitors_active 
		ON monitors(status, uptime) 
		WHERE paused = 0
	`).Error; err != nil {
		log.Warn().Err(err).Msg("Failed to create partial index for active monitors")
	}

	// Create unique index for check_history_buckets to ensure one bucket per monitor per hour
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_bucket_unique 
		ON check_history_buckets(monitor_id, bucket_hour)
	`).Error; err != nil {
		log.Warn().Err(err).Msg("Failed to create unique index for check_history_buckets")
	}

	// Create aggregation views
	if err := createAggregationViews(db); err != nil {
		log.Fatal().Err(err).Msg("Failed to create aggregation views")
	}

	log.Info().Str("path", dbPath).Msg("✅ Database initialized")

	// Always sync YAML config on startup (creates if empty, updates if changed)
	syncYAMLConfig(dbPath)
}

// syncYAMLConfig synchronizes monitors from YAML config with the database
// Looks for monitors.yaml in the same directory as the database
// Compares hashes to detect changes and updates monitors accordingly
func syncYAMLConfig(dbPath string) {
	// Look for monitors.yaml in the same directory as the database
	dbDir := filepath.Dir(dbPath)
	configPath := filepath.Join(dbDir, "monitors.yaml")
	
	// Try to load from YAML config file
	yamlMonitors, yamlHashes, err := loadMonitorsFromYAML(configPath)
	if err != nil {
		log.Warn().Err(err).Str("config_path", configPath).Msg("[Config] Failed to load YAML config")
	}
	
	// Get all existing monitors from database
	var existingMonitors []Monitor
	db.Find(&existingMonitors)
	
	// Create a map of existing monitors by config hash (only YAML-managed monitors)
	existingByHash := make(map[string]*Monitor)
	existingByID := make(map[uint]*Monitor)
	for i := range existingMonitors {
		existingByID[existingMonitors[i].ID] = &existingMonitors[i]
		if existingMonitors[i].ConfigHash != "" {
			existingByHash[existingMonitors[i].ConfigHash] = &existingMonitors[i]
		}
	}
	
	// If no YAML config found and database is empty, use defaults
	if len(yamlMonitors) == 0 {
		var count int64
		db.Model(&Monitor{}).Count(&count)
		if count == 0 {
			log.Info().Msg("[Config] No YAML config found and database is empty, seeding defaults...")
			defaultMonitors := []Monitor{
				{
					Name:         "Example.com",
					URL:          "https://example.com",
					Status:       "up",
					ResponseTime: 229,
					LastCheck:    "5s ago",
					CheckInterval: 60,
				},
				{
					Name:         "Google",
					URL:          "https://google.com",
					Status:       "up",
					ResponseTime: 2097,
					LastCheck:    "1s ago",
					IsThirdParty: true,
					CheckInterval: 60,
				},
			}
			for _, monitor := range defaultMonitors {
				if err := db.Create(&monitor).Error; err != nil {
					log.Error().Err(err).Str("monitor", monitor.Name).Msg("[Config] Failed to seed default monitor")
				} else {
					log.Info().Str("name", monitor.Name).Str("url", monitor.URL).Msg("[Config] Created default monitor")
					go checkService(&monitor)
				}
			}
		}
		return
	}
	
	log.Info().Int("count", len(yamlMonitors)).Msg("[Config] Syncing monitors from YAML configuration")
	
	// Track which YAML hashes we've processed
	processedHashes := make(map[string]bool)
	
	// Process each monitor from YAML
	for i, monitor := range yamlMonitors {
		hash := yamlHashes[i]
		processedHashes[hash] = true
		
		// Check if a monitor with this hash already exists
		if _, exists := existingByHash[hash]; exists {
			// Monitor exists with same hash - no changes needed
			log.Debug().Str("name", monitor.Name).Str("url", monitor.URL).Str("hash", hash[:8]).Msg("[Config] Monitor unchanged")
			continue
		}
		
		// Check if a monitor with same name/URL exists
		var existingMonitor Monitor
		result := db.Where("name = ? AND url = ?", monitor.Name, monitor.URL).First(&existingMonitor)
		
		// Check if record exists (ignore "record not found" error as it's expected)
		if result.Error == nil {
			// Monitor with same name/URL exists
			if existingMonitor.ConfigHash == "" {
				// Monitor was created via UI/API (no ConfigHash) - skip YAML version to avoid duplicates
				log.Debug().Str("name", monitor.Name).Str("url", monitor.URL).Msg("[Config] Skipping monitor - already exists (created via UI/API)")
				continue
			}
			
			// Monitor exists and is YAML-managed - check if hash changed
			if existingMonitor.ConfigHash == hash {
				// Hash matches - no update needed (shouldn't reach here due to existingByHash check, but just in case)
				continue
			}
			
			// Monitor exists but hash changed - update it
			log.Info().Str("name", monitor.Name).Str("url", monitor.URL).
				Str("old_hash", existingMonitor.ConfigHash[:8]).Str("new_hash", hash[:8]).
				Msg("[Config] Updating monitor - config changed")
			
			// Preserve runtime data (status, uptime, response time, last check)
			monitor.ID = existingMonitor.ID
			monitor.Status = existingMonitor.Status
			monitor.Uptime = existingMonitor.Uptime
			monitor.ResponseTime = existingMonitor.ResponseTime
			monitor.LastCheck = existingMonitor.LastCheck
			monitor.CreatedAt = existingMonitor.CreatedAt
			
			if err := db.Save(&monitor).Error; err != nil {
				log.Error().Err(err).Str("name", monitor.Name).Msg("[Config] Failed to update monitor")
			} else {
				log.Info().Str("name", monitor.Name).Str("url", monitor.URL).Msg("[Config] Updated monitor")
				broadcastUpdate("monitor_update", monitor)
				// Re-check if not paused
				if !monitor.Paused {
					go checkService(&monitor)
				}
			}
		} else {
			// New monitor - create it
			if err := db.Create(&monitor).Error; err != nil {
				log.Error().Err(err).Str("name", monitor.Name).Msg("[Config] Failed to create monitor")
			} else {
				log.Info().Str("name", monitor.Name).Str("url", monitor.URL).Str("hash", hash[:8]).Msg("[Config] Created monitor")
				broadcastUpdate("monitor_added", monitor)
				// Immediately check the monitor
				go checkService(&monitor)
			}
		}
	}
	
	// Remove monitors that were in YAML but are no longer present
	// Only remove monitors that have a config_hash (were created from YAML)
	for hash, existing := range existingByHash {
		if !processedHashes[hash] {
			log.Info().Str("name", existing.Name).Str("url", existing.URL).Str("hash", hash[:8]).
				Msg("[Config] Removing monitor - no longer in YAML config")
			
			monitorID := existing.ID
			if err := db.Delete(&Monitor{}, monitorID).Error; err != nil {
				log.Error().Err(err).Str("name", existing.Name).Msg("[Config] Failed to delete monitor")
			} else {
				log.Info().Str("name", existing.Name).Str("url", existing.URL).Msg("[Config] Deleted monitor")
				broadcastUpdate("monitor_deleted", map[string]interface{}{"id": monitorID})
			}
		}
	}
	
	broadcastStatsIfChanged()
	log.Info().Msg("[Config] ✅ YAML configuration synchronized")
}

// createAggregationViews creates SQL views for common aggregations
func createAggregationViews(db *gorm.DB) error {
	// View 1: monitor_stats_24h - Pre-aggregates 24-hour uptime stats per monitor
	view1SQL := `
		CREATE VIEW IF NOT EXISTS monitor_stats_24h AS
		SELECT 
			monitor_id,
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_checks,
			CAST(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) AS REAL) / COUNT(*) * 100.0 as uptime_percent,
			AVG(CASE WHEN status = 'up' AND response_time > 0 THEN response_time ELSE NULL END) as avg_response_time
		FROM check_histories
		WHERE created_at > datetime('now', '-24 hours')
		GROUP BY monitor_id
	`

	if err := db.Exec(view1SQL).Error; err != nil {
		return err
	}

	log.Info().Msg("✅ Created aggregation views")
	return nil
}

