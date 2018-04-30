package events

import (
	"os"
	"strconv"
	"testing"
)

var (
	ev Events
)

func TestMain(test *testing.M) {
	ev = Init()
	retCode := test.Run()
	os.Exit(retCode)
}

func TestSimpleEvent(t *testing.T) {
	e1 := ev.New("source1", []string{"tag1"}, "bin1", "file1")

	if len(ev.GetEventsInProgress(0, 100)) != 1 {
		t.Fatal("Unexpected number of events. Not 1.")
	}

	e1.Done()

	if len(ev.GetEventsInProgress(0, 100)) != 0 {
		t.Fatal("Unexpected number of events. Not 0.")
	}

	if len(ev.GetAllEvents(0, 100)) != 1 {
		t.Fatal("Unexpected number of events. Not 1.")
	}
}

func TestComplexEvents(t *testing.T) {
	var text string
	var count int

	e2 := ev.New("source2", []string{"tag2", "footag2"}, "bin2", "file2")
	e3 := ev.New("source3", []string{"tag3", "footag3"}, "bin3", "file3")
	e2.Update("e2 text number 1", 1)
	e3.Update("e3 text number 1", 1)
	e3.Done()
	e2.Update("e2 text number 2", 2)

	count = len(ev.GetEventsInProgress(0, 100))
	if count != 1 {
		t.Fatal("Unexpected number of open events. Not 1: ", count)
	}

	count = len(ev.GetAllEvents(0, 100))
	if count != 3 {
		t.Fatal("Unexpected number of events. Not 3: ", count)
	}

	e2.Done()

	count = len(ev.GetEventsInProgress(0, 100))
	if count != 0 {
		t.Fatal("Unexpected number of open events. Not 0: ", count)
	}

	text = e2.Text()
	if text != "e2 text number 2" {
		t.Fatal("Unexpected text: " + text)
	}

	text = e3.Text()
	if text != "e3 text number 1" {
		t.Fatal("Unexpected text: " + text)
	}
}

func TestGetEventsInProgress(t *testing.T) {
	var count int

	for i := 0; i < 20000; i++ {
		id := strconv.Itoa(i)
		e := ev.New("source"+id, []string{"tag" + id}, "bin"+id, "file"+id)

		if i == 10000 {
			e.Done()
		}
	}

	count = len(ev.GetEventsInProgress(0, 100000))
	if count != 5000 {
		t.Fatal("Unexpected number of open events. Not 5000: ", count)
	}

	count = len(ev.GetAllEvents(0, 100000))
	if count != 5000 {
		t.Fatal("Unexpected number of events. Not 5000: ", count)
	}

	events := ev.GetEventsInProgress(0, 100000)
	bin := events[0].Bin()
	if bin != "bin19999" {
		t.Fatal("The sorting is broken. Last event should come first: " + bin)
	}
}

func TestGetEventsByTags(t *testing.T) {
	var count int

	_ = ev.New("foo", []string{"admin", "dashboard"}, "", "")
	_ = ev.New("bar", []string{"admin", "events"}, "", "")
	_ = ev.New("baz", []string{"admin", "counters"}, "", "")

	count = len(ev.GetEventsByTags([]string{"admin"}, 0, 2))
	if count != 2 {
		t.Fatal("Unexpected number of (admin) events. Not 2: ", count)
	}

	count = len(ev.GetEventsByTags([]string{"admin", "dashboard"}, 0, 100))
	if count != 3 {
		t.Fatal("Unexpected number of (admin, dashboard) events. Not 3: ", count)
	}

	count = len(ev.GetEventsByTags([]string{"dashboard"}, 0, 100))
	if count != 1 {
		t.Fatal("Unexpected number of events. Not 1: ", count)
	}
}
