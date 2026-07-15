package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/config"
	"crc/internal/state"
)

// 로그 파일이 있는 워크스페이스 하나를 심은 목록 model을 만든다.
func newLogModel(t *testing.T, logBody string) model {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home+"/cfg")
	t.Setenv("XDG_STATE_HOME", home+"/state")

	cfg := &config.Config{Workspaces: []config.Workspace{{Name: "proj-a", Path: "/tmp/proj-a"}}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("픽스처: %v", err)
	}
	stateDir, err := state.EnsureDir()
	if err != nil {
		t.Fatalf("state dir: %v", err)
	}
	if logBody != "" {
		if err := os.WriteFile(state.LogPath(stateDir, "proj-a"), []byte(logBody), 0o600); err != nil {
			t.Fatalf("로그 픽스처: %v", err)
		}
	}
	m := model{dir: stateDir, input: textinput.New(), height: 24, width: 80}
	m.rows, _ = loadRows(m.dir)
	return m
}

// l로 로그뷰에 들어가면 로그 내용이 viewport에 실리는지.
func TestEnterLogLoadsContent(t *testing.T) {
	m := newLogModel(t, "boom: config missing\nline2\n")
	nm, cmd := m.handleKey(key("l"))
	m = nm.(model)
	if m.mode != modeLog {
		t.Fatalf("mode=%v, modeLog 기대", m.mode)
	}
	if cmd == nil {
		t.Fatal("follow tick 커맨드가 없음")
	}
	if !strings.Contains(m.viewport.View(), "boom: config missing") {
		t.Fatalf("로그 내용이 viewport에 없음:\n%s", m.viewport.View())
	}
	if !m.logFollow {
		t.Fatal("진입 시 follow가 켜져 있어야")
	}
}

// 로그 파일이 없으면 안내 문구를 보여주는지.
func TestEnterLogMissingFile(t *testing.T) {
	m := newLogModel(t, "")
	nm, _ := m.handleKey(key("l"))
	m = nm.(model)
	if !strings.Contains(m.viewport.View(), "로그 없음") {
		t.Fatalf("로그 없음 안내가 없음:\n%s", m.viewport.View())
	}
}

// f로 follow를 껐다 켜고, esc로 목록에 복귀하는지.
func TestLogFollowToggleAndExit(t *testing.T) {
	m := newLogModel(t, "line1\n")
	nm, _ := m.handleKey(key("l"))
	m = nm.(model)

	nm, _ = m.handleLogKey(key("f")) // follow off
	m = nm.(model)
	if m.logFollow {
		t.Fatal("f로 follow가 꺼지지 않음")
	}

	nm, _ = m.handleLogKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(model)
	if m.mode != modeList {
		t.Fatalf("esc 후 mode=%v, modeList 기대", m.mode)
	}
}
