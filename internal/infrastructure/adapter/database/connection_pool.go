package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"gorm.io/gorm"
)

// ConnectionPoolMetrics tracks database connection pool metrics
type ConnectionPoolMetrics struct {
	OpenConnections    int
	IdleConnections    int
	MaxOpenConnections int
	InUse              int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
}

// ConnectionPoolMonitor monitors the database connection pool
type ConnectionPoolMonitor struct {
	db           *Manager
	logger       coreport.Logger
	metricsCache *ConnectionPoolMetrics
	mutex        sync.RWMutex
	stopChan     chan struct{}
}

// NewConnectionPoolMonitor creates a new connection pool monitor
func NewConnectionPoolMonitor(db *Manager, logger coreport.Logger) *ConnectionPoolMonitor {
	return &ConnectionPoolMonitor{
		db:       db,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start begins monitoring the connection pool
func (m *ConnectionPoolMonitor) Start(interval time.Duration) error {
	ticker := time.NewTicker(interval)

	// Collect metrics initially
	if err := m.collectMetrics(); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := m.collectMetrics(); err != nil {
					m.logger.Error("Failed to collect connection pool metrics", map[string]any{
						"error": err.Error(),
					})
				}
			case <-m.stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// Stop stops the monitoring
func (m *ConnectionPoolMonitor) Stop() {
	close(m.stopChan)
}

// GetMetrics returns the current connection pool metrics
func (m *ConnectionPoolMonitor) GetMetrics() ConnectionPoolMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.metricsCache == nil {
		return ConnectionPoolMetrics{}
	}

	return *m.metricsCache
}

// collectMetrics collects current connection pool metrics
func (m *ConnectionPoolMonitor) collectMetrics() error {
	sqlDB, err := m.db.DB().DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	stats := sqlDB.Stats()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metricsCache = &ConnectionPoolMetrics{
		OpenConnections:    stats.OpenConnections,
		IdleConnections:    stats.Idle,
		MaxOpenConnections: stats.MaxOpenConnections,
		InUse:              stats.InUse,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}

	// Log metrics if too many connections are in use
	threshold := float64(stats.MaxOpenConnections) * 0.8
	if float64(stats.InUse) > threshold {
		m.logger.Warn("Database connection pool nearly exhausted", map[string]any{
			"in_use":     stats.InUse,
			"max_open":   stats.MaxOpenConnections,
			"idle":       stats.Idle,
			"wait_count": stats.WaitCount,
			"wait_time":  stats.WaitDuration.String(),
		})
	}

	return nil
}

// ConnectionPool manages and monitors database connections
type ConnectionPool struct {
	db            *gorm.DB
	logger        coreport.Logger
	timeProvider  coreport.TimeProvider
	healthChecker *HealthChecker
}

// NewConnectionPool creates a new connection pool with monitoring
func NewConnectionPool(db *gorm.DB, logger coreport.Logger, timeProvider coreport.TimeProvider) *ConnectionPool {
	pool := &ConnectionPool{
		db:           db,
		logger:       logger,
		timeProvider: timeProvider,
	}

	// Initialize health checker
	pool.healthChecker = NewHealthChecker(db, logger, timeProvider)
	pool.healthChecker.StartMonitoring()

	return pool
}

// GetDB returns the database connection
func (p *ConnectionPool) GetDB() *gorm.DB {
	return p.db
}

// Close closes the connection pool
func (p *ConnectionPool) Close() {
	if p.healthChecker != nil {
		p.healthChecker.StopMonitoring()
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		p.logger.Error("Failed to get SQL DB instance", map[string]any{
			"error": err.Error(),
		})
		return
	}

	if err := sqlDB.Close(); err != nil {
		p.logger.Error("Failed to close database connection", map[string]any{
			"error": err.Error(),
		})
	}
}

// HealthChecker monitors database connection health
type HealthChecker struct {
	db           *gorm.DB
	logger       coreport.Logger
	timeProvider coreport.TimeProvider
	stopChan     chan struct{}
	checkPeriod  time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *gorm.DB, logger coreport.Logger, timeProvider coreport.TimeProvider) *HealthChecker {
	return &HealthChecker{
		db:           db,
		logger:       logger,
		timeProvider: timeProvider,
		stopChan:     make(chan struct{}),
		checkPeriod:  30 * time.Second, // Check every 30 seconds
	}
}

// StartMonitoring starts the health monitoring goroutine
func (h *HealthChecker) StartMonitoring() {
	go h.monitorHealth()
}

// StopMonitoring stops the health monitoring goroutine
func (h *HealthChecker) StopMonitoring() {
	close(h.stopChan)
}

// monitorHealth periodically checks database health and logs metrics
func (h *HealthChecker) monitorHealth() {
	ticker := time.NewTicker(h.checkPeriod)
	defer ticker.Stop()

	h.logger.Info("Database health monitoring started", nil)

	for {
		select {
		case <-h.stopChan:
			h.logger.Info("Database health monitoring stopped", nil)
			return
		case <-ticker.C:
			h.checkHealth()
		}
	}
}

// checkHealth performs health check and logs metrics
func (h *HealthChecker) checkHealth() {
	sqlDB, err := h.db.DB()
	if err != nil {
		h.logger.Error("Failed to get SQL DB instance during health check", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Check if database is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		h.logger.Error("Database ping failed", map[string]any{
			"error": err.Error(),
		})
	}

	// Log connection pool stats
	stats := sqlDB.Stats()
	h.logger.Info("Database connection pool stats", map[string]any{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration_ms":     stats.WaitDuration.Milliseconds(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	})
}
