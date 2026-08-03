package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/config"
	"crc/internal/server"
	"crc/internal/state"
)

// doDelete는 커서 워크스페이스를 등록 삭제한다(실행 중이면 정지 후).
func (m model) doDelete() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	ws := m.rows[m.cursor].ws
	if server.Status(state.PidPath(m.dir, ws.Name)) == server.Running {
		_ = server.StopByName(m.dir, ws.Name)
	}
	cfg, err := config.Load()
	if err != nil {
		m.err = err
		return m, nil
	}
	if err := cfg.Remove(ws.Name); err != nil {
		m.err = err
		return m, nil
	}
	if err := config.Save(cfg); err != nil {
		m.err = err
		return m, nil
	}
	return m, refreshCmd(m.dir)
}

// foreground는 커서 워크스페이스를 tea.ExecProcess로 현재 터미널에 붙여 실행한다.
// TUI를 잠시 접었다가 프로세스 종료 시 복귀하며 상태를 갱신한다.
func (m model) foreground() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	r := m.rows[m.cursor]
	if r.st == server.Running {
		m.err = nil // 이미 떠 있으면 중복 기동 방지 — 조용히 무시
		return m, nil
	}
	s := server.Server{Name: r.ws.Name, Path: r.ws.Path}
	dir := m.dir
	return m, tea.ExecProcess(s.Command(""), func(error) tea.Msg {
		rows, err := loadRows(dir)
		return refreshMsg{rows: rows, err: err}
	})
}
