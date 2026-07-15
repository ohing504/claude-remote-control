package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"crc/internal/config"
)

// 등록 두 개를 격리 환경에 심고 model을 만든다.
func newTestModel(t *testing.T) model {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir+"/cfg")
	t.Setenv("XDG_STATE_HOME", dir+"/state")

	cfg := &config.Config{Workspaces: []config.Workspace{
		{Name: "proj-a", Path: "/tmp/proj-a"},
		{Name: "proj-b", Path: "/tmp/proj-b"},
	}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("픽스처 저장: %v", err)
	}

	m := model{dir: dir + "/state"}
	var err error
	m.rows, err = loadRows(m.dir)
	if err != nil {
		t.Fatalf("loadRows: %v", err)
	}
	return m
}

// 리스트가 이름·상태와 함께 렌더되고, j로 커서가 내려가며, q로 종료하는지.
func TestTUIRenderMoveQuit(t *testing.T) {
	m := newTestModel(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 24))

	// 초기 렌더: 제목 + 두 워크스페이스 + stopped 상태.
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("workspaces")) &&
			bytes.Contains(b, []byte("proj-a")) &&
			bytes.Contains(b, []byte("proj-b")) &&
			bytes.Contains(b, []byte("stopped"))
	}, teatest.WithDuration(3*time.Second))

	// j로 커서를 두 번째 행으로.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// q로 종료.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// 커서 이동이 경계를 벗어나지 않는지(위/아래 끝에서 멈춤).
func TestCursorClamped(t *testing.T) {
	m := newTestModel(t)

	// 위로 — 이미 0이면 그대로.
	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if got := nm.(model).cursor; got != 0 {
		t.Fatalf("맨 위에서 k, cursor=%d, 0 기대", got)
	}
	// 아래로 두 번 — 행이 2개면 마지막(1)에서 멈춤.
	nm, _ = nm.(model).handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	nm, _ = nm.(model).handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := nm.(model).cursor; got != 1 {
		t.Fatalf("맨 아래에서 j 반복, cursor=%d, 1 기대", got)
	}
}
