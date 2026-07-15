# crc 기능 명세

각 명령이 **무엇을 받고·어떻게 검증하고·무엇을 출력하고·언제 실패하는가**의 SSOT. 규칙마다 구현 상태를 `- [ ]`(미구현) / `- [x]`(구현)로 표기한다. 제품 정의는 [`../README.md`](../README.md), 설계·기술 구조는 [`ARCHITECTURE.md`](ARCHITECTURE.md).

- **상태 표기**: 표의 `상태` 열 — ✅ 구현·검증 완료 / 🚧 부분 / ⬜ 미구현(담당 마일스톤).
- **완료 처리**: 기능이 완료되면 표의 상태를 ✅로 바꾸고, 하단 *진행·우선순위*의 해당 줄을 삭제한다.

## 성공기준 (제품 판정)

이게 안 되면 나머지가 다 돼도 도구는 실패다.

- ✅ **핵심** — 새로 띄운 서버의 **spawn 자식 세션에서 폰으로 `SendUserFile`이 실제 도착**한다. env 절대 규칙(`CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`)의 존재 이유.
  - **자동 실증**: 가짜 `claude`로 자식이 받는 env를 캡처 → `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1` 주입·금지키 제거·인자·cwd 확인.
  - **실기기 검증 완료**: `crc up`으로 이 저장소를 띄우고 폰 앱에서 자식 세션 생성 → `SendUserFile`로 README 전송 → 폰에 파일 도착·미리보기 확인. "환경변수 넣었으니 되겠지"가 아니라 실제 도착으로 통과.
- ✅ 등록한 서버의 상태(running/stopped/dead)가 실제 프로세스와 일치. (실제 프로세스로 up→running→down→stopped, 죽은 pid→dead 검증)

## 명령 명세

### `crc add <path> [name]`
워크스페이스 등록. `name` 기본값 = `path`의 basename.

| 규칙 | 동작 | 상태 |
|---|---|---|
| path 절대경로 변환 | 상대경로 → 절대경로로 정규화 | ✅ |
| path 존재+디렉토리 검증 | 없는 경로·파일(디렉토리 아님)이면 거부 | ✅ |
| 이름 중복 거부 | name을 명시했는데 이미 있으면 에러 | ✅ |
| 이름 자동 유니크 | name 미지정 시 basename 충돌을 부모명 접두로 해소(`references-budget-master`) | ✅ |
| 경로 중복 거부 | 같은 path가 다른 name으로도 등록돼 있으면 에러 | ✅ |
| 출력 | `등록: <name> → <path>` | ✅ |

### `crc rm <name>`
등록 삭제.

| 규칙 | 동작 | 상태 |
|---|---|---|
| 존재 검증 | 없는 name이면 에러 | ✅ |
| 실행 중 처리 | 떠 있으면 정지 후 삭제 | ✅ |
| 삭제 반영 | workspaces.json에서 제거 | ✅ |

### `crc ls`
등록 목록 출력.

| 규칙 | 동작 | 상태 |
|---|---|---|
| 빈 목록 | `(등록된 워크스페이스 없음)` | ✅ |
| 목록 | `<name>  <status>  <업타임>  <path>` 한 줄씩 | ✅ |
| 상태 열 | running/stopped/dead 표기 | ✅ |
| 업타임 | running이면 pid mtime 기준 경과 시간(`↑ 12m`) | ✅ |

### `crc up [name...]`
서버 시작. 인자 없으면 전체.

| 규칙 | 동작 | 상태 |
|---|---|---|
| env 절대 규칙 | `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1` 주입 + 금지키 `CLAUDE_CODE_REMOTE` 제거 | ✅ |
| detached 실행 | `Setsid`로 부모 죽어도 생존, 로그 파일로 리다이렉트 | ✅ |
| PID 기록 | `<name>.pid` 기록 | ✅ |
| 이미 실행 중 | 중복 기동 거부 | ✅ |

### `crc down [name...]`
서버 정지. 인자 없으면 전체.

| 규칙 | 동작 | 상태 |
|---|---|---|
| 정지 | SIGTERM → 잔존 시 SIGKILL | ✅ |
| pid 정리 | `<name>.pid` 삭제 | ✅ |

### `crc status`
전체 상태 판정 출력.

| 규칙 | 동작 | 상태 |
|---|---|---|
| 판정 | `kill(pid,0)` → running/stopped/dead. **좀비는 dead**(darwin `sysctl`로 감지 — `kill(0)`은 좀비도 통과하므로) | ✅ |

### `crc log <name>`
로그 tail -f.

| 규칙 | 동작 | 상태 |
|---|---|---|
| follow | `<name>.log`를 실시간 추적(폴링, 의존성 0), Ctrl-C 종료 | ✅ |
| dead 원인 | 조기종료 서버의 원인이 로그에서 보임 | ✅ |
| 미등록/인자 없음 | 이름 없거나 미등록이면 에러 | ✅ |

### `crc fg <name>`
포그라운드 실행(현재 터미널 연결).

| 규칙 | 동작 | 상태 |
|---|---|---|
| TTY 연결 | stdin/out/err를 현재 터미널에 연결, env 동일, `Setsid` 없음 | ✅ |
| pid 미기록 | 눈앞 임시 실행 — pid 파일 안 만듦(`status`에 안 뜸) | ✅ |
| 이미 실행 중 | detached로 떠 있으면 중복 기동 거부 | ✅ |

### `crc scan [root]`
`root`(기본 **현재 폴더**) 아래 CLAUDE.md/.claude 있는 디렉토리를 **후보로 나열**(자동 등록 아님 — 개인 경로 하드코딩 회피). 실제 등록은 `add` 또는 TUI 추가 화면.

| 규칙 | 동작 | 상태 |
|---|---|---|
| 수집 | `WalkDir`로 CLAUDE.md/.claude 보유 디렉토리만 | ✅ |
| 스킵 | 숨김 디렉토리(`.git`·`.cache` 등)·`node_modules` 제외, 깊이 상한(기본 4) | ✅ |
| 중첩 제외 | 하위에서 프로젝트 발견 시 서브트리 스킵(최상위만). root 자신 마커는 하위 탐색 유지 | ✅ |
| 등록 표시 | 이미 등록된 후보는 `(등록됨)` 표기 | ✅ |
| 후보 없음 | 안내 문구 출력 | ✅ |

### `crc` (TUI)
bubbletea 상태판.

| 규칙 | 동작 | 상태 |
|---|---|---|
| 상태색 | running=초록 / stopped=회색 / dead=빨강 (마커 ●/○/✗) | ✅ |
| 키맵(핵심) | ↑↓/jk 이동, ↵ 토글, A 전체시작/x 전체정지, r 새로고침, q 종료 | ✅ |
| 키맵(추가) | a → 추가 화면: 경로 입력창 + scan 후보 `space` 멀티선택 → `enter` 일괄 등록, tab 포커스 전환, esc 취소. 스캔은 백그라운드(UI 안 멈춤), 입력창에 경로를 넣으면 프로젝트면 등록·아니면 그 아래로 재스캔 | ✅ |
| 키맵(조작) | l 로그뷰(viewport, follow 토글)·f fg(`tea.ExecProcess`)·d 삭제(y 확인) | ✅ |
| 갱신 | tick으로 상태 주기 갱신(2s) | ✅ |

## 진행 · 우선순위

완료된 항목은 위 표에서 ✅로 바꾸고 여기서 줄을 삭제한다.

- **완료(전 마일스톤)**: M0(스캐폴드·config·ls·add), M0.5(품질 인프라: golangci-lint·lefthook·commit-msg 훅·테스트), M1(등록 CRUD), M2(생명주기 `up`/`down`/`status`·env 절대 규칙·CLI E2E testscript), M3(로그·포그라운드: `log` follow·`fg`), M4(bubbletea 상태판·토글·전체·tick·스크롤), M5(scan 후보 발견·중첩 제외·이름 유니크), M5.5(TUI 추가 화면), M4.5(TUI 조작: `l` viewport 로그·`f` fg·`d` 삭제), M6(업타임 표시·좀비→dead 판정·`~/.local/bin` 설치).
- **남은 수동 검증**: M2 핵심 성공기준의 폰 `SendUserFile` 도착(위 성공기준 참조 — 이미 1회 실기기 확인). 신규 조작(TUI 토글/fg)으로 띄운 서버에서도 폰 도착 재확인 권장.
