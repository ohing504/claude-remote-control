// Package tui는 등록된 워크스페이스를 한 화면에 띄우고 토글하는 bubbletea 상태판이다.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/config"
	"crc/internal/server"
	"crc/internal/state"
)

// refreshInterval은 상태를 다시 판정하는 주기다. 사람이 보는 상태판이라 촘촘할 필요 없다.
const refreshInterval = 2 * time.Second

type row struct {
	ws config.Workspace
	st server.State
}

type model struct {
	rows   []row
	cursor int
	dir    string // state 디렉토리(pid·log)
	err    error
}

type (
	tickMsg    time.Time
	refreshMsg struct {
		rows []row
		err  error
	}
)

// Run은 TUI를 실행한다(무인자 `crc`의 진입점).
func Run() error {
	dir, err := state.EnsureDir()
	if err != nil {
		return err
	}
	m := model{dir: dir}
	m.rows, m.err = loadRows(dir)
	_, err = tea.NewProgram(m).Run()
	return err
}

// loadRows는 등록 목록을 읽어 각 워크스페이스의 실제 상태를 판정한다.
func loadRows(dir string) ([]row, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	rows := make([]row, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		rows[i] = row{ws: ws, st: server.Status(state.PidPath(dir, ws.Name))}
	}
	return rows, nil
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func refreshCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		rows, err := loadRows(dir)
		return refreshMsg{rows: rows, err: err}
	}
}

// toggleCmd는 선택 행을 상태 반대로 전환한다(running이면 정지, 아니면 시작).
// Stop은 최대 ~2초 블로킹할 수 있어 Cmd(별도 goroutine)로 돌려 UI를 막지 않는다.
func toggleCmd(dir string, r row) tea.Cmd {
	return func() tea.Msg {
		if r.st == server.Running {
			_ = server.StopByName(dir, r.ws.Name)
		} else {
			s := server.Server{Name: r.ws.Name, Path: r.ws.Path}
			_ = s.Start(dir)
		}
		rows, err := loadRows(dir)
		return refreshMsg{rows: rows, err: err}
	}
}

// allCmd는 전체를 시작(up)하거나 정지(down)한다.
func allCmd(dir string, rows []row, up bool) tea.Cmd {
	return func() tea.Msg {
		for _, r := range rows {
			if up {
				if r.st != server.Running {
					s := server.Server{Name: r.ws.Name, Path: r.ws.Path}
					_ = s.Start(dir)
				}
			} else if r.st == server.Running {
				_ = server.StopByName(dir, r.ws.Name)
			}
		}
		rows, err := loadRows(dir)
		return refreshMsg{rows: rows, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(refreshCmd(m.dir), tick())
	case refreshMsg:
		m.rows, m.err = msg.rows, msg.err
		if m.cursor >= len(m.rows) && len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.rows) > 0 {
			return m, toggleCmd(m.dir, m.rows[m.cursor])
		}
	case "a":
		return m, allCmd(m.dir, m.rows, true)
	case "x":
		return m, allCmd(m.dir, m.rows, false)
	case "r":
		return m, refreshCmd(m.dir)
	}
	return m, nil
}
