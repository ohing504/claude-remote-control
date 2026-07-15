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
	if m.mode == modeAdd {
		return m.addView()
	}
	return m.listView()
}

func (m model) listView() string {
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
		"↑↓/jk 이동 · ↵ 토글 · a 추가 · A 전체시작 · x 전체정지 · r 새로고침 · q 종료"))
	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("에러: "+m.err.Error()))
	}
	return b.String()
}

func (m model) addView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("crc — 추가") + "\n\n")

	// 경로 입력창(포커스면 마커 표시).
	inputPrefix := "  "
	if m.focus == focusInput {
		inputPrefix = "▸ "
	}
	b.WriteString(inputPrefix + m.input.View() + "\n\n")

	b.WriteString(helpStyle.Render("후보 · "+m.addRoot) + "\n")
	if len(m.cands) == 0 {
		b.WriteString("  (이 폴더 아래 후보 없음 — 위 입력창으로 경로를 직접 추가)\n")
	}
	for i, c := range m.cands {
		b.WriteString(renderCand(c, m.focus == focusList && i == m.addCursor) + "\n")
	}

	if m.addMsg != "" {
		b.WriteString("\n" + m.addMsg)
	}
	b.WriteString("\n\n" + helpStyle.Render(
		"tab 포커스 전환 · space 선택 · ↵ 등록 · esc 취소"))
	return b.String()
}

func renderCand(c addCand, cursor bool) string {
	prefix := "  "
	if cursor {
		prefix = "▸ "
	}
	box := "[ ]"
	switch {
	case c.already:
		box = "[–]"
	case c.selected:
		box = "[x]"
	}
	line := fmt.Sprintf("%s%s %-20s %s", prefix, box, c.c.Name, c.c.Path)
	switch {
	case c.already:
		return helpStyle.Render(line + "  (등록됨)")
	case cursor:
		return cursorRow.Render(line)
	default:
		return line
	}
}

func renderRow(r row) string {
	st := r.st
	mark := statusStyle[st].Render(statusMark[st])
	status := statusStyle[st].Render(fmt.Sprintf("%-8s", st.String()))
	return fmt.Sprintf("%s %-20s %s %s", mark, r.ws.Name, status, r.ws.Path)
}
