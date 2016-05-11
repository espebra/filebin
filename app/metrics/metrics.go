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
	Timestamp         time.Time
	TimestampReadable string
	Category          string
	Text string
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

//func (m *Metrics) Event(category string, text string) {
//	m.Lock()
//	defer m.Unlock()
//	e := Event{
//		Timestamp: time.Now().UTC(),
//		Category: category,
//		Text: text,
//	}
//	m.events = append(m.events, e)
//
//	if len(m.events) > 10000 {
//		_, m.events = m.events[len(m.events)-1], m.events[:len(m.events)-1]
//	}
//}

func (m *Metrics) Event(category string, text string) {
	m.Lock()
	defer m.Unlock()
	e := Event{
		Timestamp: time.Now().UTC(),
		Category:  category,
		Text:      text,
	}
	m.events = append(m.events, e)

	if len(m.events) > 10000 {
		_, m.events = m.events[len(m.events)-1], m.events[:len(m.events)-1]
	}
}

func (m *Metrics) GetEvents(limit int) []Event {
	m.RLock()
	defer m.RUnlock()
	var r []Event
	counter := 0
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		e.TimestampReadable = humanize.Time(e.Timestamp)
		r = append(r, e)
		counter += 1
		if limit != 0 && limit == counter {
			break
		}
	}
	return r
}

func (m *Metrics) GetEventCategory(category string, limit int) []Event {
	m.RLock()
	defer m.RUnlock()
	var r []Event
	counter := 0
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		if e.Category == category {
			counter += 1
			e.TimestampReadable = humanize.Time(e.Timestamp)
			r = append(r, e)
			if limit != 0 && limit == counter {
				break
			}
		}
	}
	return r
}
