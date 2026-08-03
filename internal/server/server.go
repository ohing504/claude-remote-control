// Package server는 remote-control 서버 프로세스의 생명주기(시작·상태·정지)를 담당한다.
package server

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"crc/internal/state"
)

// 절대 규칙 관련 env 키.
const (
	envRemoteType = "CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE" // =1 주입 (도구가 뜨는 게이트)
	envRemote     = "CLAUDE_CODE_REMOTE"                  // 금지 (서버 기동 거부) — 부모에 있으면 제거
)

// State는 서버 프로세스의 판정 결과다.
type State int

const (
	Stopped State = iota // pid 파일 없음 = 안 떠 있음
	Running              // pid 살아 있음
	Dead                 // pid 파일 있으나 프로세스 없음 = 조기 종료
)

func (s State) String() string {
	switch s {
	case Running:
		return "running"
	case Dead:
		return "dead"
	default:
		return "stopped"
	}
}

// Server는 실행 대상 워크스페이스 하나다.
type Server struct {
	Name string
	Path string
}

// buildEnv은 절대 규칙을 적용한 환경변수를 만든다:
// 부모 환경에서 CLAUDE_CODE_REMOTE를 제거하고 CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1을 넣는다.
func buildEnv(parent []string) []string {
	out := make([]string, 0, len(parent)+1)
	for _, kv := range parent {
		k, _, _ := strings.Cut(kv, "=")
		if k == envRemote || k == envRemoteType {
			continue // 금지 키 제거, 주입 키는 아래에서 단일값으로 다시 넣음
		}
		out = append(out, kv)
	}
	return append(out, envRemoteType+"=1")
}

// Uptime은 pid 파일의 생성 시각(= 서버 시작 시각)으로부터 경과 시간을 반환한다.
// pid 파일이 없으면(안 떠 있으면) ok=false. Start가 pid를 한 번만 쓰므로 mtime이 시작 시각이다.
func Uptime(pidPath string) (time.Duration, bool) {
	fi, err := os.Stat(pidPath)
	if err != nil {
		return 0, false
	}
	return time.Since(fi.ModTime()), true
}

// HumanDuration은 경과 시간을 짧게 포맷한다: 45s / 12m / 1h3m / 2d4h.
func HumanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h" + strconv.Itoa(int(d.Minutes())%60) + "m"
	default:
		return strconv.Itoa(int(d.Hours())/24) + "d" + strconv.Itoa(int(d.Hours())%24) + "h"
	}
}

// Status는 pid 파일을 읽어 프로세스 생존을 판정한다.
func Status(pidPath string) State {
	pid, err := readPid(pidPath)
	if err != nil {
		return Stopped
	}
	if processAlive(pid) {
		return Running
	}
	return Dead
}

// readPid는 pid 파일에서 PID를 읽는다.
func readPid(pidPath string) (int, error) {
	data, err := os.ReadFile(pidPath) //nolint:gosec // pidPath는 내부에서 구성한 상태 경로
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// processAlive는 signal 0으로 프로세스 존재를 확인한다.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Unix에서 FindProcess는 항상 성공하므로 signal 0으로 실제 존재를 확인한다.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if proc.Signal(syscall.Signal(0)) != nil {
		return false
	}
	// signal 0은 좀비(종료했으나 미reap)도 통과한다 — 좀비는 죽은 것으로 본다.
	return !isZombie(pid)
}

// Stop은 서버에 SIGTERM을 보내고, 잔존 시 SIGKILL로 종료한 뒤 pid 파일을 지운다.
func Stop(pidPath string) error {
	pid, err := readPid(pidPath)
	if err != nil {
		return errors.New("실행 중이 아님(pid 없음)")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidPath)
		return nil
	}

	_ = proc.Signal(syscall.SIGTERM)
	// 최대 ~2초 동안 graceful 종료를 기다린다.
	for i := 0; i < 20; i++ {
		if !processAlive(pid) {
			_ = os.Remove(pidPath)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	// 잔존 시 강제 종료. 곧바로 반환하면 재시작이 아직 살아 있는 서버 위에 새 서버를
	// 띄워 같은 이름으로 두 개가 등록을 시도한다(등록이 429로 거부될 수 있다).
	_ = proc.Signal(syscall.SIGKILL)
	for i := 0; i < 10; i++ {
		if !processAlive(pid) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = os.Remove(pidPath)
	return nil
}

// StopByName은 상태 디렉토리에서 이름으로 pid 경로를 찾아 정지한다.
func StopByName(dir, name string) error {
	return Stop(state.PidPath(dir, name))
}
