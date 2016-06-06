package metrics

import (
	"os"
	"strconv"
	"testing"
	"time"
)

var (
	m Metrics
)

func TestMain(test *testing.M) {
	m = Init()
	retCode := test.Run()
	os.Exit(retCode)
}

func TestIncrSet(t *testing.T) {
	value, found := m.Get("foo")
	if found == true {
		t.Fatal("The key was unexpectedly found")
	}

	if value != 0 {
		t.Fatal("The value is not 0")
	}

	value = m.IncrSet("foo", 1)
	if value != 1 {
		t.Fatal("The value is not 1")
	}

	value = m.IncrSet("foo", 1)
	if value != 2 {
		t.Fatal("The value is not 2")
	}
}

func TestGet(t *testing.T) {
	value, found := m.Get("foo")
	if found == false {
		t.Fatal("The key does not exist")
	}
	if value != 2 {
		t.Fatal("The value is not 2. Weird.")
	}
}

func TestGetAll(t *testing.T) {
	stats := m.GetStats()
	if len(stats) != 1 {
		t.Fatal("The number of stats is not 1")
	}
	if stats["foo"] != 2 {
		t.Fatal("The value is not 2. Weird.")
	}
}

func TestIncr(t *testing.T) {
	if m.Incr("bar") != 1 {
		t.Fatal("The value is not 1")
	}
	if m.Incr("bar") != 2 {
		t.Fatal("The value is not 2")
	}
	if m.Incr("bar") != 3 {
		t.Fatal("The value is not 3")
	}
}

func TestDecr(t *testing.T) {
	if m.Decr("bar") != 2 {
		t.Fatal("The value is not 2")
	}
	if m.Decr("bar") != 1 {
		t.Fatal("The value is not 1")
	}
	if m.Decr("bar") != 0 {
		t.Fatal("The value is not 0")
	}
}

func TestEvent(t *testing.T) {
	event := Event{
		Category: "foo",
		Text:     "som bare happened",
	}
	m.AddEvent(event)

	if len(m.events) != 1 {
		t.Fatal("Unexpected number of events. Not 1.")
	}
	for i := 0; i <= 20000; i++ {
		event := Event{
			Text: "som bare happened: " + strconv.Itoa(i),
		}
		m.AddEvent(event)
	}
	if len(m.events) != 10000 {
		t.Fatal("Unexpected number of events. Not 10000.")
	}
}

func TestGetEvents(t *testing.T) {
	events := m.GetEvents(Event{}, time.Time{}, 0)
	if len(events) != 10000 {
		t.Fatal("Unexpected number of events. Not 10000.")
	}
	events = m.GetEvents(Event{}, time.Time{}, 100)
	if len(events) != 100 {
		t.Fatal("Unexpected number of events. Not 100.")
	}
}
