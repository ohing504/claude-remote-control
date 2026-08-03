package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
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
	cmd := s.Command("")

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
	if err := startDetached(cmd, pidPath, logPath, state.ErrPath(dir, "x")); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) }) // 실패해도 프로세스 정리

	// 진단 로그는 서버가 --debug-file로 직접 쓴다. crc가 만드는 건 stderr 파일이다.
	if _, err := os.Stat(state.ErrPath(dir, "x")); err != nil {
		t.Fatalf("stderr 파일 미생성: %v", err)
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
	if err := startDetached(cmd, pidPath, logPath, state.ErrPath(dir, "y")); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // 종료 대기
	if err := Stop(pidPath); err != nil {
		t.Fatalf("Stop(이미 종료): %v", err)
	}
}

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{50 * time.Hour, "2d2h"},
	}
	for _, c := range cases {
		if got := HumanDuration(c.d); got != c.want {
			t.Errorf("HumanDuration(%v)=%q, %q 기대", c.d, got, c.want)
		}
	}
}

// 살아있는 프로세스로 Uptime이 양수를 반환하는지.
func TestUptime(t *testing.T) {
	dir := t.TempDir()
	pidPath := state.PidPath(dir, "u")
	cmd := exec.Command("sleep", "30")
	if err := startDetached(cmd, pidPath, state.LogPath(dir, "u"), state.ErrPath(dir, "u")); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) })

	d, ok := Uptime(pidPath)
	if !ok || d < 0 {
		t.Fatalf("Uptime=(%v,%v), 양수 기대", d, ok)
	}
	// 없는 pid 파일은 ok=false.
	if _, ok := Uptime(state.PidPath(dir, "nope")); ok {
		t.Fatal("없는 pid인데 ok=true")
	}
}

// 즉시 종료해 좀비가 된 프로세스를 Dead로 판정하는지(darwin: sysctl, 그 외: 종료 감지).
func TestZombieDetectedAsDead(t *testing.T) {
	dir := t.TempDir()
	pidPath := state.PidPath(dir, "z")
	cmd := exec.Command("true") // 즉시 종료
	if err := startDetached(cmd, pidPath, state.LogPath(dir, "z"), state.ErrPath(dir, "z")); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) })

	// startDetached는 wait하지 않으므로 자식은 좀비로 남는다(부모=이 테스트 프로세스).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if Status(pidPath) == Dead {
			return // 좀비를 Dead로 정확히 판정
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("좀비를 Dead로 판정하지 못함(status=%v)", Status(pidPath))
}

// Restart가 떠 있던 프로세스를 죽이고 새 프로세스로 갈아끼우는지.
// claude 대신 PATH 앞단에 sleep을 실행하는 가짜 claude를 놓고 검증한다.
func TestRestartReplacesProcess(t *testing.T) {
	dir := t.TempDir()
	bin := t.TempDir()
	script := "#!/bin/sh\nexec sleep 30\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o700); err != nil { //nolint:gosec // 테스트용 실행 스크립트
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	s := Server{Name: "x", Path: dir}
	if err := s.Start(dir); err != nil {
		t.Fatalf("Start: %v", err)
	}
	pidPath := state.PidPath(dir, "x")
	t.Cleanup(func() { _ = Stop(pidPath) })

	first, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Restart(dir); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	second, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) == string(first) {
		t.Fatalf("pid가 그대로다: %s", second)
	}
	if got := Status(pidPath); got != Running {
		t.Fatalf("재시작 후 %v, Running 기대", got)
	}

	// crc는 자식을 wait하지 않아 종료한 프로세스가 좀비로 남는다. 좀비는 죽은 것으로 본다.
	old, _ := strconv.Atoi(strings.TrimSpace(string(first)))
	if err := syscall.Kill(old, 0); err == nil && !isZombie(old) {
		t.Fatalf("이전 프로세스 %d가 살아 있다", old)
	}
}

// 안 떠 있는 서버에 Restart를 걸면 그냥 시작되는지.
func TestRestartFromStopped(t *testing.T) {
	dir := t.TempDir()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil { //nolint:gosec // 테스트용 실행 스크립트
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	s := Server{Name: "x", Path: dir}
	if err := s.Restart(dir); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	pidPath := state.PidPath(dir, "x")
	t.Cleanup(func() { _ = Stop(pidPath) })

	if got := Status(pidPath); got != Running {
		t.Fatalf("%v, Running 기대", got)
	}
}

// Start 경로의 커맨드에 --debug-file이 붙고, fg 경로에는 안 붙는지.
func TestCommandDebugFile(t *testing.T) {
	s := Server{Name: "proj-a", Path: "/tmp/proj-a"}

	want := []string{"claude", "remote-control", "--name", "proj-a", "--debug-file", "/tmp/s/proj-a.log"}
	if got := s.Command("/tmp/s/proj-a.log").Args; !slices.Equal(got, want) {
		t.Fatalf("Args=%v, %v 기대", got, want)
	}
	// fg는 화면을 눈앞에서 보므로 진단 파일을 지정하지 않는다.
	if got := s.Command("").Args; slices.Contains(got, "--debug-file") {
		t.Fatalf("fg 커맨드에 --debug-file이 붙음: %v", got)
	}
}

// 서버 stdout은 버리고 stderr만 <name>.err에 남기는지.
func TestStartDropsStdoutKeepsStderr(t *testing.T) {
	dir := t.TempDir()
	pidPath := state.PidPath(dir, "x")
	logPath := state.LogPath(dir, "x")
	errPath := state.ErrPath(dir, "x")

	cmd := exec.Command("sh", "-c", "echo 화면반복; echo 기동실패 1>&2")
	if err := startDetached(cmd, pidPath, logPath, errPath); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) })
	time.Sleep(200 * time.Millisecond)

	gotErr, err := os.ReadFile(errPath)
	if err != nil {
		t.Fatalf("err 파일 미생성: %v", err)
	}
	if !strings.Contains(string(gotErr), "기동실패") {
		t.Fatalf("stderr 내용이 없음: %q", gotErr)
	}
	if strings.Contains(string(gotErr), "화면반복") {
		t.Fatalf("stdout이 err 파일에 섞임: %q", gotErr)
	}
	// crc는 빈 로그 파일만 미리 만들고, 내용은 서버가 --debug-file로 쓴다.
	if fi, err := os.Stat(logPath); err != nil || fi.Size() != 0 {
		t.Fatalf("로그 파일에 stdout이 흘러들어감: size=%v, err=%v", fi, err)
	}
}

// 로그 디렉토리 총량이 상한을 넘으면 시작 시 비우고, 넘지 않으면 그대로 두는지(이어쓰기 보존).
func TestStartTrimsOversizedLogDir(t *testing.T) {
	dir := t.TempDir()
	big := state.LogPath(dir, "big")
	small := state.LogPath(dir, "small")
	for _, p := range []string{big, small} {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// 상한 초과는 세션 로그가 함께 쌓인 상황을 흉내 낸다(파일 여러 개의 합).
	if err := os.WriteFile(big, make([]byte, maxLogBytes/2+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(big), "session.log"), make([]byte, maxLogBytes/2+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(small, []byte("이전 실행 기록\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{"big": big, "small": small} {
		pidPath := state.PidPath(dir, name)
		cmd := exec.Command("sleep", "30")
		if err := startDetached(cmd, pidPath, path, state.ErrPath(dir, name)); err != nil {
			t.Fatalf("startDetached(%s): %v", name, err)
		}
		t.Cleanup(func() { _ = Stop(pidPath) })
	}

	// 디렉토리를 비운 뒤 빈 로그 파일만 다시 만든다.
	if fi, err := os.Stat(big); err != nil || fi.Size() != 0 {
		t.Fatalf("상한 초과 로그가 안 비워짐: size=%v, err=%v", fi, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(big), "session.log")); !os.IsNotExist(err) {
		t.Fatal("함께 쌓인 세션 로그가 안 지워짐")
	}
	if b, err := os.ReadFile(small); err != nil || !strings.Contains(string(b), "이전 실행 기록") {
		t.Fatalf("상한 이하 로그가 사라짐: %q, %v", b, err)
	}
}

// 시작 직후 진단 로그 파일이 존재하는지. 서버가 파일을 만들기까지 시간이 걸려
// 그전에 `crc log`를 부르면 "파일 없음"으로 실패했다.
func TestStartCreatesLogFileUpfront(t *testing.T) {
	dir := t.TempDir()
	pidPath := state.PidPath(dir, "x")
	logPath := state.LogPath(dir, "x")

	// 진단 로그를 쓰지 않는 프로그램으로도 파일이 있어야 한다.
	if err := startDetached(exec.Command("sleep", "30"), pidPath, logPath, state.ErrPath(dir, "x")); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) })

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("시작 직후 로그 파일이 없음: %v", err)
	}
}

// 미리 만드는 동작이 이전 실행 기록을 지우지 않는지(서버가 이어쓴다).
func TestStartKeepsExistingLogContent(t *testing.T) {
	dir := t.TempDir()
	logPath := state.LogPath(dir, "x")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("이전 실행 기록\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	pidPath := state.PidPath(dir, "x")
	if err := startDetached(exec.Command("sleep", "30"), pidPath, logPath, state.ErrPath(dir, "x")); err != nil {
		t.Fatalf("startDetached: %v", err)
	}
	t.Cleanup(func() { _ = Stop(pidPath) })

	b, err := os.ReadFile(logPath)
	if err != nil || !strings.Contains(string(b), "이전 실행 기록") {
		t.Fatalf("이전 기록이 사라짐: %q, %v", b, err)
	}
}
