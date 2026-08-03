package server

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"crc/internal/state"
)

// Command는 실행할 claude remote-control 커맨드를 구성한다(env 절대 규칙 적용).
// TUI의 포그라운드 실행(tea.ExecProcess)도 이 커맨드를 쓴다.
func (s Server) Command() *exec.Cmd {
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
	return startDetached(s.Command(), pidPath, state.LogPath(dir, s.Name))
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
	cmd := s.Command() // env 절대 규칙 + cmd.Dir 동일, Setsid 없음(터미널에 붙는다)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
