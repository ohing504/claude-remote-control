# crc 아키텍처 문서

`claude-remote-control`(도구명 `crc`)의 **설계 결정·기술 구조 SSOT**. 제품 정의(왜·무엇·범위)는 [`../README.md`](../README.md), 기능 명세·구현 상태는 [`SPEC.md`](SPEC.md)가 담당한다. 이 문서는 "어떻게 설계했나"만 다룬다.

- **프로젝트명**: `claude-remote-control` / **바이너리명**: `crc` (`rc`는 너무 광범위·흔함 → 폐기)

## 설계 결정

- **tmux 배제** — remote-control 서버는 로컬 TTY 불필요 데몬이라(조작은 폰·웹) tmux 핵심 가치인 양방향 attach가 무의미. 남는 건 백그라운드 유지·상태·로그뿐이고 셋 다 Go 표준 라이브러리로 됨 → **런타임 의존성 0**.
- **env 주입 절대 규칙** — 서버 시작 모든 경로에 `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`. 빠지면 spawn된 자식에서 `SendUserFile`이 안 떠 도구 존재 이유가 소멸(성공기준은 [`SPEC.md`](SPEC.md)). `CLAUDE_CODE_REMOTE=1`은 금지(서버 기동 거부).
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
~/.config/crc/workspaces.json    # 등록: { "workspaces": [{ "name": "...", "path": "..." }] }
~/.local/state/crc/<name>.pid    # 실행 중 PID
~/.local/state/crc/<name>.log    # stdout+stderr
```

config는 temp 파일 → rename으로 atomic하게 쓴다(토글마다 갱신되므로 크래시에 파일이 깨지지 않게). config 디렉토리는 `XDG_CONFIG_HOME` 존중, 권한 `0700`.

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

### CLAUDE.md 스캔

`filepath.WalkDir`로 CLAUDE.md/.claude 있는 디렉토리 수집(깊이 상한, `.git`·`node_modules` 스킵). fd/find 불필요.

### TUI (기술 접근)

bubbletea 리스트 + tick 갱신, lipgloss 상태 색. 키맵·상태색·상호작용 명세는 [`SPEC.md`](SPEC.md).
