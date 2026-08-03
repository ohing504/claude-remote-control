# crc 아키텍처 문서

`claude-remote-control`(도구명 `crc`)의 **설계 결정·기술 구조 SSOT**. 제품 정의(왜·무엇·범위)는 [`../README.md`](../README.md), 기능 명세는 [`SPEC.md`](SPEC.md)가 담당한다. 이 문서는 "어떻게 설계했나"만 다룬다.

- **프로젝트명**: `claude-remote-control` / **바이너리명**: `crc` (`rc`는 너무 광범위·흔함 → 폐기)

## 설계 결정

- **tmux 배제** — remote-control 서버는 로컬 TTY 불필요 데몬이라(조작은 폰·웹) tmux 핵심 가치인 양방향 attach가 무의미. 남는 건 백그라운드 유지·상태·로그뿐이고 셋 다 Go 표준 라이브러리로 되니 tmux를 뺀다. 이건 *tmux가 이 용도에 안 맞아서*지 "의존성을 0으로"라는 규칙이 아니다 — **값어치 있는 Go 패키지는 쓴다**(TUI에 bubbletea/lipgloss, 필요하면 메뉴바에 systray 등). 지향점은 "설치가 번거로운 외부 런타임 없이 바이너리로 배포".
- **env 주입 절대 규칙** — 서버 시작 모든 경로에 `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`을 주입하고 `CLAUDE_CODE_REMOTE`는 제거한다. 성공기준은 [`SPEC.md`](SPEC.md).
  - **TYPE=1이 켜는 것(실측 2.1.209)**: `SendUserFile`(파일→폰)과 첨부파일 업로드 경로뿐. remote 기능 *전체*가 아니라 이 둘만이다 — push 알림·`Workflow` 도구·`/rc` 명령 등 나머지는 실제 브리지 연결(`replBridgeActive`) 또는 `CLAUDE_CODE_REMOTE`가 필요하다. spawn된 자식은 브리지 밖이라, 이 env로 "브리지 연결된 척" 흉내 내 도구만 켠다. 빠지면 자식에서 `SendUserFile`이 안 떠 도구 존재 이유가 소멸.
  - **REMOTE 제거는 필수(실측)**: `CLAUDE_CODE_REMOTE=1`이 환경에 있으면 claude가 자신을 remote 자식으로 인식해 `Error: Remote Control is not available inside a cloud session.`로 기동을 거부한다. 로컬/cloud 무관하게 env 존재만으로 거부되므로, crc가 remote 세션 안에서 실행될 때 부모 env로 새어들지 않게 반드시 제거한다(`buildEnv`가 담당).
- **Go 단일 바이너리 + bubbletea/lipgloss** — 배포 바이너리 하나, 실시간 상태판.
- **포그라운드 모드** — 미심쩍을 때 백그라운드 대신 현재 터미널에 붙여 로그 보며 실행(`crc fg`).
- **로그는 서버 화면이 아니라 `--debug-file`로 받는다** — 서버 stdout을 파일로 받던 방식은 실측에서 1.46MB 파일의 고유한 줄이 11줄(663바이트)뿐이었다. 연결 상태 화면을 1초에 한 번꼴로 다시 그려 같은 6줄이 3,967번 쌓인 것으로, 12일이면 400MB 규모다. 게다가 세션이 안 붙은 서버는 화면을 다시 그릴 일이 없어 로그가 자라지 않으므로, "정상적으로 조용한 것"과 "멎어서 조용한 것"이 구분되지 않는다. `--debug-file`은 유휴여도 폴링 기록이 타임스탬프와 함께 남아 **마지막 기록 시각으로 서버가 언제까지 살아 움직였는지 판별**할 수 있고, 용량은 1/50 수준이다. 서버가 이어쓰기로 기록하므로 재시작 전 기록도 보존된다(무한 성장은 시작 시 상한 검사로 막는다). stdout은 버리되 **stderr는 남긴다** — 기동 거부(`Error: Remote Control is not available inside a cloud session.`) 같은 실패 원인이 나오는 유일한 곳이고, 정상 기동 시에는 비어 있다(실측).
- **NFC 정규화** — macOS 파일명은 NFD(자모 분해)라 한글이 깨져 보이고 이름 매칭이 어긋난다. scan 후보 이름과 화면 표시(경로 포함)를 NFC로 정규화한다(`golang.org/x/text/unicode/norm`).

## 조사 결론 (4축)

1. **서버 1개 = 디렉토리 1개** (구조적). `--spawn=worktree`는 같은 repo 격리일 뿐 다른 프로젝트를 못 담음 → 프로젝트마다 서버 하나씩. 이게 이 도구의 존재 이유.
2. **env 게이트** — 자식은 도구를 여는 코드 경로(`replBridgeActive`)가 구조적으로 없어 env 게이트를 타야 함. `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE`만 게이트를 열고 cloud 거부 가드엔 안 걸림. Claude Code 2.1.208 실측.
3. **기성 tmux 매니저 불충분** — sesh·smug·zellij·tmuxinator·tmuxp 중 "고정 세트 상태판 + 토글 + 로그"를 통째로 주는 것 없음. 각자 조각만 → 얇게 직접, tmux조차 없이.
4. **Orca** — 채택 X(폰 주도 spawn 없음, 데스크톱 주도). 상태 인박스 UX(한 리스트 + 3상태)만 차용.
5. **환경 라벨 커스텀 불가(실측 2.1.209)** — 폰/데스크탑 "환경 선택" 목록의 라벨은 claude가 `실제 디렉토리 basename + git 브랜치`로 자동 결정한다. `--name`(세션 계층이라 환경 라벨과 무관), `--remote-control-session-name-prefix`, 심링크(claude가 physical path로 resolve) **모두 무영향**을 실측 확인. ⟹ crc는 이 화면 라벨을 못 바꾼다. 서로 다른 경로라도 basename이 같으면(`~/a/my-app` vs `~/b/my-app`) 화면에서 같은 라벨로 충돌하며, crc의 경로 중복 거부는 *완전 동일 경로*만 막아 basename 충돌은 못 막는다. 따라서 crc는 화면 라벨 관리자가 아니라 **로컬 오케스트레이터 + 대조 도구**다(로컬 유일 name·전체 경로로 식별, 난립·좀비 정리).
6. **로컬 실재 ≠ relay 표시(실측)** — 로컬 서버가 죽어도 claude.ai relay가 세션 레코드를 즉시 지우지 않아, 폰/데스크탑 목록엔 좀비가 잠시 남는다. crc의 상태 판정은 로컬 pid(`kill(pid,0)` + darwin 좀비 sysctl) 기준이라 정확하고(stopped/dead), 화면 잔상은 relay 지연이라 crc가 못 고친다. 이 갈림이 오히려 "로컬 진실을 보여주는" crc의 존재 이유를 강화한다.

## 아키텍처 요약

### 데이터 배치

```
~/.config/crc/workspaces.json    # 등록: { "workspaces": [{ "name": "...", "path": "..." }] }
~/.local/state/crc/<name>.pid    # 실행 중 PID
~/.local/state/crc/logs/<name>/server.log   # 서버 진단 로그 (서버가 --debug-file로 직접 기록)
~/.local/state/crc/logs/<name>/server.err   # 서버 stderr (기동 실패 원인)
```

config는 temp 파일 → rename으로 atomic하게 쓴다(토글마다 갱신되므로 크래시에 파일이 깨지지 않게). config 디렉토리는 `XDG_CONFIG_HOME` 존중, 권한 `0700`.

### 프로세스 생명주기

- **시작**: `exec.Command("claude","remote-control","--name",name,"--debug-file",state.LogPath(dir,name))`, `cmd.Dir=path`, `Env += CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`, stdout은 버리고 stderr는 `logs/<name>/server.err`로, `SysProcAttr{Setsid:true}`(부모 죽어도 생존), PID 기록.
- **판정**: PID `kill(pid,0)` → running / stopped / dead(pid 있으나 프로세스 없음=조기종료). **단 `kill(0)`은 좀비도 통과**하므로, crc가 `Setsid` 자식을 wait하지 않아 생기는 좀비를 running으로 오판하지 않도록 darwin은 `sysctl("kern.proc.pid")`의 `P_stat==SZOMB`로 좀비를 감지해 **dead로 판정**한다(그 외 플랫폼은 signal 0만; `proc_darwin.go`/`proc_other.go`로 분리).
- **업타임**: running이면 pid 파일 mtime(=시작 시각) 기준 경과 시간을 파생 표시(`Uptime`).
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

`filepath.WalkDir`로 CLAUDE.md/.claude 있는 디렉토리 수집. fd/find 불필요(macOS 실측 홈 스캔 0.15초대).

- **스킵**: 모든 숨김 디렉토리(`.` 접두 — `.git`·`.cache`·`.Trash` 등) + `node_modules`, 깊이 상한(기본 4). `.claude` 마커는 부모에서 `Stat`으로 감지하므로 그 안에 진입할 필요가 없다.
- **중첩 미스킵**: 프로젝트를 찾아도 서브트리를 스킵하지 않는다 — 상위가 마커를 가져도(예: `~/workspace/.claude`) 하위 프로젝트를 가리지 않기 위함. 중첩 노이즈는 깊이 상한으로만 제한하고, 취사선택은 TUI 멀티선택에 맡긴다.
- **심링크**: `WalkDir`는 심링크를 따라가지 않으므로(순환 위험), 심링크가 프로젝트 디렉토리를 가리키면 후보에 넣되 그 안으로는 진입하지 않는다(예: `~/notes`).
- **NFC 정규화**: macOS 파일명은 NFD(자모 분해)라, 그대로 두면 한글이 깨져 보이고 이름 매칭도 어긋난다. 후보 이름을 NFC로 정규화한다.
- **비동기(TUI)**: 넓은 트리(홈 등)에서 UI가 멈추지 않도록 추가 화면의 스캔은 goroutine(`tea.Cmd`)으로 돌리고 완료 시 목록을 채운다.

### TUI (기술 접근)

bubbletea 리스트 + tick 갱신, lipgloss 상태 색. 키맵·상태색·상호작용 명세는 [`SPEC.md`](SPEC.md).
