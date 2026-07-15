# CLAUDE.md — claude-remote-control (crc)

## 프로젝트

여러 `claude remote-control` 서버를 한 화면에서 켜고·끄고·상태 보고·로그 보는 **Go 단일 바이너리 TUI**. 개발 착수 전 아래 문서를 먼저 읽는다.

## 문서 (경계 분리)

- **[README.md](README.md)** — 제품 SSOT: 왜·무엇·범위·비목표·설치·사용.
- **[docs/SPEC.md](docs/SPEC.md)** — 기능 명세 SSOT: 명령별 규칙·검증·출력·에러 + 구현 상태 + 하단 우선순위(구현 순서). "뭐가 됐고 뭘 해야 하나"는 여기서 본다.
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** — 순수 설계: 설계 결정·조사 결론·데이터 배치·생명주기·스캔 알고리즘.

경계: 제품 사실→README / 기능 명세·상태→SPEC / 설계·구현 방식→ARCHITECTURE. 재서술 없이 서로 참조.

## 상태

초기 구현 중. 현재 상태·다음 작업은 `docs/SPEC.md`(명령 표의 상태 + 하단 우선순위)가 SSOT.

## 개발 원칙 (ARCHITECTURE.md에서 발췌 — 어기면 안 되는 것)

- **env 주입 절대 규칙**: 서버를 띄우는 모든 경로에 `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=1`을 주입한다. 빠지면 spawn된 자식에서 `SendUserFile`이 안 떠 도구의 존재 이유가 사라진다. `CLAUDE_CODE_REMOTE=1`은 금지(서버 기동 거부).
- **외부 런타임 도구 비의존**: tmux처럼 별도 설치가 필요한 외부 런타임에 기대지 않는다. 이건 *tmux가 이 용도(폰·웹 조작 데몬)에 안 맞아서 뺀 것*이지 "의존성을 0으로" 만드는 규칙이 아니다. **값어치 있는 Go 패키지는 쓴다**(TUI에 bubbletea/lipgloss, 필요하면 메뉴바에 systray 등). 지향점은 "설치가 번거로운 외부 런타임 없이 바이너리로 배포".
- **M2 검증 기준**: 생명주기 구현은 "환경변수 넣었으니 되겠지"로 통과시키지 않는다. 새로 띄운 서버의 자식에서 폰으로 `SendUserFile` 파일이 실제 도착하는 것으로만 통과.

## 공개

**public repo로 공개 예정.** 문서·코드·커밋 메시지에 개인 워크스페이스명·절대경로·사적 정보를 노출하지 않는다(예시는 `proj-a` 같은 일반명으로). README는 공개 첫인상이므로 정갈하게 유지.

## 개발 진행

착수 시 `docs/SPEC.md`의 하단 우선순위에서 다음 마일스톤을 확인하고, 설계가 필요하면 `docs/ARCHITECTURE.md`를 참조한다. 개발기는 별도 개인 블로그(`~/workspace/portfolio`)에서 관리하므로 이 레포엔 두지 않는다.
