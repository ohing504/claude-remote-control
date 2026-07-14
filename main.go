// crc: 여러 claude remote-control 서버를 한 화면에서 켜고·끄고·상태 보는 TUI.
// M0에서는 config 로드/저장과 ls·add 명령만 제공한다.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"crc/internal/config"
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
	case "-h", "--help", "help":
		return cmdUsage()
	default:
		return fmt.Errorf("알 수 없는 명령: %s (crc help)", args[0])
	}
}

func cmdUsage() error {
	fmt.Print(`crc — claude remote-control 매니저

사용법:
  crc ls                   등록된 워크스페이스 목록
  crc add <path> [name]    워크스페이스 등록 (name 기본 basename)
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
	for _, ws := range cfg.Workspaces {
		fmt.Printf("%-20s %s\n", ws.Name, ws.Path)
	}
	return nil
}

func cmdAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("경로가 필요합니다: crc add <path> [name]")
	}
	abs, err := filepath.Abs(args[0])
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
