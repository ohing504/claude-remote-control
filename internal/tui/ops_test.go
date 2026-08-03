package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"crc/internal/config"
)

// 격리 환경에 워크스페이스 두 개를 심은 목록 model을 만든다.
func newListModel(t *testing.T) model {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home+"/cfg")
	t.Setenv("XDG_STATE_HOME", home+"/state")
	cfg := &config.Config{Workspaces: []config.Workspace{
		{Name: "proj-a", Path: "/tmp/proj-a"},
		{Name: "proj-b", Path: "/tmp/proj-b"},
	}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("픽스처: %v", err)
	}
	m := model{dir: home + "/state", input: textinput.New()}
	m.rows, _ = loadRows(m.dir)
	return m
}

// d → y로 커서 워크스페이스가 등록에서 사라지는지.
func TestDeleteConfirmed(t *testing.T) {
	m := newListModel(t)
	nm, _ := m.handleKey(key("d"))
	m = nm.(model)
	if !m.pendingDelete {
		t.Fatal("d 후 삭제 확인 대기 아님")
	}
	nm, cmd := m.handleKey(key("y"))
	m = nm.(model)
	if m.pendingDelete {
		t.Fatal("y 후에도 확인 대기 중")
	}
	// refreshCmd가 반환돼야(삭제 반영).
	if cmd == nil {
		t.Fatal("삭제 후 refresh 커맨드 없음")
	}
	cfg, _ := config.Load()
	if cfg.Find("proj-a") != nil {
		t.Fatal("proj-a가 삭제되지 않음")
	}
	if cfg.Find("proj-b") == nil {
		t.Fatal("엉뚱하게 proj-b가 사라짐")
	}
}

// d → 다른 키면 취소되어 아무것도 지워지지 않는지.
func TestDeleteCancelled(t *testing.T) {
	m := newListModel(t)
	nm, _ := m.handleKey(key("d"))
	m = nm.(model)
	nm, _ = m.handleKey(key("n")) // 취소
	m = nm.(model)
	if m.pendingDelete {
		t.Fatal("취소 후에도 확인 대기 중")
	}
	cfg, _ := config.Load()
	if len(cfg.Workspaces) != 2 {
		t.Fatalf("취소했는데 %d개로 변함", len(cfg.Workspaces))
	}
}

// 재시작이 실패하면 화면에 에러가 뜨는지(조용히 묻히지 않는지).
func TestRestartFailureSurfaces(t *testing.T) {
	m := newListModel(t)
	t.Setenv("PATH", "") // claude를 찾을 수 없게 만들어 시작을 실패시킨다

	msg, ok := restartCmd(m.dir, m.rows[0])().(refreshMsg)
	if !ok {
		t.Fatal("refreshMsg가 아님")
	}
	if msg.err == nil {
		t.Fatal("재시작 실패가 err로 안 올라옴")
	}
}

// 토글(시작)이 실패해도 마찬가지.
func TestToggleFailureSurfaces(t *testing.T) {
	m := newListModel(t)
	t.Setenv("PATH", "")

	msg, ok := toggleCmd(m.dir, m.rows[0])().(refreshMsg)
	if !ok {
		t.Fatal("refreshMsg가 아님")
	}
	if msg.err == nil {
		t.Fatal("시작 실패가 err로 안 올라옴")
	}
}
