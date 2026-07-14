package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolate는 XDG_CONFIG_HOME를 임시 디렉토리로 돌려 실제 홈을 건드리지 않게 한다.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	isolate(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Workspaces) != 0 {
		t.Fatalf("빈 Config 기대, got %d개", len(cfg.Workspaces))
	}
}

func TestAddSaveLoadRoundTrip(t *testing.T) {
	isolate(t)
	cfg := &Config{}
	if err := cfg.Add(Workspace{Name: "proj-a", Path: "/tmp/proj-a"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Workspaces) != 1 {
		t.Fatalf("1개 기대, got %d", len(got.Workspaces))
	}
	if w := got.Workspaces[0]; w.Name != "proj-a" || w.Path != "/tmp/proj-a" {
		t.Fatalf("round-trip 불일치: %+v", w)
	}
}

func TestAddDuplicateNameRejected(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Add(Workspace{Name: "dup", Path: "/a"}); err != nil {
		t.Fatalf("첫 Add: %v", err)
	}
	if err := cfg.Add(Workspace{Name: "dup", Path: "/b"}); err == nil {
		t.Fatal("중복 이름 거부 기대, nil 반환")
	}
	if len(cfg.Workspaces) != 1 {
		t.Fatalf("거부 후에도 1개 유지 기대, got %d", len(cfg.Workspaces))
	}
}

func TestFind(t *testing.T) {
	cfg := &Config{Workspaces: []Workspace{{Name: "x", Path: "/x"}}}
	tests := []struct {
		name  string
		query string
		found bool
	}{
		{"존재", "x", true},
		{"부재", "y", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cfg.Find(tc.query)
			if (got != nil) != tc.found {
				t.Fatalf("Find(%q): found=%v 기대, got %v", tc.query, tc.found, got != nil)
			}
		})
	}
}

func TestSaveAtomicNoTempLeftover(t *testing.T) {
	isolate(t)
	if err := Save(&Config{Workspaces: []Workspace{{Name: "a", Path: "/a"}}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("temp 파일 잔존: %s", e.Name())
		}
	}
}
