package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"crc/internal/server"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cursorRow  = lipgloss.NewStyle().Bold(true)

	// 상태색: running=초록 / stopped=회색 / dead=빨강.
	statusStyle = map[server.State]lipgloss.Style{
		server.Running: lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		server.Stopped: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		server.Dead:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	}
	statusMark = map[server.State]string{
		server.Running: "●",
		server.Stopped: "○",
		server.Dead:    "✗",
	}
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("crc — workspaces") + "\n\n")

	if len(m.rows) == 0 {
		b.WriteString("  (등록된 워크스페이스 없음 — crc add <path>)\n")
	}
	for i, r := range m.rows {
		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}
		line := prefix + renderRow(r)
		if i == m.cursor {
			line = cursorRow.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + helpStyle.Render(
		"↑↓/jk 이동 · ↵ 토글 · a 전체시작 · x 전체정지 · r 새로고침 · q 종료"))
	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("에러: "+m.err.Error()))
	}
	return b.String()
}

func renderRow(r row) string {
	st := r.st
	mark := statusStyle[st].Render(statusMark[st])
	status := statusStyle[st].Render(fmt.Sprintf("%-8s", st.String()))
	return fmt.Sprintf("%s %-20s %s %s", mark, r.ws.Name, status, r.ws.Path)
}
