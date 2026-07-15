# crc 기능 명세

각 명령이 **무엇을 받고·어떻게 검증하고·무엇을 출력하고·언제 실패하는가**의 SSOT. 규범 문장으로 crc의 동작을 규정한다. 제품 정의는 [`../README.md`](../README.md), 설계·기술 구조는 [`ARCHITECTURE.md`](ARCHITECTURE.md).

## 성공기준

제품 판정의 뿌리. 이게 안 되면 나머지가 다 돼도 도구는 실패다.

- **핵심** — 새로 띄운 서버의 spawn 자식 세션에서 폰으로 `SendUserFile`이 실제 도착한다. env 절대 규칙(`CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`)의 존재 이유다. "환경변수 넣었으니 되겠지"가 아니라 실제 도착으로만 통과한다.
- **상태 정합** — 등록한 서버의 상태(running/stopped/dead)가 실제 프로세스와 일치한다.

## `crc add <path> [name]`

워크스페이스를 등록한다. `name` 기본값은 `path`의 basename.

- 상대경로는 절대경로로 정규화한다.
- 존재하지 않거나 디렉토리가 아닌 `path`는 거부한다.
- `name` 미지정 시 basename 충돌은 부모명 접두로 자동 해소한다(`references-budget-master`).
- `name`을 명시했는데 이미 있으면 에러. 같은 `path`가 다른 `name`으로 등록돼 있어도 에러.
- 성공 시 `등록: <name> → <path>`를 출력한다.

## `crc rm <name>`

등록을 삭제한다.

- 없는 `name`이면 에러.
- 떠 있으면 정지한 뒤 삭제한다.
- `workspaces.json`에서 제거한다.

## `crc ls`

등록 목록을 출력한다.

- 비어 있으면 `(등록된 워크스페이스 없음)`.
- 있으면 `<name>  <status>  <업타임>  <path>`를 한 줄씩. status는 running/stopped/dead.
- 업타임은 running일 때만 pid 파일 mtime 기준 경과 시간(`↑ 12m`).

## `crc up [name...]`

서버를 시작한다. 인자가 없으면 전체.

- env 절대 규칙: `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`을 주입하고 금지키 `CLAUDE_CODE_REMOTE`를 제거한다.
- `Setsid`로 부모가 죽어도 생존하며, stdout/stderr는 로그 파일로 리다이렉트한다.
- `<name>.pid`에 PID를 기록한다.
- 이미 실행 중이면 중복 기동을 거부한다.

## `crc down [name...]`

서버를 정지한다. 인자가 없으면 전체.

- SIGTERM을 보내고, 잔존하면 SIGKILL.
- `<name>.pid`를 삭제한다.

## `crc status`

전체 상태를 판정해 출력한다.

- `kill(pid,0)`으로 running/stopped/dead를 가른다.
- 좀비는 dead로 판정한다. `kill(0)`은 좀비도 통과하므로 darwin은 `sysctl`로 좀비를 감지한다.

## `crc log <name>`

로그를 `tail -f`처럼 추적한다.

- `<name>.log`를 실시간 추적한다(폴링, 외부 의존성 없음). Ctrl-C로 종료.
- 조기종료(dead) 서버의 원인이 로그에서 보인다.
- 이름이 없거나 미등록이면 에러.

## `crc fg <name>`

포그라운드로 실행한다(현재 터미널에 연결).

- stdin/out/err를 현재 터미널에 연결하고 env는 `up`과 동일. `Setsid`는 없다.
- 눈앞 임시 실행이므로 pid 파일을 만들지 않는다(`status`에 안 뜬다).
- 이미 detached로 떠 있으면 중복 기동을 거부한다.

## `crc scan [root]`

`root`(기본 현재 폴더) 아래 CLAUDE.md/.claude 있는 디렉토리를 **후보로 나열**한다. 자동 등록이 아니다(개인 경로 하드코딩 회피) — 실제 등록은 `add` 또는 TUI 추가 화면.

- `WalkDir`로 CLAUDE.md 또는 `.claude`를 가진 디렉토리만 수집한다.
- 숨김 디렉토리(`.git`·`.cache` 등)와 `node_modules`를 제외하고, 깊이 상한(기본 4)을 둔다.
- 중첩을 허용한다 — 상위가 마커를 가져도(예: `~/workspace/.claude`) 하위 프로젝트를 가리지 않는다. 중첩 노이즈는 깊이 상한으로만 제한한다.
- 심링크가 프로젝트를 가리키면 후보에 넣되 그 안으로는 진입하지 않는다(순환 방지, 예: `~/notes`).
- 후보 이름을 NFC로 정규화한다(macOS NFD 한글 깨짐·매칭 어긋남 방지).
- 이미 등록된 후보는 `(등록됨)`으로 표기한다.
- 후보가 없으면 안내 문구를 출력한다.

## `crc` (TUI)

인자 없이 실행하면 bubbletea 상태판을 띄운다.

- 상태색: running=초록(●) / stopped=회색(○) / dead=빨강(✗).
- 핵심 키맵: `↑↓`/`jk` 이동, `↵` 토글, `A` 전체시작, `x` 전체정지, `r` 새로고침, `q` 종료.
- 추가(`a`): 경로 입력창 + scan 후보를 `space`로 멀티선택 → `enter` 일괄 등록. `tab` 포커스 전환, `esc` 취소. 스캔은 백그라운드로 돌아 UI가 멈추지 않으며, 입력창에 경로를 넣으면 프로젝트면 등록, 아니면 그 아래로 재스캔한다.
- 조작: `l` 로그뷰(viewport, follow 토글), `f` fg(`tea.ExecProcess`), `d` 삭제(`y` 확인).
- 상태를 tick으로 주기 갱신한다(2s).
