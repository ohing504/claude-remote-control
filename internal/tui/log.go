package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/state"
)

// logRefresh는 follow 중 로그를 다시 읽는 주기다.
const logRefresh = 500 * time.Millisecond

type logTickMsg time.Time

func logTick() tea.Cmd {
	return tea.Tick(logRefresh, func(t time.Time) tea.Msg { return logTickMsg(t) })
}

// logHeight는 헤더·푸터를 뺀 viewport 높이다(제목1 + 도움말1 + 여유).
func logHeight(termHeight int) int {
	if termHeight <= 0 {
		termHeight = 24
	}
	h := termHeight - 3
	if h < 3 {
		h = 3
	}
	return h
}

// enterLog는 커서 워크스페이스의 로그를 viewport에 띄우고 follow를 켠다.
func (m model) enterLog() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	m.logName = m.rows[m.cursor].ws.Name
	m.mode = modeLog
	m.logFollow = true
	m.viewport = viewport.New(m.width, logHeight(m.height))
	m = m.loadLog()
	return m, logTick()
}

func (m model) exitLog() model {
	m.mode = modeList
	return m
}

// loadLog는 로그 파일을 읽어 viewport에 채운다. follow면 맨 아래로 스크롤.
func (m model) loadLog() model {
	data, err := os.ReadFile(state.LogPath(m.dir, m.logName)) //nolint:gosec // 내부 상태 경로
	if err != nil {
		m.viewport.SetContent("(로그 없음 — 아직 실행한 적 없거나 파일이 지워짐)")
		return m
	}
	m.viewport.SetContent(string(data))
	if m.logFollow {
		m.viewport.GotoBottom()
	}
	return m
}

func (m model) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m.exitLog(), nil
	case "f":
		m.logFollow = !m.logFollow
		if m.logFollow {
			m.viewport.GotoBottom()
		}
		return m, nil
	}
	// 스크롤은 viewport에 위임. 사용자가 위로 올리면 follow를 끊어 방해하지 않는다.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if !m.viewport.AtBottom() {
		m.logFollow = false
	}
	return m, cmd
}
