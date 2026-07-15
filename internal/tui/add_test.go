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

// enterAddSync는 추가 화면에 진입하고 백그라운드 스캔(scanCmd)을 동기적으로 완료시킨다.
func enterAddSync(t *testing.T, m model) model {
	t.Helper()
	nm, cmd := m.handleKey(key("a"))
	m = nm.(model)
	if cmd == nil {
		t.Fatal("scan 커맨드가 없음")
	}
	nm2, _ := m.Update(cmd()) // scanCmd 실행 → candsMsg 반영
	return nm2.(model)
}

// a로 추가 화면 진입 → 후보 두 개가 스캔되는지.
func TestEnterAddScansCandidates(t *testing.T) {
	m := newAddModel(t)
	// 진입 직후에는 스캔 중이어야(비동기).
	nm, _ := m.handleKey(key("a"))
	if am := nm.(model); am.mode != modeAdd || !am.scanning {
		t.Fatalf("진입 직후 mode=%v scanning=%v, modeAdd+scanning 기대", am.mode, am.scanning)
	}
	// 스캔 완료 후 후보 두 개.
	am := enterAddSync(t, m)
	if am.scanning {
		t.Fatal("스캔 완료 후에도 scanning=true")
	}
	if len(am.cands) != 2 {
		t.Fatalf("후보 %d개, 2개 기대", len(am.cands))
	}
}

// 후보를 space로 선택하고 enter로 등록하면 config에 반영되는지.
func TestSelectAndRegister(t *testing.T) {
	m := newAddModel(t)
	m = enterAddSync(t, m)

	// 첫 후보 선택(space), 커서 내려 둘째도 선택.
	nm, _ := m.handleAddKey(key(" "))
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
	// 등록 완료 → 목록 화면으로 복귀해야.
	if m.mode != modeList {
		t.Fatalf("등록 후 mode=%v, modeList 기대(복귀 안 함)", m.mode)
	}
	// 등록 후 후보는 already로 전환.
	for _, c := range m.cands {
		if !c.already {
			t.Fatalf("등록 후 후보가 already 아님: %+v", c)
		}
	}
}

// 선택 없이 enter면 목록으로 나가지 않고 안내만 표시.
func TestRegisterNoneStaysInAdd(t *testing.T) {
	m := newAddModel(t)
	m = enterAddSync(t, m)

	nm, _ := m.handleAddKey(key("enter")) // 아무것도 선택 안 함
	m = nm.(model)
	if m.mode != modeAdd {
		t.Fatalf("선택 없이 enter인데 mode=%v, modeAdd 유지 기대", m.mode)
	}
	if m.addMsg == "" {
		t.Fatal("안내 메시지가 비어 있음")
	}
}

// 입력창에 프로젝트 아닌 디렉토리를 넣으면 그 아래로 좁혀 재스캔한다.
func TestSubmitInputRescansDir(t *testing.T) {
	root := t.TempDir()
	// root 아래에 프로젝트 하위 폴더를 둔다(root 자신은 마커 없음).
	sub := filepath.Join(root, "nested", "proj-x")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newAddModel(t) // cwd는 다른 트리
	m.mode = modeAdd

	nm, cmd := m.submitInput(root)
	m = nm.(model)
	if !m.scanning || m.addRoot != root {
		t.Fatalf("재스캔 진입 실패: scanning=%v root=%q", m.scanning, m.addRoot)
	}
	nm2, _ := m.Update(cmd()) // 스캔 완료
	m = nm2.(model)
	if len(m.cands) != 1 || m.cands[0].c.Name != "proj-x" {
		t.Fatalf("재스캔 후보=%+v, proj-x 하나 기대", m.cands)
	}
}

// 입력창에 프로젝트 경로를 넣으면 바로 등록하고 목록으로 복귀한다.
func TestSubmitInputRegistersProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newAddModel(t)
	m.mode = modeAdd

	nm, _ := m.submitInput(root)
	m = nm.(model)
	if m.mode != modeList {
		t.Fatalf("프로젝트 등록 후 mode=%v, modeList 기대", m.mode)
	}
	cfg, _ := config.Load()
	if cfg.FindByPath(root) == nil {
		t.Fatal("입력 경로가 등록되지 않음")
	}
}

// 이미 등록된 후보는 space로 선택되지 않는지.
func TestAlreadyRegisteredNotSelectable(t *testing.T) {
	m := newAddModel(t)
	m = enterAddSync(t, m)
	m.cands[0].already = true

	nm, _ := m.handleAddKey(key(" ")) // 커서는 0
	m = nm.(model)
	if m.cands[0].selected {
		t.Fatal("등록된 후보가 선택됨")
	}
}
