# crc 아키텍처 문서

`claude-remote-control`(도구명 `crc`)의 설계 결정·아키텍처 SSOT. 새 세션이 이 문서를 읽고 개발을 이어받는다. 구현 순서·진행 트래킹은 [`ROADMAP.md`](ROADMAP.md).

- **프로젝트명**: `claude-remote-control`
- **도구(바이너리)명**: `crc` (`rc`는 너무 광범위·흔함 → 폐기)
- **상태**: 설계만. 코드 없음. 구현 순서·진행은 [`ROADMAP.md`](ROADMAP.md).

## 목표·성공기준

- **목표**: 등록한 워크스페이스가 지금 떠 있는지 한눈에 보고, 키 하나로 서버를 토글하고, 로그를 본다.
- **핵심 성공기준**: 새로 띄운 서버의 **spawn 자식에서 폰으로 `SendUserFile`이 실제 도착**한다(env 절대 규칙의 존재 이유). 이게 안 되면 나머지가 다 돼도 도구는 실패.
- **비목표**: 로컬 TTY attach(폰·웹으로 조작), 세션 내부 조작, tmux 대체 일반화.

## 왜 만드나 (pain points)

폰·claude.ai/code로 여러 워크스페이스(저장소 루트 + 하위 스튜디오들 + 다른 프로젝트, 보통 3~4곳)를 오가며 작업하는데:

1. **실행 스크립트가 갈라져 있었다** — 단일 포그라운드용·전체 실행용이 따로, 특정 저장소에만 묶임.
2. **폰으로 파일이 안 갔다** — `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1` 누락으로 spawn된 자식에서 `SendUserFile`이 안 뜸.
3. **뭐가 떠 있는지 몰랐다** — 상태를 한눈에 보고 켜고/끄는 수단 부재.

## 설계 결정

- **tmux 배제** — remote-control 서버는 로컬 TTY 불필요 데몬이라(조작은 폰·웹) tmux 핵심 가치인 양방향 attach가 무의미. 남는 건 백그라운드 유지·상태·로그뿐이고 셋 다 Go 표준 라이브러리로 됨 → **런타임 의존성 0**.
- **env 주입 절대 규칙** — 서버 시작 모든 경로에 `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`. 빠지면 도구 존재 이유 소멸. `CLAUDE_CODE_REMOTE=1`은 금지(서버 기동 거부).
- **Go 단일 바이너리 + bubbletea/lipgloss** — 배포 바이너리 하나, 실시간 상태판.
- **포그라운드 모드** — 미심쩍을 때 백그라운드 대신 현재 터미널에 붙여 로그 보며 실행(`crc fg`).

## 조사 결론 (4축)

1. **서버 1개 = 디렉토리 1개** (구조적). `--spawn=worktree`는 같은 repo 격리일 뿐 다른 프로젝트를 못 담음 → 프로젝트마다 서버 하나씩. 이게 이 도구의 존재 이유.
2. **env 게이트** — 자식은 도구를 여는 코드 경로(`replBridgeActive`)가 구조적으로 없어 env 게이트를 타야 함. `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE`만 게이트를 열고 cloud 거부 가드엔 안 걸림. Claude Code 2.1.208 실측.
3. **기성 tmux 매니저 불충분** — sesh·smug·zellij·tmuxinator·tmuxp 중 "고정 세트 상태판 + 토글 + 로그"를 통째로 주는 것 없음. 각자 조각만 → 얇게 직접, tmux조차 없이.
4. **Orca** — 채택 X(폰 주도 spawn 없음, 데스크톱 주도). 상태 인박스 UX(한 리스트 + 3상태)만 차용.

## 아키텍처 요약

### 데이터 배치

```
~/.config/crc/workspaces.json    # 등록: [{ "name": "...", "path": "..." }]
~/.local/state/crc/<name>.pid    # 실행 중 PID
~/.local/state/crc/<name>.log    # stdout+stderr
```

### 프로세스 생명주기

- **시작**: `exec.Command("claude","remote-control","--name",name)`, `cmd.Dir=path`, `Env += CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`, stdout/stderr→로그, `SysProcAttr{Setsid:true}`(부모 죽어도 생존), PID 기록.
- **판정**: PID `kill(pid,0)` → running / stopped / dead(pid 있으나 프로세스 없음=조기종료).
- **정지**: SIGTERM→(잔존 시)SIGKILL, pid 정리.
- **포그라운드**: stdin/out/err를 현재 터미널에 연결, env 동일.

```go
cmd := exec.Command("claude", "remote-control", "--name", name)
cmd.Dir = workspacePath
cmd.Env = append(os.Environ(), "CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1") // 절대 규칙
logFile, _ := os.Create(logPath)
cmd.Stdout, cmd.Stderr = logFile, logFile
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // 부모(crc)가 죽어도 서버는 산다
cmd.Start()
os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
```

### 명령 표면

```
crc                     # TUI
crc add <path> [name]   # 등록 (name 기본 basename)
crc scan [root]         # root(기본 ~/workspace) 아래 CLAUDE.md 있는 디렉토리 후보로 골라 등록
crc rm <name>
crc ls
crc up   [name...]      # 시작 (없으면 전체)
crc down [name...]      # 정지 (없으면 전체)
crc status
crc log  <name>         # tail -f
crc fg   <name>         # 포그라운드
```

### TUI

bubbletea 리스트 + tick 갱신. 키: ↑↓/jk 이동, enter 토글, l 로그(viewport), f 포그라운드, a 전체시작/x 전체정지, s scan추가, d 삭제, r 새로고침, q 종료. 상태 색: running=초록 / stopped=회색 / dead=빨강.

### CLAUDE.md 스캔

`filepath.WalkDir`로 CLAUDE.md/.claude 있는 디렉토리 수집(깊이 상한, `.git`·`node_modules` 스킵). fd/find 불필요.

## 블로그

이 프로젝트의 개발기는 별도 개인 블로그(`~/workspace/portfolio` `/blog`)에 있다. 첫 글 "여러 프로젝트를 폰에서 굴리려다 tmux를 걷어낸 이야기". 구현이 진행되면 실제 활용 내용을 더해 글을 업데이트한다.
