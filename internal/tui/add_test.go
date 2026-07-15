package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/config"
)

// 격리 환경 + cwd 아래 후보 두 개를 만든 model을 반환한다.
func newAddModel(t *testing.T) model {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home+"/cfg")
	t.Setenv("XDG_STATE_HOME", home+"/state")

	// cwd를 후보가 있는 트리로 옮긴다.
	root := t.TempDir()
	for _, n := range []string{"proj-a", "proj-b"} {
		if err := os.MkdirAll(filepath.Join(root, n), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, n, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	chdir(t, root)

	m := model{dir: home + "/state", input: textinput.New()}
	return m
}

// chdir는 테스트 동안 작업 디렉토리를 바꾸고 종료 시 되돌린다.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// a로 추가 화면 진입 → 후보 두 개가 스캔되는지.
func TestEnterAddScansCandidates(t *testing.T) {
	m := newAddModel(t)
	nm, _ := m.handleKey(key("a"))
	am := nm.(model)
	if am.mode != modeAdd {
		t.Fatalf("mode=%v, modeAdd 기대", am.mode)
	}
	if len(am.cands) != 2 {
		t.Fatalf("후보 %d개, 2개 기대", len(am.cands))
	}
}

// 후보를 space로 선택하고 enter로 등록하면 config에 반영되는지.
func TestSelectAndRegister(t *testing.T) {
	m := newAddModel(t)
	nm, _ := m.handleKey(key("a"))
	m = nm.(model)

	// 첫 후보 선택(space), 커서 내려 둘째도 선택.
	nm, _ = m.handleAddKey(key(" "))
	m = nm.(model)
	nm, _ = m.handleAddKey(key("j"))
	m = nm.(model)
	nm, _ = m.handleAddKey(key(" "))
	m = nm.(model)

	// enter로 일괄 등록.
	nm, _ = m.handleAddKey(key("enter"))
	m = nm.(model)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Workspaces) != 2 {
		t.Fatalf("등록 %d개, 2개 기대 (msg=%q)", len(cfg.Workspaces), m.addMsg)
	}
	// 등록 후 후보는 already로 전환.
	for _, c := range m.cands {
		if !c.already {
			t.Fatalf("등록 후 후보가 already 아님: %+v", c)
		}
	}
}

// 이미 등록된 후보는 space로 선택되지 않는지.
func TestAlreadyRegisteredNotSelectable(t *testing.T) {
	m := newAddModel(t)
	nm, _ := m.handleKey(key("a"))
	m = nm.(model)
	m.cands[0].already = true

	nm, _ = m.handleAddKey(key(" ")) // 커서는 0
	m = nm.(model)
	if m.cands[0].selected {
		t.Fatal("등록된 후보가 선택됨")
	}
}
