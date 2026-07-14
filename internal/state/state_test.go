package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirRespectsXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/custom/state")
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if want := "/custom/state/crc"; dir != want {
		t.Fatalf("Dir=%q, %q 기대", dir, want)
	}
}

func TestPathHelpers(t *testing.T) {
	dir := "/s"
	if got, want := PidPath(dir, "a"), filepath.Join("/s", "a.pid"); got != want {
		t.Fatalf("PidPath=%q, %q 기대", got, want)
	}
	if got, want := LogPath(dir, "a"), filepath.Join("/s", "a.log"); got != want {
		t.Fatalf("LogPath=%q, %q 기대", got, want)
	}
}

func TestEnsureDirCreates(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir, err := EnsureDir()
	if err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("디렉토리 미생성: dir=%q err=%v", dir, err)
	}
}
