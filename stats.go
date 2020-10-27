package main

import (
	"sync"
	"time"
)

// HashStats represents the password hashing statistics data
type HashStats struct {
	Total   uint64 `json:"total"`
	Average uint64 `json:"average"`
}

// HashStatsStorage manipulates the statistics data
type HashStatsStorage struct {
	mu    sync.RWMutex
	Stats HashStats
}

// NewHashStatsStorage constructs a new instance of the password hashing statistics data storage
func NewHashStatsStorage() *HashStatsStorage {
	hashStatsStorage := &HashStatsStorage{}
	return hashStatsStorage
}

// Update the statistics data with the new call information
func (s *HashStatsStorage) Update(startTime time.Time) {
	elapsed := time.Now().Sub(startTime)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats.Average = (s.Stats.Average*s.Stats.Total + uint64(elapsed.Microseconds())) / (s.Stats.Total + 1)
	s.Stats.Total++
	return
}

// GetCurrentStats returns current statistics
func (s *HashStatsStorage) GetCurrentStats() HashStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Stats
}
