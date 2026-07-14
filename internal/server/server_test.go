package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"crc/internal/state"
)

// buildEnv이 절대 규칙을 지키는지 — 금지 키 제거 + 주입 키 단일값.
func TestBuildEnvAbsoluteRule(t *testing.T) {
	parent := []string{
		"FOO=bar",
		envRemote + "=1",       // 금지 — 제거돼야
		envRemoteType + "=old", // 기존값 — 새 값으로 대체돼야
	}
	got := buildEnv(parent)

	if slices.Contains(got, envRemote+"=1") {
		t.Fatal("금지 키 CLAUDE_CODE_REMOTE가 남아 있음")
	}
	if !slices.Contains(got, "FOO=bar") {
		t.Fatal("무관한 env FOO=bar가 사라짐")
	}
	// 주입 키는 정확히 한 번, 값은 1.
	n := 0
	for _, kv := range got {
		if k, v, _ := strings.Cut(kv, "="); k == envRemoteType {
			n++
			if v != "1" {
				t.Fatalf("%s=%s, 1 기대", k, v)
			}
		}
	}
	if n != 1 {
		t.Fatalf("%s가 %d번 — 정확히 1번 기대", envRemoteType, n)
	}
}

// command()가 올바른 인자·작업 디렉토리·env를 구성하는지.
func TestCommand(t *testing.T) {
	s := Server{Name: "proj-a", Path: "/tmp/proj-a"}
	cmd := s.command()

	wantArgs := []string{"claude", "remote-control", "--name", "proj-a"}
	if !slices.Equal(cmd.Args, wantArgs) {
		t.Fatalf("Args=%v, %v 기대", cmd.Args, wantArgs)
	}
	if cmd.Dir != "/tmp/proj-a" {
		t.Fatalf("Dir=%q, /tmp/proj-a 기대", cmd.Dir)
	}
	if !slices.Contains(cmd.Env, envRemoteType+"=1") {
		t.Fatalf("env에 %s=1 없음", envRemoteType)
	}
}

// 실제 프로세스로 시작→running→정지→stopped 사이클을 검증한다(claude 대신 sleep).
func TestLifecycleWithRealProcess(t *testing.T) {
	dir := t.TempDir()
	pidPath := state.PidPath(dir, "x")
	logPath := state.LogPath(dir, "x")

	if got := Status(pidPath); got != Stopped {
		t.Fatalf("시작 전 %v, Stopped 기대", got)
	}

	cmd := exec.Command("sleep", "30")
	if err := startDetached(cmd, pidPath, logPath); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) }) // 실패해도 프로세스 정리

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("로그 파일 미생성: %v", err)
	}
	if got := Status(pidPath); got != Running {
		t.Fatalf("시작 후 %v, Running 기대", got)
	}

	if err := Stop(pidPath); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if got := Status(pidPath); got != Stopped {
		t.Fatalf("정지 후 %v, Stopped 기대", got)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatal("정지 후 pid 파일이 남아 있음")
	}
}

// pid 파일은 있으나 프로세스가 없으면 Dead로 판정하는지.
func TestStatusDead(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "d.pid")
	// 존재하지 않을 매우 큰 PID.
	if err := os.WriteFile(pidPath, []byte("999999\n"), 0o600); err != nil {
		t.Fatalf("픽스처: %v", err)
	}
	if got := Status(pidPath); got != Dead {
		t.Fatalf("%v, Dead 기대", got)
	}
}

// Stop을 잠깐 뒤 다시 불러도(이미 종료) 안전한지.
func TestStopIdempotentishOnFastExit(t *testing.T) {
	dir := t.TempDir()
	pidPath := state.PidPath(dir, "y")
	logPath := state.LogPath(dir, "y")

	cmd := exec.Command("true") // 즉시 종료
	if err := startDetached(cmd, pidPath, logPath); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // 종료 대기
	if err := Stop(pidPath); err != nil {
		t.Fatalf("Stop(이미 종료): %v", err)
	}
}
