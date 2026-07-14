# CLAUDE.md — claude-remote-control (crc)

## 프로젝트

여러 `claude remote-control` 서버를 한 화면에서 켜고·끄고·상태 보고·로그 보는 **Go 단일 바이너리 TUI**. 개발 착수 전 아래 문서를 먼저 읽는다.

## 문서

- **[README.md](README.md)** — 공개 얼굴: 왜 만드나·무엇을·설치·사용.
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** — 설계·아키텍처 SSOT: 목표·성공기준·설계 결정·조사 결론·생명주기·명령 표면·TUI.
- **[docs/ROADMAP.md](docs/ROADMAP.md)** — 구현 순서·진행 트래킹(M0~M6 체크박스, 마일스톤별 검증 기준).

## 상태

설계 단계, 코드 없음. `docs/ROADMAP.md`의 M0부터 순서로 구현한다.

## 개발 원칙 (ARCHITECTURE.md에서 발췌 — 어기면 안 되는 것)

- **env 주입 절대 규칙**: 서버를 띄우는 모든 경로에 `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`을 주입한다. 빠지면 spawn된 자식에서 `SendUserFile`이 안 떠 도구의 존재 이유가 사라진다. `CLAUDE_CODE_REMOTE=1`은 금지(서버 기동 거부).
- **런타임 의존성 0**: tmux 등 외부 도구에 의존하지 않는다. Go 표준 라이브러리 + bubbletea/lipgloss만.
- **M2 검증 기준**: 생명주기 구현은 "환경변수 넣었으니 되겠지"로 통과시키지 않는다. 새로 띄운 서버의 자식에서 폰으로 `SendUserFile` 파일이 실제 도착하는 것으로만 통과.

## 공개

**public repo로 공개 예정.** 문서·코드·커밋 메시지에 개인 워크스페이스명·절대경로·사적 정보를 노출하지 않는다(예시는 `proj-a` 같은 일반명으로). README는 공개 첫인상이므로 정갈하게 유지.

## 개발 진행

이 프로젝트는 새 세션에서 개발한다. 착수 시 `docs/ARCHITECTURE.md` → `docs/ROADMAP.md` M0부터. 개발기는 별도 개인 블로그(`~/workspace/portfolio`)에서 관리하므로 이 레포엔 두지 않는다.
