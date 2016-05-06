package stats

import (
	"sync"
	"time"
)

type Stats struct {
	sync.RWMutex
	startTime    time.Time
	counters   map[string]int64
}

func InitStats() Stats {
	s := Stats{}
	s.Lock()
	s.startTime = time.Now().UTC()
	s.counters = make(map[string]int64)
	s.Unlock()
	return s
}

func (s *Stats) Uptime() time.Duration {
	return time.Since(s.startTime)
}

func (s *Stats) GetCounters() map[string]int64 {
	s.RLock()
	defer s.RUnlock()
	return s.counters
}

func (s *Stats) Get(key string) (int64, bool) {
	s.RLock()
	defer s.RUnlock()
	value, found := s.counters[key]
	return value, found
}

func (s *Stats) IncrSet(key string, i int64) int64 {
	newValue := i
	s.Lock()
	defer s.Unlock()
	currentValue, found := s.counters[key]
	if found {
		newValue = currentValue + i
	}
	s.counters[key] = newValue
	return newValue
}
