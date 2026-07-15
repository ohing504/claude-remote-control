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

func TestAddDuplicatePathRejected(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Add(Workspace{Name: "a", Path: "/same"}); err != nil {
		t.Fatalf("첫 Add: %v", err)
	}
	// 이름이 달라도 경로가 같으면 거부.
	if err := cfg.Add(Workspace{Name: "b", Path: "/same"}); err == nil {
		t.Fatal("중복 경로 거부 기대, nil 반환")
	}
	if len(cfg.Workspaces) != 1 {
		t.Fatalf("거부 후 1개 유지 기대, got %d", len(cfg.Workspaces))
	}
}

func TestRemove(t *testing.T) {
	cfg := &Config{Workspaces: []Workspace{{Name: "a", Path: "/a"}, {Name: "b", Path: "/b"}}}
	if err := cfg.Remove("a"); err != nil {
		t.Fatalf("Remove(a): %v", err)
	}
	if cfg.Find("a") != nil {
		t.Fatal("a가 제거되지 않음")
	}
	if cfg.Find("b") == nil {
		t.Fatal("b까지 사라짐")
	}
	if err := cfg.Remove("nope"); err == nil {
		t.Fatal("없는 이름 Remove는 에러 기대")
	}
}

func TestValidatePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("픽스처 생성: %v", err)
	}
	tests := []struct {
		name string
		path string
		ok   bool
	}{
		{"존재 디렉토리", dir, true},
		{"없는 경로", filepath.Join(dir, "nope"), false},
		{"파일(디렉토리 아님)", file, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			abs, err := ValidatePath(tc.path)
			if tc.ok {
				if err != nil {
					t.Fatalf("통과 기대, err=%v", err)
				}
				if !filepath.IsAbs(abs) {
					t.Fatalf("절대경로 기대, got %q", abs)
				}
			} else if err == nil {
				t.Fatalf("거부 기대, abs=%q", abs)
			}
		})
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

func TestUniqueName(t *testing.T) {
	cfg := &Config{Workspaces: []Workspace{
		{Name: "budget-master", Path: "/a/outsourcing/budget-master"},
	}}

	// 안 겹치면 base 그대로.
	if got := cfg.UniqueName("/a/proj-x", "proj-x"); got != "proj-x" {
		t.Fatalf("비충돌: %q, proj-x 기대", got)
	}
	// 겹치면 부모명을 붙여 유니크.
	if got := cfg.UniqueName("/a/references/budget-master", "budget-master"); got != "references-budget-master" {
		t.Fatalf("충돌 해소: %q, references-budget-master 기대", got)
	}

	// 부모까지 겹치면 조상, 끝내 숫자 접미사로.
	cfg.Workspaces = append(cfg.Workspaces,
		Workspace{Name: "references-budget-master", Path: "/a/references/budget-master"})
	got := cfg.UniqueName("/z/budget-master", "budget-master")
	if got == "budget-master" || cfg.Find(got) != nil {
		t.Fatalf("유니크하지 않은 이름 반환: %q", got)
	}
}
