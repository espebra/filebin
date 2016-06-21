package metrics

import (
	"os"
	"testing"
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
