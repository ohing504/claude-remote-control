package server

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"crc/internal/state"
)

// maxLogBytes는 서버 하나의 로그 디렉토리 총량 상한이다. 서버가 진단 로그를
// 이어쓰고 세션마다 로그를 새로 만들어 그대로 두면 무한히 자란다. 시작 시 총량이
// 이 크기를 넘으면 오래된 파일부터 지운다. 실행 중에는 검사하지 않으므로 오래
// 켜둔 서버는 다시 띄울 때까지 줄지 않는다.
const maxLogBytes = 5 << 20

// StartInterval은 여러 서버를 연달아 띄울 때 시작 사이에 두는 간격이다. 등록 요청이
// 몰리면 claude 쪽이 429로 거부한다(실측: 9개를 한꺼번에 띄우자 4개가
// `Registration: Rate limited (429)`로 조기 종료).
const StartInterval = time.Second

// Command는 실행할 claude remote-control 커맨드를 구성한다(env 절대 규칙 적용).
// debugFile이 비어 있지 않으면 서버가 그 경로에 진단 로그를 기록한다. 포그라운드
// 실행(tea.ExecProcess)은 화면을 눈앞에서 보므로 빈 문자열을 넘긴다.
func (s Server) Command(debugFile string) *exec.Cmd {
	// s.Name은 사용자가 등록한 워크스페이스 이름 — 프로세스 실행이 이 도구의 목적이다.
	// --no-create-session-in-dir: `claude remote-control`의 기본 동작(시작 시 빈 세션
	// 하나를 미리 만들어 둠)을 끈다. 세션은 폰이나 데스크탑에서 "새 세션"으로 직접
	// 만들어 쓰므로 미리 만든 빈 세션은 목록에 쌓이기만 하고 쓰이지 않는다.
	args := []string{"remote-control", "--name", s.Name, "--no-create-session-in-dir"}
	if debugFile != "" {
		args = append(args, "--debug-file", debugFile)
	}
	cmd := exec.Command("claude", args...) //nolint:gosec
	cmd.Dir = s.Path
	cmd.Env = buildEnv(os.Environ())
	return cmd
}

// Start는 서버를 detached로 띄운다. 이미 running이면 거부한다.
func (s Server) Start(dir string) error {
	pidPath := state.PidPath(dir, s.Name)
	if Status(pidPath) == Running {
		return errors.New("이미 실행 중: " + s.Name)
	}
	logPath := state.LogPath(dir, s.Name)
	return startDetached(s.Command(logPath), pidPath, logPath, state.ErrPath(dir, s.Name))
}

// Restart는 떠 있으면 정지한 뒤 다시 띄운다. 응답이 멎은 서버를 되살리는 게 주 용도라
// 정지 실패(이미 죽었거나 pid 파일만 남은 경우)는 무시하고 시작으로 넘어간다.
func (s Server) Restart(dir string) error {
	_ = StopByName(dir, s.Name)
	return s.Start(dir)
}

// Foreground는 서버를 현재 터미널에 붙여(blocking) 실행한다. 미심쩍을 때 로그를
// 눈앞에서 보며 띄우는 용도라 detached하지 않고 pid 파일도 만들지 않는다.
// 이미 detached로 떠 있으면 중복 기동을 거부한다.
func (s Server) Foreground(dir string) error {
	if Status(state.PidPath(dir, s.Name)) == Running {
		return errors.New("이미 실행 중: " + s.Name)
	}
	cmd := s.Command("") // env 절대 규칙 + cmd.Dir 동일, Setsid 없음(터미널에 붙는다)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// startDetached는 커맨드를 백그라운드로 띄우고(부모가 죽어도 생존) pid를 기록한다.
// stdout은 버린다 — 서버가 연결 상태 화면을 1초에 한 번꼴로 다시 그려 같은 몇 줄이
// 무한히 쌓이고, 그 내용은 진단 로그(logPath)에 더 정확히 들어 있다. stderr는
// 기동 실패 원인이 나오는 유일한 곳이라 errPath에 남긴다.
// 테스트는 임의 cmd를 넘겨 생명주기를 검증한다.
func startDetached(cmd *exec.Cmd, pidPath, logPath, errPath string) error {
	logDir := filepath.Dir(logPath)
	trimOversizedLogDir(logDir)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}

	// 서버가 진단 로그를 만들기까지 시간이 걸려, 그전에 `crc log`를 부르면 파일이
	// 없어 실패한다. 빈 파일을 미리 만들어 둔다(서버는 이어쓰므로 내용은 안전하다).
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil { //nolint:gosec // logPath는 내부에서 구성한 상태 경로
		_ = f.Close()
	}

	errFile, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // errPath는 내부에서 구성한 상태 경로
	if err != nil {
		return err
	}
	cmd.Stdout = nil // /dev/null
	cmd.Stderr = errFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // 새 세션 — 부모(crc)가 죽어도 서버는 산다

	if err := cmd.Start(); err != nil {
		_ = errFile.Close()
		return err
	}
	// 자식이 fd를 상속했으므로 부모는 파일을 닫아도 된다.
	_ = errFile.Close()

	pid := strconv.Itoa(cmd.Process.Pid)
	if err := os.WriteFile(pidPath, []byte(pid+"\n"), 0o600); err != nil {
		// pid를 기록하지 못하면 crc가 이 서버를 추적할 수 없다(down이나 status에 안 잡힌다).
		// 추적 불가능한 서버를 남기느니 방금 띄운 프로세스를 정리한다.
		_ = cmd.Process.Kill()
		return err
	}
	return nil
}

// trimOversizedLogDir은 총량이 상한을 넘으면 오래된 파일부터 지워 상한 이하로
// 만든다. 세션 로그와 브리지 트랜스크립트가 함께 쌓이는데, 디렉토리를 통째로
// 지우면 방금까지 쓰던 진단 로그도 사라져 재시작 전 기록을 볼 수 없다.
// 상한 이하면 아무것도 지우지 않는다(서버가 진단 로그를 이어쓴다).
func trimOversizedLogDir(logDir string) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	type logFile struct {
		path    string
		size    int64
		modTime time.Time
	}
	files := make([]logFile, 0, len(entries))
	var total int64
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, logFile{filepath.Join(logDir, e.Name()), fi.Size(), fi.ModTime()})
		total += fi.Size()
	}
	if total <= maxLogBytes {
		return
	}
	slices.SortFunc(files, func(a, b logFile) int { return a.modTime.Compare(b.modTime) })
	for _, f := range files {
		if total <= maxLogBytes {
			return
		}
		if err := os.RemoveAll(f.path); err == nil { // 실패해도 기동은 막지 않는다
			total -= f.size
		}
	}
}
