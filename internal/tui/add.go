package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/config"
	"crc/internal/scan"
)

// candsMsg는 백그라운드 스캔 결과다.
type candsMsg struct{ cands []addCand }

// scanCmd는 root 스캔을 goroutine으로 돌린다 — 넓은 트리(홈 등)에서 UI가 멈추지 않게.
func scanCmd(root string) tea.Cmd {
	return func() tea.Msg { return candsMsg{cands: loadCands(root)} }
}

// enterAdd는 현재 폴더를 루트로 추가 화면에 진입하고, 스캔은 백그라운드로 시작한다.
func (m model) enterAdd() (tea.Model, tea.Cmd) {
	cwd, err := os.Getwd()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.addRoot = cwd
	m.cands = nil
	m.scanning = true
	m.mode = modeAdd
	m.focus = focusList
	m.addCursor = 0
	m.addMsg = ""
	m.input.SetValue("")
	m.input.Blur()
	return m, scanCmd(cwd)
}

// exitAdd는 목록판으로 돌아가며 방금 등록분을 반영해 목록을 갱신한다.
func (m model) exitAdd() model {
	m.mode = modeList
	m.input.Blur()
	m.rows, m.err = loadRows(m.dir)
	return m
}

// loadCands는 root 아래 후보를 찾고 이미 등록된 것을 표시한다.
func loadCands(root string) []addCand {
	found, _ := scan.Find(root, scan.DefaultMaxDepth)
	registered := func(string) bool { return false }
	if cfg, err := config.Load(); err == nil {
		registered = func(p string) bool { return cfg.FindByPath(p) != nil }
	}
	out := make([]addCand, len(found))
	for i, c := range found {
		out[i] = addCand{c: c, already: registered(c.Path)}
	}
	return out
}

func (m model) handleAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.exitAdd(), nil
	case "tab":
		m.toggleFocus()
		return m, nil
	}
	if m.focus == focusInput {
		return m.handleInputKey(msg)
	}
	return m.handleCandKey(msg)
}

func (m *model) toggleFocus() {
	if m.focus == focusInput {
		m.focus = focusList
		m.input.Blur()
	} else {
		m.focus = focusInput
		m.input.Focus()
	}
}

// handleInputKey는 경로 입력창의 키를 처리한다. Enter면 입력 경로를 처리(등록 또는 재스캔).
func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		return m.submitInput(m.input.Value())
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// submitInput은 입력 경로가 프로젝트면 바로 등록하고, 그냥 디렉토리면 그 아래로 좁혀 재스캔한다.
// 홈처럼 넓은 곳에서 시작해도 입력창으로 범위를 좁힐 수 있게 한다.
func (m model) submitInput(p string) (tea.Model, tea.Cmd) {
	p = strings.TrimSpace(p)
	if p == "" {
		return m, nil
	}
	abs, err := config.ValidatePath(p)
	if err != nil {
		m.addMsg = "오류: " + err.Error()
		return m, nil
	}
	if scan.ProjectMarker(abs) != "" {
		// 프로젝트 → 등록하고 목록으로.
		if err := addPath(abs, filepath.Base(abs)); err != nil {
			m.addMsg = "오류: " + err.Error()
			return m, nil
		}
		markAlready(m.cands, abs)
		return m.exitAdd(), nil
	}
	// 프로젝트 아닌 디렉토리 → 그 아래로 좁혀 재스캔.
	m.addRoot = abs
	m.cands = nil
	m.scanning = true
	m.addMsg = ""
	m.input.SetValue("")
	m.focus = focusList
	m.input.Blur()
	return m, scanCmd(abs)
}

func (m model) handleCandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.addCursor > 0 {
			m.addCursor--
		}
	case "down", "j":
		if m.addCursor < len(m.cands)-1 {
			m.addCursor++
		}
	case " ":
		if len(m.cands) > 0 && !m.cands[m.addCursor].already {
			m.cands[m.addCursor].selected = !m.cands[m.addCursor].selected
		}
	case "enter":
		return m.registerSelected(), nil
	}
	return m, nil
}

// registerSelected는 선택된(이미 등록 아닌) 후보들을 일괄 등록한다.
func (m model) registerSelected() model {
	added, failed := 0, 0
	for i := range m.cands {
		if !m.cands[i].selected || m.cands[i].already {
			continue
		}
		if err := addPath(m.cands[i].c.Path, m.cands[i].c.Name); err != nil {
			failed++
			continue
		}
		m.cands[i].already = true
		m.cands[i].selected = false
		added++
	}
	switch {
	case added == 0 && failed == 0:
		m.addMsg = "선택된 후보 없음" // 머묾 — 아무것도 안 골랐으니
		return m
	case added == 0:
		m.addMsg = fmt.Sprintf("%d개 실패(이름 중복 등)", failed) // 전부 실패 → 머물며 알림
		return m
	default:
		return m.exitAdd() // 하나라도 등록 → 목록으로 복귀
	}
}

// addPath는 config에 워크스페이스 하나를 더해 저장한다.
// basename이 겹치면 부모명을 붙여 자동으로 유니크하게 만든다.
func addPath(abs, base string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name := cfg.UniqueName(abs, base)
	if err := cfg.Add(config.Workspace{Name: name, Path: abs}); err != nil {
		return err
	}
	return config.Save(cfg)
}

// markAlready는 후보 목록에서 해당 경로를 등록됨으로 표시한다(입력창 등록 후 반영).
func markAlready(cands []addCand, abs string) {
	for i := range cands {
		if cands[i].c.Path == abs {
			cands[i].already = true
			cands[i].selected = false
		}
	}
}
