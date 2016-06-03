package metrics

import (
	"github.com/dustin/go-humanize"
	"sync"
	"time"
)

type Metrics struct {
	sync.RWMutex
	startTime time.Time
	stats     map[string]int64
	events    []Event
}

type Event struct {
	Bin               string
	Category          string
	Filename          string
	RemoteAddr        string
	Text              string
	Timestamp         time.Time
	URL        string
}

func Init() Metrics {
	m := Metrics{}
	m.Lock()
	m.startTime = time.Now().UTC()
	m.stats = make(map[string]int64)
	m.Unlock()
	return m
}

func (e *Event) TimestampReadable() string {
	return humanize.Time(e.Timestamp)
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

func (m *Metrics) AddEvent(e Event) {
	e.Timestamp = time.Now().UTC()
	m.Lock()
	defer m.Unlock()
	m.events = append(m.events, e)

	// Remove the last event from the ring buffer if the limit is reached.
	if len(m.events) > 10000 {
		// The last event is the first entry in the slice.
		_, m.events = m.events[0], m.events[1:]
	}
}

func (m *Metrics) GetEvents(filter Event, limitTime time.Time, limitCount int) []Event {
	m.RLock()
	defer m.RUnlock()
	var r []Event
	counter := 0
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]

		// Only consider records newer than the time limit
		if e.Timestamp.IsZero() == false {
			if e.Timestamp.Before(limitTime) || e.Timestamp.Equal(limitTime) {
				continue
			}
		}

		match := false

		// Empty filter, should match everything
		if filter == (Event{}) {
			match = true
		}

		if filter.Bin != "" {
			if filter.Bin == e.Bin {
				match = true
			}
		}

		if filter.Category != "" {
			if filter.Category == e.Category {
				match = true
			}
		}

		if filter.Filename != "" {
			if filter.Filename == e.Filename {
				match = true
			}
		}

		if filter.RemoteAddr != "" {
			if filter.RemoteAddr == e.RemoteAddr {
				match = true
			}
		}

		if filter.URL != "" {
			if filter.URL == e.URL {
				match = true
			}
		}

		if match {
			r = append(r, e)
			counter += 1
		}

		if limitCount != 0 && limitCount == counter {
			break
		}
	}
	return r
}

