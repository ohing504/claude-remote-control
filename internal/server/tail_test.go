package server

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"crc/internal/state"
)

// safeBuf는 Tail이 다른 goroutine에서 Write하는 동안 테스트가 안전하게 읽게 한다.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// Tail이 기존 내용을 출력하고, 이후 append되는 내용도 따라가는지(follow).
func TestTailFollows(t *testing.T) {
	dir := t.TempDir()
	logPath := state.LogPath(dir, "x")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		t.Fatalf("픽스처: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("line1\n"), 0o600); err != nil {
		t.Fatalf("픽스처: %v", err)
	}

	var out safeBuf
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- Tail(logPath, &out, stop) }()

	waitFor(t, func() bool { return strings.Contains(out.String(), "line1") })

	// follow 대상 파일에 새 내용 append.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("append 열기: %v", err)
	}
	if _, err := f.WriteString("line2\n"); err != nil {
		t.Fatalf("append 쓰기: %v", err)
	}
	_ = f.Close()

	waitFor(t, func() bool { return strings.Contains(out.String(), "line2") })

	close(stop)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Tail 반환 에러: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stop 후 Tail이 반환하지 않음")
	}
}

// 로그 파일이 없으면 즉시 에러.
func TestTailMissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := Tail(state.LogPath(dir, "nope"), &safeBuf{}, make(chan struct{})); err == nil {
		t.Fatal("없는 로그 파일인데 에러가 없음")
	}
}

// 이미 detached로 떠 있으면 Foreground는 거부한다.
func TestForegroundRejectsWhenRunning(t *testing.T) {
	dir := t.TempDir()
	// 살아 있는 pid(자기 자신)를 기록해 Running으로 만든다.
	if err := os.WriteFile(state.PidPath(dir, "z"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatalf("픽스처: %v", err)
	}
	s := Server{Name: "z", Path: dir}
	if err := s.Foreground(dir); err == nil {
		t.Fatal("이미 실행 중인데 Foreground가 거부하지 않음")
	}
}

// waitFor는 cond가 참이 될 때까지 짧게 폴링한다(follow 폴링보다 넉넉히).
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("조건이 시간 내에 충족되지 않음")
}
