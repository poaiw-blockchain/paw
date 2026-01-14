//go:build stress
// +build stress

package stress_test

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceSnapshot captures resource usage at a point in time
type ResourceSnapshot struct {
	Timestamp     time.Time
	GoRoutines    int
	HeapAllocMB   float64
	HeapSysMB     float64
	TotalAllocMB  float64
	NumGC         uint32
	PauseTotalNs  uint64
	CPUPercent    float64
	NumCPU        int
	StackInUseMB  float64
	MSpanInUseMB  float64
	MCacheInUseMB float64
}

// ResourceMonitor monitors system resources during stress tests
type ResourceMonitor struct {
	snapshots       []ResourceSnapshot
	mu              sync.RWMutex
	stopChan        chan struct{}
	stopped         atomic.Bool
	interval        time.Duration
	maxSnapshots    int
	warningCallback func(string)
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(interval time.Duration, maxSnapshots int) *ResourceMonitor {
	return &ResourceMonitor{
		snapshots:    make([]ResourceSnapshot, 0, maxSnapshots),
		stopChan:     make(chan struct{}),
		interval:     interval,
		maxSnapshots: maxSnapshots,
	}
}

// SetWarningCallback sets a callback for resource warnings
func (rm *ResourceMonitor) SetWarningCallback(cb func(string)) {
	rm.warningCallback = cb
}

// Start begins monitoring resources
func (rm *ResourceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rm.stopped.Store(true)
			return
		case <-rm.stopChan:
			rm.stopped.Store(true)
			return
		case <-ticker.C:
			snapshot := rm.captureSnapshot()
			rm.addSnapshot(snapshot)
			rm.checkThresholds(snapshot)
		}
	}
}

// Stop stops monitoring
func (rm *ResourceMonitor) Stop() {
	if !rm.stopped.Load() {
		close(rm.stopChan)
	}
}

// captureSnapshot captures current resource usage
func (rm *ResourceMonitor) captureSnapshot() ResourceSnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ResourceSnapshot{
		Timestamp:     time.Now(),
		GoRoutines:    runtime.NumGoroutine(),
		HeapAllocMB:   float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapSysMB:     float64(memStats.HeapSys) / 1024 / 1024,
		TotalAllocMB:  float64(memStats.TotalAlloc) / 1024 / 1024,
		NumGC:         memStats.NumGC,
		PauseTotalNs:  memStats.PauseTotalNs,
		NumCPU:        runtime.NumCPU(),
		StackInUseMB:  float64(memStats.StackInuse) / 1024 / 1024,
		MSpanInUseMB:  float64(memStats.MSpanInuse) / 1024 / 1024,
		MCacheInUseMB: float64(memStats.MCacheInuse) / 1024 / 1024,
	}
}

// addSnapshot adds a snapshot to the history
func (rm *ResourceMonitor) addSnapshot(snapshot ResourceSnapshot) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.snapshots = append(rm.snapshots, snapshot)

	// Keep only the most recent snapshots
	if len(rm.snapshots) > rm.maxSnapshots {
		rm.snapshots = rm.snapshots[len(rm.snapshots)-rm.maxSnapshots:]
	}
}

// checkThresholds checks for resource issues
func (rm *ResourceMonitor) checkThresholds(snapshot ResourceSnapshot) {
	if rm.warningCallback == nil {
		return
	}

	// Check for excessive goroutines (>10000)
	if snapshot.GoRoutines > 10000 {
		rm.warningCallback(fmt.Sprintf("High goroutine count: %d", snapshot.GoRoutines))
	}

	// Check for high memory usage (>1GB heap)
	if snapshot.HeapAllocMB > 1024 {
		rm.warningCallback(fmt.Sprintf("High heap allocation: %.2f MB", snapshot.HeapAllocMB))
	}

	// Check for memory leak indicators
	if len(rm.snapshots) > 10 {
		rm.detectMemoryLeak()
	}
}

// detectMemoryLeak analyzes snapshots for memory leak patterns
func (rm *ResourceMonitor) detectMemoryLeak() {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if len(rm.snapshots) < 10 {
		return
	}

	// Compare last 10 snapshots
	recent := rm.snapshots[len(rm.snapshots)-10:]
	firstHeap := recent[0].HeapAllocMB
	lastHeap := recent[len(recent)-1].HeapAllocMB

	// If heap grows >50% over 10 samples without significant GC, warn
	growth := (lastHeap - firstHeap) / firstHeap * 100
	if growth > 50 && rm.warningCallback != nil {
		rm.warningCallback(fmt.Sprintf("Potential memory leak detected: %.2f%% growth", growth))
	}
}

// GetSnapshots returns all captured snapshots
func (rm *ResourceMonitor) GetSnapshots() []ResourceSnapshot {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	result := make([]ResourceSnapshot, len(rm.snapshots))
	copy(result, rm.snapshots)
	return result
}

// GetLatest returns the most recent snapshot
func (rm *ResourceMonitor) GetLatest() *ResourceSnapshot {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if len(rm.snapshots) == 0 {
		return nil
	}

	snapshot := rm.snapshots[len(rm.snapshots)-1]
	return &snapshot
}

// GetStats returns summary statistics
func (rm *ResourceMonitor) GetStats() ResourceStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if len(rm.snapshots) == 0 {
		return ResourceStats{}
	}

	var (
		totalGoroutines float64
		totalHeap       float64
		maxGoroutines   int
		maxHeap         float64
		minGoroutines   int = int(^uint(0) >> 1) // Max int
		minHeap         float64
	)

	first := rm.snapshots[0]
	last := rm.snapshots[len(rm.snapshots)-1]

	for _, s := range rm.snapshots {
		totalGoroutines += float64(s.GoRoutines)
		totalHeap += s.HeapAllocMB

		if s.GoRoutines > maxGoroutines {
			maxGoroutines = s.GoRoutines
		}
		if s.GoRoutines < minGoroutines {
			minGoroutines = s.GoRoutines
		}
		if s.HeapAllocMB > maxHeap {
			maxHeap = s.HeapAllocMB
		}
		if minHeap == 0 || s.HeapAllocMB < minHeap {
			minHeap = s.HeapAllocMB
		}
	}

	count := float64(len(rm.snapshots))

	return ResourceStats{
		Duration:          last.Timestamp.Sub(first.Timestamp),
		SampleCount:       len(rm.snapshots),
		AvgGoroutines:     totalGoroutines / count,
		MaxGoroutines:     maxGoroutines,
		MinGoroutines:     minGoroutines,
		AvgHeapMB:         totalHeap / count,
		MaxHeapMB:         maxHeap,
		MinHeapMB:         minHeap,
		HeapGrowthMB:      last.HeapAllocMB - first.HeapAllocMB,
		TotalGCCount:      last.NumGC - first.NumGC,
		TotalGCPauseMs:    float64(last.PauseTotalNs-first.PauseTotalNs) / 1e6,
		FinalGoroutines:   last.GoRoutines,
		FinalHeapMB:       last.HeapAllocMB,
		InitialGoroutines: first.GoRoutines,
		InitialHeapMB:     first.HeapAllocMB,
	}
}

// ResourceStats contains summary statistics
type ResourceStats struct {
	Duration          time.Duration
	SampleCount       int
	AvgGoroutines     float64
	MaxGoroutines     int
	MinGoroutines     int
	AvgHeapMB         float64
	MaxHeapMB         float64
	MinHeapMB         float64
	HeapGrowthMB      float64
	TotalGCCount      uint32
	TotalGCPauseMs    float64
	FinalGoroutines   int
	FinalHeapMB       float64
	InitialGoroutines int
	InitialHeapMB     float64
}

// String formats resource stats for display
func (rs ResourceStats) String() string {
	return fmt.Sprintf(`Resource Statistics (Duration: %v, Samples: %d):
  Goroutines: avg=%.0f, min=%d, max=%d, final=%d (delta=%+d)
  Heap Memory: avg=%.2f MB, min=%.2f MB, max=%.2f MB, final=%.2f MB (growth=%+.2f MB)
  GC: count=%d, total_pause=%.2f ms (avg=%.2f ms/gc)`,
		rs.Duration,
		rs.SampleCount,
		rs.AvgGoroutines,
		rs.MinGoroutines,
		rs.MaxGoroutines,
		rs.FinalGoroutines,
		rs.FinalGoroutines-rs.InitialGoroutines,
		rs.AvgHeapMB,
		rs.MinHeapMB,
		rs.MaxHeapMB,
		rs.FinalHeapMB,
		rs.HeapGrowthMB,
		rs.TotalGCCount,
		rs.TotalGCPauseMs,
		func() float64 {
			if rs.TotalGCCount > 0 {
				return rs.TotalGCPauseMs / float64(rs.TotalGCCount)
			}
			return 0
		}(),
	)
}

// ForceGC triggers garbage collection and waits for it to complete
func ForceGC() {
	runtime.GC()
	debug.FreeOSMemory()
	time.Sleep(100 * time.Millisecond) // Allow GC to complete
}

// GetBaselineSnapshot captures a baseline snapshot after GC
func GetBaselineSnapshot() ResourceSnapshot {
	ForceGC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ResourceSnapshot{
		Timestamp:     time.Now(),
		GoRoutines:    runtime.NumGoroutine(),
		HeapAllocMB:   float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapSysMB:     float64(memStats.HeapSys) / 1024 / 1024,
		TotalAllocMB:  float64(memStats.TotalAlloc) / 1024 / 1024,
		NumGC:         memStats.NumGC,
		PauseTotalNs:  memStats.PauseTotalNs,
		NumCPU:        runtime.NumCPU(),
		StackInUseMB:  float64(memStats.StackInuse) / 1024 / 1024,
		MSpanInUseMB:  float64(memStats.MSpanInuse) / 1024 / 1024,
		MCacheInUseMB: float64(memStats.MCacheInuse) / 1024 / 1024,
	}
}
