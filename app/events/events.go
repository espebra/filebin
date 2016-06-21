package events

import (
	//"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize"
	"sync"
	"time"
)

type Events struct {
	sync.RWMutex
	events []*Event
}

type Event struct {
	sync.RWMutex
	tags      []string
	startTime time.Time
	endTime   time.Time
	//duration   time.Duration
	done     bool
	status   int
	bin      string
	filename string
	source   string
	text     string

	url string
}

func Init() Events {
	s := Events{}
	return s
}

func (s *Events) New(source string, tags []string, bin string, filename string) *Event {
	e := Event{}
	e.startTime = time.Now().UTC()
	e.tags = tags
	e.bin = bin
	e.filename = filename
	e.source = source
	s.Lock()
	defer s.Unlock()
	s.events = append(s.events, &e)

	// Remove the last event from the ring buffer if the limit is reached.
	if len(s.events) > 50000 {
		// The last event is the first entry in the slice.
		_, s.events = s.events[0], s.events[1:]
	}

	return &e
}

func (s *Events) GetEventsInProgress(offset int, limit int) []Event {
	s.RLock()
	defer s.RUnlock()
	var r []Event
	for i := len(s.events) - 1; i >= 0; i-- {
		if offset != 0 || limit != 0 {
			if i < offset {
				continue
			}
			if i > offset+limit {
				continue
			}
		}
		e := s.events[i]
		if e.done == false {
			//e.duration = time.Now().UTC().Sub(e.startTime)
			r = append(r, *e)
		}
	}
	return r
}

func (s *Events) GetAllEvents(offset int, limit int) []Event {
	s.RLock()
	defer s.RUnlock()
	var r []Event

	//for i := len(s.events) - 1; i >= 0; i-- {
	for i := range s.events {
		if offset != 0 && i < offset {
			continue
		}
		if limit != 0 && i >= offset+limit {
			continue
		}
		e := s.events[len(s.events)-1-i]
		r = append(r, *e)
	}
	return r
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (s *Events) GetEventsByTags(tags []string, offset int, limit int) []Event {
	s.RLock()
	defer s.RUnlock()
	var r []Event
	for i := len(s.events) - 1; i >= 0; i-- {
		e := s.events[i]
		for _, tag := range tags {
			if stringInSlice(tag, e.tags) {
				if offset != 0 || limit != 0 {
					found := len(r)
					if found < offset {
						continue
					}
					if found >= offset+limit {
						continue
					}
				}
				r = append(r, *e)
				break
			}
		}
	}
	return r
}

//func (m *Metrics) GetEvents(filter Event, limitTime time.Time, limitCount int) []Event {
//	m.RLock()
//	defer m.RUnlock()
//	var r []Event
//	counter := 0
//	for i := len(m.events) - 1; i >= 0; i-- {
//		e := m.events[i]
//
//		// Only consider records newer than the time limit
//		if e.Timestamp.IsZero() == false {
//			if e.Timestamp.Before(limitTime) || e.Timestamp.Equal(limitTime) {
//				continue
//			}
//		}
//
//		match := false
//
//		// Empty filter, should match everything
//		if filter == (Event{}) {
//			match = true
//		}
//
//		if filter.Bin != "" {
//			if filter.Bin == e.Bin {
//				match = true
//			}
//		}
//
//		if filter.Category != "" {
//			if filter.Category == e.Category {
//				match = true
//			}
//		}
//
//		if filter.Filename != "" {
//			if filter.Filename == e.Filename {
//				match = true
//			}
//		}
//
//		if filter.RemoteAddr != "" {
//			if filter.RemoteAddr == e.RemoteAddr {
//				match = true
//			}
//		}
//
//		if filter.URL != "" {
//			if filter.URL == e.URL {
//				match = true
//			}
//		}
//
//		if match {
//			r = append(r, e)
//			counter += 1
//		}
//
//		if limitCount != 0 && limitCount == counter {
//			break
//		}
//	}
//	return r
//}

func (e *Event) Update(text string, status int) {
	e.Lock()
	defer e.Unlock()
	e.text = text
	e.status = status
}

func (e *Event) Done() {
	e.Lock()
	defer e.Unlock()
	e.endTime = time.Now().UTC()
	e.done = true
}

func (e *Event) Status() int {
	e.RLock()
	defer e.RUnlock()
	return e.status
}

func (e *Event) Tags() []string {
	e.RLock()
	defer e.RUnlock()
	return e.tags
}

func (e *Event) Bin() string {
	e.RLock()
	defer e.RUnlock()
	return e.bin
}

func (e *Event) Filename() string {
	e.RLock()
	defer e.RUnlock()
	return e.filename
}

func (e *Event) Source() string {
	e.RLock()
	defer e.RUnlock()
	return e.source
}

func (e *Event) Text() string {
	e.RLock()
	defer e.RUnlock()
	return e.text
}

func (e *Event) StartTime() time.Time {
	e.RLock()
	defer e.RUnlock()
	return e.startTime
}

func (e *Event) Duration() time.Duration {
	e.RLock()
	defer e.RUnlock()
	if e.done == false {
		return time.Now().UTC().Sub(e.startTime)
	}
	return e.endTime.Sub(e.startTime)
}

func (e *Event) DurationReadable() string {
	e.RLock()
	defer e.RUnlock()
	return humanize.Time(e.startTime)
}

func (e *Event) IsDone() bool {
	e.RLock()
	defer e.RUnlock()
	return e.done
}
