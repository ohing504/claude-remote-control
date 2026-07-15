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
	// 터미널 높이를 넘으면 커서 기준 창만 그린다(제목2 + 도움말2 + 여유).
	start, end := windowBounds(m.cursor, len(m.rows), visibleCount(m.height, 5))
	b.WriteString(moreHint(start, false))
	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}
		line := prefix + renderRow(m.rows[i])
		if i == m.cursor {
			line = cursorRow.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(moreHint(len(m.rows)-end, true))

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

	b.WriteString(helpStyle.Render(fmt.Sprintf("후보 %d · %s", len(m.cands), m.addRoot)) + "\n")
	if len(m.cands) == 0 {
		b.WriteString("  (이 폴더 아래 후보 없음 — 위 입력창으로 경로를 직접 추가)\n")
	}
	// 입력창3 + 제목2 + 헤더1 + 메시지1 + 도움말2 = 약 9줄을 리스트 밖이 차지.
	start, end := windowBounds(m.addCursor, len(m.cands), visibleCount(m.height, 9))
	b.WriteString(moreHint(start, false))
	for i := start; i < end; i++ {
		b.WriteString(renderCand(m.cands[i], m.focus == focusList && i == m.addCursor) + "\n")
	}
	b.WriteString(moreHint(len(m.cands)-end, true))

	if m.addMsg != "" {
		b.WriteString("\n" + m.addMsg)
	}
	b.WriteString("\n\n" + helpStyle.Render(
		"tab 포커스 전환 · space 선택 · ↵ 등록 · esc 취소"))
	return b.String()
}

// visibleCount는 헤더·푸터(chrome)를 뺀, 리스트에 쓸 수 있는 행 수다.
// height가 아직 안 들어왔으면(0) 넉넉한 기본을 쓴다.
func visibleCount(height, chrome int) int {
	if height <= 0 {
		height = 24
	}
	v := height - chrome
	if v < 3 {
		v = 3 // 최소한 커서 주변은 보이게
	}
	return v
}

// windowBounds는 cursor가 항상 보이도록 [start,end) 창을 고른다(가능하면 가운데).
func windowBounds(cursor, total, visible int) (int, int) {
	if total <= visible {
		return 0, total
	}
	start := cursor - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return start, start + visible
}

// moreHint는 창 밖에 가려진 항목 수를 한 줄로 알린다(없으면 빈 문자열).
func moreHint(hidden int, below bool) string {
	if hidden <= 0 {
		return ""
	}
	arrow := "↑"
	if below {
		arrow = "↓"
	}
	return helpStyle.Render(fmt.Sprintf("  %s %d개 더", arrow, hidden)) + "\n"
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
