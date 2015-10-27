package config

import (
	"testing"
)

func TestInit(t *testing.T) {
	var cfg = Global

	if cfg.Port == 0 {
		t.Fatal("Missing Port")
	}

	if cfg.Host == "" {
		t.Fatal("Missing Host")
	}
}

