package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceMonitor monitors system resources and API usage
type ResourceMonitor struct {
	maxMemoryMB    int64
	maxCPUPercent  int
	maxCallsPerMin int
	
	// Counters
	apiCalls       int64
	apiCallsWindow []time.Time
	totalCost      float64
	maxCostPerHour float64
	
	mu sync.RWMutex
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(maxMemoryMB int64, maxCPUPercent int, maxCallsPerMin int) *ResourceMonitor {
	return &ResourceMonitor{
		maxMemoryMB:    maxMemoryMB,
		maxCPUPercent:  maxCPUPercent,
		maxCallsPerMin: maxCallsPerMin,
		apiCallsWindow: make([]time.Time, 0),
	}
}

// SetCostLimit sets the maximum cost per hour
func (m *ResourceMonitor) SetCostLimit(maxCostPerHour float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxCostPerHour = maxCostPerHour
}

// RecordAPICall records an API call
func (m *ResourceMonitor) RecordAPICall(cost float64) {
	atomic.AddInt64(&m.apiCalls, 1)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	m.apiCallsWindow = append(m.apiCallsWindow, now)
	m.totalCost += cost
	
	// Clean old entries (older than 1 hour)
	cutoff := now.Add(-time.Hour)
	newWindow := make([]time.Time, 0)
	for _, t := range m.apiCallsWindow {
		if t.After(cutoff) {
			newWindow = append(newWindow, t)
		}
	}
	m.apiCallsWindow = newWindow
}

// CanMakeAPICall checks if we can make another API call
func (m *ResourceMonitor) CanMakeAPICall() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check rate limit
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	recentCalls := 0
	for _, t := range m.apiCallsWindow {
		if t.After(cutoff) {
			recentCalls++
		}
	}
	
	if m.maxCallsPerMin > 0 && recentCalls >= m.maxCallsPerMin {
		return false
	}
	
	// Check cost limit
	if m.maxCostPerHour > 0 && m.totalCost >= m.maxCostPerHour {
		return false
	}
	
	return true
}

// WaitForAPISlot waits until an API call can be made
func (m *ResourceMonitor) WaitForAPISlot(ctx context.Context) error {
	for !m.CanMakeAPICall() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			// Check again after 1 second
		}
	}
	return nil
}

// CheckMemory checks if memory usage is acceptable
func (m *ResourceMonitor) CheckMemory() bool {
	if m.maxMemoryMB <= 0 {
		return true
	}
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	usedMB := int64(memStats.Alloc / 1024 / 1024)
	return usedMB < m.maxMemoryMB
}

// GetMemoryUsageMB returns current memory usage in MB
func (m *ResourceMonitor) GetMemoryUsageMB() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.Alloc / 1024 / 1024)
}

// CanStartWorker checks if we have resources to start a new worker
func (m *ResourceMonitor) CanStartWorker() bool {
	return m.CheckMemory() && m.CanMakeAPICall()
}

// WaitForResources waits until resources are available
func (m *ResourceMonitor) WaitForResources(ctx context.Context) error {
	for !m.CanStartWorker() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			// Check again
		}
	}
	return nil
}

// GetStats returns current resource statistics
func (m *ResourceMonitor) GetStats() ResourceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	recentCalls := 0
	for _, t := range m.apiCallsWindow {
		if t.After(cutoff) {
			recentCalls++
		}
	}
	
	return ResourceStats{
		TotalAPICalls:    atomic.LoadInt64(&m.apiCalls),
		CallsPerMinute:   recentCalls,
		TotalCost:        m.totalCost,
		MemoryUsageMB:    m.GetMemoryUsageMB(),
		MaxMemoryMB:      m.maxMemoryMB,
		MaxCallsPerMin:   m.maxCallsPerMin,
		MaxCostPerHour:   m.maxCostPerHour,
	}
}

// ResourceStats contains resource usage statistics
type ResourceStats struct {
	TotalAPICalls   int64
	CallsPerMinute  int
	TotalCost       float64
	MemoryUsageMB   int64
	MaxMemoryMB     int64
	MaxCallsPerMin  int
	MaxCostPerHour  float64
}

// Print prints resource statistics
func (s ResourceStats) Print() {
	fmt.Println("\n📊 Resource Statistics")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("API Calls: %d total, %d/min\n", s.TotalAPICalls, s.CallsPerMinute)
	if s.MaxCallsPerMin > 0 {
		fmt.Printf("Rate Limit: %d calls/min (%.1f%% used)\n", 
			s.MaxCallsPerMin, float64(s.CallsPerMinute)/float64(s.MaxCallsPerMin)*100)
	}
	fmt.Printf("Memory: %d MB", s.MemoryUsageMB)
	if s.MaxMemoryMB > 0 {
		fmt.Printf(" / %d MB (%.1f%%)", s.MaxMemoryMB, float64(s.MemoryUsageMB)/float64(s.MaxMemoryMB)*100)
	}
	fmt.Println()
	if s.TotalCost > 0 {
		fmt.Printf("Cost: $%.4f", s.TotalCost)
		if s.MaxCostPerHour > 0 {
			fmt.Printf(" / $%.2f/hr (%.1f%%)", s.MaxCostPerHour, s.TotalCost/s.MaxCostPerHour*100)
		}
		fmt.Println()
	}
	fmt.Println("═══════════════════════════════════════")
}

// RateLimiter provides token bucket rate limiting
type RateLimiter struct {
	rate       float64 // tokens per second
	maxTokens  float64
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(callsPerMinute int) *RateLimiter {
	rate := float64(callsPerMinute) / 60.0
	return &RateLimiter{
		rate:       rate,
		maxTokens:  float64(callsPerMinute),
		tokens:     float64(callsPerMinute),
		lastUpdate: time.Now(),
	}
}

// Acquire blocks until a token is available
func (r *RateLimiter) Acquire(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(r.lastUpdate).Seconds()
		r.tokens += elapsed * r.rate
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastUpdate = now
		
		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}
		r.mu.Unlock()
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Try again
		}
	}
}

// TryAcquire attempts to acquire a token without blocking
func (r *RateLimiter) TryAcquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(r.lastUpdate).Seconds()
	r.tokens += elapsed * r.rate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastUpdate = now
	
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// Available returns the number of available tokens
func (r *RateLimiter) Available() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(r.lastUpdate).Seconds()
	tokens := r.tokens + elapsed*r.rate
	if tokens > r.maxTokens {
		tokens = r.maxTokens
	}
	return tokens
}
