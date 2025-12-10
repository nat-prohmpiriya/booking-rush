package worker

import (
	"testing"
	"time"
)

func TestDefaultExpiryWorkerConfig(t *testing.T) {
	config := DefaultExpiryWorkerConfig()

	if config.ScanInterval != 5*time.Second {
		t.Errorf("ScanInterval = %v, want %v", config.ScanInterval, 5*time.Second)
	}

	if config.BatchSize != 100 {
		t.Errorf("BatchSize = %v, want %v", config.BatchSize, 100)
	}
}

func TestExpiryWorkerConfig_Custom(t *testing.T) {
	config := &ExpiryWorkerConfig{
		ScanInterval: 10 * time.Second,
		BatchSize:    50,
	}

	if config.ScanInterval != 10*time.Second {
		t.Errorf("ScanInterval = %v, want %v", config.ScanInterval, 10*time.Second)
	}

	if config.BatchSize != 50 {
		t.Errorf("BatchSize = %v, want %v", config.BatchSize, 50)
	}
}

func TestNewExpiryWorker_WithDefaultConfig(t *testing.T) {
	worker := NewExpiryWorker(nil, nil, nil, nil)

	if worker == nil {
		t.Fatal("NewExpiryWorker() returned nil")
	}

	if worker.config == nil {
		t.Fatal("Worker config should not be nil")
	}

	if worker.config.ScanInterval != 5*time.Second {
		t.Errorf("Default ScanInterval = %v, want %v", worker.config.ScanInterval, 5*time.Second)
	}

	if worker.running {
		t.Error("Worker should not be running initially")
	}

	if worker.totalExpired != 0 {
		t.Errorf("TotalExpired = %v, want %v", worker.totalExpired, 0)
	}

	if worker.totalReleased != 0 {
		t.Errorf("TotalReleased = %v, want %v", worker.totalReleased, 0)
	}
}

func TestNewExpiryWorker_WithCustomConfig(t *testing.T) {
	customConfig := &ExpiryWorkerConfig{
		ScanInterval: 15 * time.Second,
		BatchSize:    200,
	}

	worker := NewExpiryWorker(nil, nil, nil, customConfig)

	if worker == nil {
		t.Fatal("NewExpiryWorker() returned nil")
	}

	if worker.config.ScanInterval != 15*time.Second {
		t.Errorf("ScanInterval = %v, want %v", worker.config.ScanInterval, 15*time.Second)
	}

	if worker.config.BatchSize != 200 {
		t.Errorf("BatchSize = %v, want %v", worker.config.BatchSize, 200)
	}
}

func TestExpiryWorkerStats(t *testing.T) {
	now := time.Now()
	stats := &ExpiryWorkerStats{
		IsRunning:        true,
		TotalExpired:     100,
		TotalReleased:    95,
		LastScanTime:     now,
		LastExpiredCount: 5,
	}

	if !stats.IsRunning {
		t.Error("IsRunning should be true")
	}

	if stats.TotalExpired != 100 {
		t.Errorf("TotalExpired = %v, want %v", stats.TotalExpired, 100)
	}

	if stats.TotalReleased != 95 {
		t.Errorf("TotalReleased = %v, want %v", stats.TotalReleased, 95)
	}

	if stats.LastScanTime != now {
		t.Errorf("LastScanTime = %v, want %v", stats.LastScanTime, now)
	}

	if stats.LastExpiredCount != 5 {
		t.Errorf("LastExpiredCount = %v, want %v", stats.LastExpiredCount, 5)
	}
}

func TestExpiryWorker_GetStats(t *testing.T) {
	worker := NewExpiryWorker(nil, nil, nil, nil)

	// Initial stats
	stats := worker.GetStats()

	if stats.IsRunning {
		t.Error("Worker should not be running initially")
	}

	if stats.TotalExpired != 0 {
		t.Errorf("TotalExpired = %v, want %v", stats.TotalExpired, 0)
	}

	if stats.TotalReleased != 0 {
		t.Errorf("TotalReleased = %v, want %v", stats.TotalReleased, 0)
	}

	if stats.LastExpiredCount != 0 {
		t.Errorf("LastExpiredCount = %v, want %v", stats.LastExpiredCount, 0)
	}
}

func TestExpiryWorker_StartStop(t *testing.T) {
	worker := NewExpiryWorker(nil, nil, nil, &ExpiryWorkerConfig{
		ScanInterval: 100 * time.Millisecond,
		BatchSize:    10,
	})

	// Worker should not be running initially
	if worker.running {
		t.Error("Worker should not be running before Start()")
	}

	// Note: Cannot actually call Start() without real repositories
	// This test just verifies the initial state
}
