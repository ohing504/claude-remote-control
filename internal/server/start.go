package server

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"crc/internal/state"
)

// command는 실행할 claude remote-control 커맨드를 구성한다(env 절대 규칙 적용).
func (s Server) command() *exec.Cmd {
	// s.Name은 사용자가 등록한 워크스페이스 이름 — 프로세스 실행이 이 도구의 목적이다.
	cmd := exec.Command("claude", "remote-control", "--name", s.Name) //nolint:gosec
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
	return startDetached(s.command(), pidPath, state.LogPath(dir, s.Name))
}

// startDetached는 커맨드를 백그라운드로 띄우고(부모가 죽어도 생존) pid를 기록한다.
// 로그는 logPath로 리다이렉트한다. 테스트는 임의 cmd를 넘겨 생명주기를 검증한다.
func startDetached(cmd *exec.Cmd, pidPath, logPath string) error {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // logPath는 내부에서 구성한 상태 경로
	if err != nil {
		return err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // 새 세션 — 부모(crc)가 죽어도 서버는 산다

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	// 자식이 fd를 상속했으므로 부모는 로그 파일을 닫아도 된다.
	_ = logFile.Close()

	pid := strconv.Itoa(cmd.Process.Pid)
	return os.WriteFile(pidPath, []byte(pid+"\n"), 0o600)
}
