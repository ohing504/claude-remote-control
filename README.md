# crc — claude-remote-control

여러 프로젝트의 `claude remote-control` 서버를 한 화면에서 켜고·끄고·상태 보고·로그 보는 **Go TUI**. tmux 없이 바이너리 하나로 돈다.

> **상태: 초기 구현 중.** 기능 명세·구현 상태는 [`docs/SPEC.md`](docs/SPEC.md), 설계 결정과 근거는 [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)가 SSOT.

## 왜 만드나

폰·브라우저(claude.ai/code)에서 여러 워크스페이스를 오가며 `claude remote-control`로 작업한다. 서버를 프로젝트마다 하나씩 띄워야 하는데, 그 관리가 번거로웠다.

- **실행 스크립트가 갈라져 있었다** — 단일 포그라운드용·전체 실행용이 따로, 게다가 특정 저장소에만 묶여 다른 프로젝트는 관리 대상 밖.
- **폰으로 파일이 안 갔다** — 서버가 spawn한 자식 세션에서 `SendUserFile` 도구가 안 떴다. 원인은 `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1` 누락(자세한 건 docs/ARCHITECTURE.md).
- **뭐가 떠 있는지 몰랐다** — 어느 서버가 살아있는지 한눈에 보고 켜고/끄는 수단이 없었다.

필요한 건 단순했다: **등록한 프로젝트가 지금 떠 있는지 한눈에 보고, 키 하나로 토글하고, 로그를 보는 것.**

## 무엇을

```
$ crc
╭─ crc ─ workspaces ──────────────────────╮
│ ● proj-a      running   ↑ 12m           │
│ ● proj-b      running   ↑ 12m           │
│ ○ proj-c      stopped                   │
│ ○ proj-d      stopped                   │
╰─ ↵ toggle  l log  f fg  a all  q quit ──╯
```

등록한 워크스페이스 목록 + 실행 상태(running/stopped/dead)를 한 화면에. 키 하나로 서버를 켜고 끄고, 로그를 보고, 미심쩍을 땐 포그라운드로 띄운다.

## 범위

- **한다** — 프로젝트마다 하나씩 띄우는 remote-control 서버(보통 3~4개)의 등록·토글·상태·로그.
- **안 한다(비목표)** — 로컬 TTY attach(조작은 폰·웹에서), 세션 내부 조작, tmux 대체 일반화, 폰/데스크탑 환경 라벨 관리(claude가 basename+브랜치로 고정 — crc 밖 영역).

기능별 상세 명세와 구현 상태는 [`docs/SPEC.md`](docs/SPEC.md)가 SSOT.

## 왜 tmux가 아니라 Go 단일 바이너리인가

remote-control 서버는 조작을 폰·웹에서 받는 **백그라운드 데몬**이다. 로컬 터미널로 attach할 일이 없다. 그래서 tmux의 핵심 가치인 양방향 attach가 이 용도에선 무의미하다. 남는 건 백그라운드 유지·상태·로그뿐이고 셋 다 Go 표준 라이브러리로 된다 → tmux 같은 외부 런타임 없이 **바이너리 하나 복사로 배포.** 이건 tmux를 뺀 것이지 의존성을 금하는 게 아니다 — 값어치 있는 Go 패키지는 쓴다(TUI는 bubbletea/lipgloss). (이 판단의 전체 과정은 docs/ARCHITECTURE.md.)

## 설치·사용

구현 후 작성.

## 라이선스

미정 (공개 시 결정).
