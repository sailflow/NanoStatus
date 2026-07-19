package main

import "time"

// Monitor represents a service being monitored
type Monitor struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	URL          string    `gorm:"not null" json:"url"`
	Uptime       float64   `gorm:"default:0" json:"uptime"`
	Status       string    `gorm:"default:unknown;index:idx_paused_status" json:"status"`
	ResponseTime int       `gorm:"default:0" json:"responseTime"`
	LastCheck    string    `gorm:"default:never" json:"lastCheck"`
	IsThirdParty bool      `gorm:"default:false" json:"isThirdParty,omitempty"`
	Icon         string    `json:"icon,omitempty"`
	CheckInterval int      `gorm:"default:60" json:"checkInterval"` // Interval in seconds
	Paused       bool      `gorm:"default:false;index:idx_paused_status" json:"paused"` // Whether monitoring is paused
	// Note: Partial index idx_monitors_active on (Status, Uptime) WHERE paused = 0 will be created via raw SQL
	ConfigHash   string    `gorm:"index" json:"configHash,omitempty"` // Hash of YAML config (empty if created via UI/API)
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CreateMonitorRequest represents the request body for creating a monitor
type CreateMonitorRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	IsThirdParty bool   `json:"isThirdParty,omitempty"`
	Icon         string `json:"icon,omitempty"`
	CheckInterval int   `json:"checkInterval,omitempty"` // Interval in seconds (default: 60)
}

// StatsResponse represents overall statistics
type StatsResponse struct {
	OverallUptime   float64 `json:"overallUptime"`
	ServicesUp      int     `json:"servicesUp"`
	ServicesDown    int     `json:"servicesDown"`
	AvgResponseTime int     `json:"avgResponseTime"`
}

// CheckHistory stores historical check data
type CheckHistory struct {
	ID           uint      `gorm:"primaryKey"`
	MonitorID    uint      `gorm:"not null;index:idx_monitor_created"`
	Status       string    `gorm:"not null"`
	ResponseTime int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"index:idx_monitor_created;index:idx_created_at"`
}

// CheckHistoryBucket stores aggregated hourly buckets of check history for older data
type CheckHistoryBucket struct {
	ID             uint      `gorm:"primaryKey"`
	MonitorID      uint      `gorm:"not null;index:idx_bucket_monitor_hour;uniqueIndex:idx_bucket_unique"`
	BucketHour     int64     `gorm:"not null;index:idx_bucket_monitor_hour;uniqueIndex:idx_bucket_unique"` // Unix timestamp rounded to hour
	TotalChecks    int       `gorm:"default:0"`
	UpChecks       int       `gorm:"default:0"`
	AvgResponseTime float64   `gorm:"default:0"`
	MinResponseTime int       `gorm:"default:0"`
	MaxResponseTime int       `gorm:"default:0"`
	CreatedAt      time.Time
}

// ResponseTimeData represents formatted response time data for charts
type ResponseTimeData struct {
	Time         string  `json:"time"`         // Formatted time string (for display)
	Timestamp    string  `json:"timestamp"`    // ISO 8601 timestamp (for client-side formatting)
	ResponseTime float64 `json:"responseTime"`
}

