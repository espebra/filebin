package stats

import (
	"testing"
	//"time"
	"os"
)

var (
	s Stats
)

func TestMain(m *testing.M) {
	s = InitStats()
	retCode := m.Run()
	os.Exit(retCode)
}

func TestIncrSet(t *testing.T) {
	value, found := s.Get("foo")
	if found == true {
		t.Fatal("The key was unexpectedly found")
	}

	if value != 0 {
		t.Fatal("The value is not 0")
	}

	value = s.IncrSet("foo", 1)
	if value != 1 {
		t.Fatal("The value is not 1")
	}

	value = s.IncrSet("foo", 1)
	if value != 2 {
		t.Fatal("The value is not 2")
	}
}

func TestGet(t *testing.T) {
	value, found := s.Get("foo")
	if found == false {
		t.Fatal("The key does not exist")
	}
	if value != 2 {
		t.Fatal("The value is not 2. Weird.")
	}
}

func TestGetAll(t *testing.T) {
	stats := s.GetAll()
	if len(stats) != 1 {
		t.Fatal("The number of stats is not 1")
	}
	if stats["foo"] != 2 {
		t.Fatal("The value is not 2. Weird.")
	}
}

func TestIncr(t *testing.T) {
	if s.Incr("bar") != 1 {
		t.Fatal("The value is not 1")
	}
	if s.Incr("bar") != 2 {
		t.Fatal("The value is not 2")
	}
	if s.Incr("bar") != 3 {
		t.Fatal("The value is not 3")
	}
}

func TestDecr(t *testing.T) {
	if s.Decr("bar") != 2 {
		t.Fatal("The value is not 2")
	}
	if s.Decr("bar") != 1 {
		t.Fatal("The value is not 1")
	}
	if s.Decr("bar") != 0 {
		t.Fatal("The value is not 0")
	}
}
