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

// enterAdd는 현재 폴더를 루트로 scan해 후보를 채우고 추가 화면으로 전환한다.
func (m model) enterAdd() (tea.Model, tea.Cmd) {
	cwd, err := os.Getwd()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.addRoot = cwd
	m.cands = loadCands(cwd)
	m.mode = modeAdd
	m.focus = focusList
	m.addCursor = 0
	m.addMsg = ""
	m.input.SetValue("")
	m.input.Blur()
	return m, nil
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

// handleInputKey는 경로 입력창의 키를 처리한다. Enter면 그 경로를 등록.
func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		return m.registerPath(m.input.Value()), nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// registerPath는 직접 입력한 경로를 검증·등록한다.
func (m model) registerPath(p string) model {
	p = strings.TrimSpace(p)
	if p == "" {
		return m
	}
	abs, err := config.ValidatePath(p)
	if err != nil {
		m.addMsg = "오류: " + err.Error()
		return m
	}
	if err := addPath(abs, filepath.Base(abs)); err != nil {
		m.addMsg = "오류: " + err.Error()
		return m
	}
	m.addMsg = "등록: " + filepath.Base(abs)
	m.input.SetValue("")
	markAlready(m.cands, abs) // 후보에 있으면 등록됨으로
	return m
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
		m.addMsg = "선택된 후보 없음"
	case failed == 0:
		m.addMsg = fmt.Sprintf("%d개 등록", added)
	default:
		m.addMsg = fmt.Sprintf("%d개 등록, %d개 실패(이름 중복 등)", added, failed)
	}
	return m
}

// addPath는 config에 워크스페이스 하나를 더해 저장한다.
func addPath(abs, name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
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
