// Package tui는 등록된 워크스페이스를 한 화면에 띄우고 토글하는 bubbletea 상태판이다.
package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"crc/internal/config"
	"crc/internal/scan"
	"crc/internal/server"
	"crc/internal/state"
)

// refreshInterval은 상태를 다시 판정하는 주기다. 사람이 보는 상태판이라 촘촘할 필요 없다.
const refreshInterval = 2 * time.Second

// mode는 화면 상태다: 목록판 ↔ 추가 화면.
type mode int

const (
	modeList mode = iota
	modeAdd
	modeLog
)

// addFocus는 추가 화면에서 키 입력을 받는 영역이다(경로 입력창 ↔ 후보 목록).
type addFocus int

const (
	focusInput addFocus = iota
	focusList
)

type row struct {
	ws     config.Workspace
	st     server.State
	reason string // dead일 때만 채우는 종료 원인(server.err 마지막 줄)
}

// addCand는 추가 화면의 scan 후보 한 줄이다.
type addCand struct {
	c        scan.Candidate
	selected bool
	already  bool // 이미 등록됨 — 선택 불가
}

type model struct {
	rows   []row
	cursor int
	dir    string // state 디렉토리(pid·log)
	err    error
	height int // 터미널 행 수(WindowSizeMsg로 갱신) — 리스트 스크롤 창 계산용
	width  int // 터미널 열 수 — 로그 viewport 폭

	pendingDelete  bool // d로 삭제 확인 대기 중
	pendingRestart bool // r로 재시작 확인 대기 중(실행 중인 서버만)

	// 로그뷰 상태.
	viewport  viewport.Model
	logName   string
	logFollow bool // tail -f처럼 새 로그를 따라감

	// 추가 화면 상태.
	mode      mode
	input     textinput.Model
	addRoot   string
	cands     []addCand
	addCursor int
	focus     addFocus
	addMsg    string // 등록 결과/에러 안내
	scanning  bool   // 후보 스캔이 백그라운드로 진행 중
}

type (
	tickMsg    time.Time
	refreshMsg struct {
		rows []row
		err  error
	}
)

// Run은 TUI를 실행한다(무인자 `crc`의 진입점).
func Run() error {
	dir, err := state.EnsureDir()
	if err != nil {
		return err
	}
	ti := textinput.New()
	ti.Placeholder = "폴더 경로 입력 후 Enter로 등록"
	ti.Prompt = "경로: "
	m := model{dir: dir, input: ti}
	m.rows, m.err = loadRows(dir)
	_, err = tea.NewProgram(m).Run()
	return err
}

// loadRows는 등록 목록을 읽어 각 워크스페이스의 실제 상태를 판정한다.
func loadRows(dir string) ([]row, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	rows := make([]row, len(cfg.Workspaces))
	for i, ws := range cfg.Workspaces {
		rows[i] = row{ws: ws, st: server.Status(state.PidPath(dir, ws.Name))}
		if rows[i].st == server.Dead {
			rows[i].reason = server.ExitReason(dir, ws.Name)
		}
	}
	return rows, nil
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func refreshCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		rows, err := loadRows(dir)
		return refreshMsg{rows: rows, err: err}
	}
}

// toggleCmd는 선택 행을 상태 반대로 전환한다(running이면 정지, 아니면 시작).
// Stop은 최대 ~2초 블로킹할 수 있어 Cmd(별도 goroutine)로 돌려 UI를 막지 않는다.
func toggleCmd(dir string, r row) tea.Cmd {
	return func() tea.Msg {
		var opErr error
		if r.st == server.Running {
			opErr = server.StopByName(dir, r.ws.Name)
		} else {
			s := server.Server{Name: r.ws.Name, Path: r.ws.Path}
			opErr = s.Start(dir)
		}
		return refreshAfter(dir, opErr)
	}
}

// refreshAfter는 조작 결과를 목록 갱신과 함께 돌려준다. 조작 실패는 사용자가 방금
// 누른 키의 결과라 목록 로드 실패보다 먼저 보여준다.
func refreshAfter(dir string, opErr error) refreshMsg {
	rows, err := loadRows(dir)
	if opErr != nil {
		err = opErr
	}
	return refreshMsg{rows: rows, err: err}
}

// restartCmd는 선택 행을 정지 후 다시 띄운다. 프로세스는 살아 있는데 응답만 멎은
// 서버를 되살리는 조작이라 stopped 상태에서도 그냥 시작으로 동작한다.
func restartCmd(dir string, r row) tea.Cmd {
	return func() tea.Msg {
		s := server.Server{Name: r.ws.Name, Path: r.ws.Path}
		return refreshAfter(dir, s.Restart(dir))
	}
}

// allCmd는 전체를 시작(up)하거나 정지(down)한다.
func allCmd(dir string, rows []row, up bool) tea.Cmd {
	return func() tea.Msg {
		started := 0
		for _, r := range rows {
			if up {
				if r.st != server.Running {
					if started > 0 {
						time.Sleep(server.StartInterval) // 등록 요청이 몰리면 429로 거부된다
					}
					s := server.Server{Name: r.ws.Name, Path: r.ws.Path}
					_ = s.Start(dir)
					started++
				}
			} else if r.st != server.Stopped {
				_ = server.StopByName(dir, r.ws.Name) // dead도 pid 파일을 정리한다
			}
		}
		rows, err := loadRows(dir)
		return refreshMsg{rows: rows, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(refreshCmd(m.dir), tick())
	case refreshMsg:
		m.rows, m.err = msg.rows, msg.err
		if m.cursor >= len(m.rows) && len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}
		return m, nil
	case candsMsg:
		m.cands = msg.cands
		m.scanning = false
		if m.addCursor >= len(m.cands) {
			m.addCursor = 0
		}
		return m, nil
	case logTickMsg:
		if m.mode != modeLog {
			return m, nil // 로그뷰를 벗어났으면 follow tick 중단
		}
		m = m.loadLog()
		return m, logTick()
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		if m.mode == modeLog {
			m.viewport.Width = msg.Width
			m.viewport.Height = logHeight(msg.Height)
		}
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeAdd:
			return m.handleAddKey(msg)
		case modeLog:
			return m.handleLogKey(msg)
		default:
			return m.handleKey(msg)
		}
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 확인 대기 중이면 y만 실행하고 나머지는 취소로 처리한다.
	if m.pendingDelete {
		m.pendingDelete = false
		if msg.String() == "y" {
			return m.doDelete()
		}
		return m, nil
	}
	if m.pendingRestart {
		m.pendingRestart = false
		if msg.String() == "y" && len(m.rows) > 0 {
			return m, restartCmd(m.dir, m.rows[m.cursor])
		}
		return m, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.rows) > 0 {
			return m, toggleCmd(m.dir, m.rows[m.cursor])
		}
	case "a":
		return m.enterAdd()
	case "A":
		return m, allCmd(m.dir, m.rows, true)
	case "x":
		return m, allCmd(m.dir, m.rows, false)
	case "d":
		if len(m.rows) > 0 {
			m.pendingDelete = true
		}
	case "f":
		return m.foreground()
	case "l":
		return m.enterLog()
	case "r":
		if len(m.rows) == 0 {
			return m, nil
		}
		// 실행 중인 서버를 재시작하면 그 아래 세션과 세션이 돌리던 명령까지 끊긴다.
		// 안 떠 있으면 잃을 게 없으므로 바로 시작한다.
		if m.rows[m.cursor].st == server.Running {
			m.pendingRestart = true
			return m, nil
		}
		return m, restartCmd(m.dir, m.rows[m.cursor])
	}
	return m, nil
}
