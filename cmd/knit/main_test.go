package main

import "testing"

func TestParseArgsSyncRequiresLockFile(t *testing.T) {
	_, err := parseArgs([]string{"sync"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgsSyncLockFileAndGlobal(t *testing.T) {
	cfg, err := parseArgs([]string{"sync", "-f", "skills-lock.json", "-g"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "sync" || cfg.LockFile != "skills-lock.json" || !cfg.Global {
		t.Fatalf("bad config: %#v", cfg)
	}
}
