// crc: 여러 claude remote-control 서버를 한 화면에서 켜고·끄고·상태 보는 TUI.
// TUI(무인자 실행)는 M4. 그전까지는 서브커맨드로 조작한다.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"crc/internal/config"
	"crc/internal/server"
	"crc/internal/state"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "crc:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		// TUI는 M4. 그전까지는 사용법 안내.
		return cmdUsage()
	}
	switch args[0] {
	case "ls":
		return cmdLs()
	case "add":
		return cmdAdd(args[1:])
	case "rm":
		return cmdRm(args[1:])
	case "up":
		return cmdUp(args[1:])
	case "down":
		return cmdDown(args[1:])
	case "status":
		return cmdStatus()
	case "-h", "--help", "help":
		return cmdUsage()
	default:
		return fmt.Errorf("알 수 없는 명령: %s (crc help)", args[0])
	}
}

func cmdUsage() error {
	fmt.Print(`crc — claude remote-control 매니저

사용법:
  crc ls                   등록된 워크스페이스 목록 + 상태
  crc add <path> [name]    워크스페이스 등록 (name 기본 basename)
  crc rm <name>            워크스페이스 등록 삭제 (실행 중이면 정지 후)
  crc up   [name...]       서버 시작 (없으면 전체)
  crc down [name...]       서버 정지 (없으면 전체)
  crc status               전체 상태 판정
`)
	return nil
}

func cmdLs() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Workspaces) == 0 {
		fmt.Println("(등록된 워크스페이스 없음)")
		return nil
	}
	dir, err := state.Dir() // 없어도 됨 — 상태 파일 없으면 stopped로 판정
	if err != nil {
		return err
	}
	for _, ws := range cfg.Workspaces {
		st := server.Status(state.PidPath(dir, ws.Name))
		fmt.Printf("%-20s %-8s %s\n", ws.Name, st, ws.Path)
	}
	return nil
}

func cmdAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("경로가 필요합니다: crc add <path> [name]")
	}
	abs, err := config.ValidatePath(args[0])
	if err != nil {
		return err
	}
	name := filepath.Base(abs)
	if len(args) >= 2 {
		name = args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.Add(config.Workspace{Name: name, Path: abs}); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("등록: %s → %s\n", name, abs)
	return nil
}

func cmdRm(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("이름이 필요합니다: crc rm <name>")
	}
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Find(name) == nil {
		return fmt.Errorf("등록되지 않은 이름: %s", name)
	}
	// 실행 중이면 먼저 정지한다(고아 프로세스·pid 잔재 방지).
	dir, err := state.Dir()
	if err != nil {
		return err
	}
	if server.Status(state.PidPath(dir, name)) == server.Running {
		if err := server.StopByName(dir, name); err != nil {
			return err
		}
		fmt.Printf("정지: %s\n", name)
	}
	if err := cfg.Remove(name); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("삭제: %s\n", name)
	return nil
}

// resolveTargets는 인자 이름들을 워크스페이스로 해석한다. 인자가 없으면 전체.
func resolveTargets(cfg *config.Config, names []string) ([]config.Workspace, error) {
	if len(names) == 0 {
		return cfg.Workspaces, nil
	}
	out := make([]config.Workspace, 0, len(names))
	for _, n := range names {
		ws := cfg.Find(n)
		if ws == nil {
			return nil, fmt.Errorf("등록되지 않은 이름: %s", n)
		}
		out = append(out, *ws)
	}
	return out, nil
}

func cmdUp(names []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	targets, err := resolveTargets(cfg, names)
	if err != nil {
		return err
	}
	dir, err := state.EnsureDir()
	if err != nil {
		return err
	}
	for _, ws := range targets {
		s := server.Server{Name: ws.Name, Path: ws.Path}
		if err := s.Start(dir); err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", ws.Name, err)
			continue
		}
		fmt.Printf("시작: %s\n", ws.Name)
	}
	return nil
}

func cmdDown(names []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	targets, err := resolveTargets(cfg, names)
	if err != nil {
		return err
	}
	dir, err := state.Dir()
	if err != nil {
		return err
	}
	for _, ws := range targets {
		if server.Status(state.PidPath(dir, ws.Name)) != server.Running {
			continue // 안 떠 있으면 조용히 건너뜀
		}
		if err := server.StopByName(dir, ws.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", ws.Name, err)
			continue
		}
		fmt.Printf("정지: %s\n", ws.Name)
	}
	return nil
}

func cmdStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	dir, err := state.Dir()
	if err != nil {
		return err
	}
	for _, ws := range cfg.Workspaces {
		st := server.Status(state.PidPath(dir, ws.Name))
		fmt.Printf("%-20s %s\n", ws.Name, st)
	}
	return nil
}
