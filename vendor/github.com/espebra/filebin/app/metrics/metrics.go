package metrics

import (
	"sync"
	"time"
)

type Metrics struct {
	sync.RWMutex
	startTime time.Time
	stats     map[string]int64
}

func Init() Metrics {
	m := Metrics{}
	m.Lock()
	m.startTime = time.Now().UTC()
	m.stats = make(map[string]int64)
	m.Unlock()
	return m
}

func (m *Metrics) Uptime() time.Duration {
	m.RLock()
	defer m.RUnlock()
	return time.Since(m.startTime)
}

func (m *Metrics) StartTime() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.startTime
}

func (m *Metrics) GetStats() map[string]int64 {
	m.RLock()
	defer m.RUnlock()
	return m.stats
}

func (m *Metrics) Get(key string) (int64, bool) {
	m.RLock()
	defer m.RUnlock()
	value, found := m.stats[key]
	return value, found
}

func (m *Metrics) IncrSet(key string, i int64) int64 {
	newValue := i
	m.Lock()
	defer m.Unlock()
	currentValue, found := m.stats[key]
	if found {
		newValue = currentValue + i
	}
	m.stats[key] = newValue
	return newValue
}

func (m *Metrics) Incr(key string) int64 {
	m.Lock()
	defer m.Unlock()
	currentValue, _ := m.stats[key]
	newValue := currentValue + 1
	m.stats[key] = newValue
	return newValue
}

func (m *Metrics) Decr(key string) int64 {
	m.Lock()
	defer m.Unlock()
	currentValue, _ := m.stats[key]
	newValue := currentValue - 1
	m.stats[key] = newValue
	return newValue
}
