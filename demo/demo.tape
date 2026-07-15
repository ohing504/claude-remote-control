# crc 데모 녹화 스크립트 (VHS). 재생성: demo/record.sh
# 격리: XDG_CONFIG_HOME·XDG_STATE_HOME를 demo/.state로, PATH에 demo/bin(가짜 claude)을 앞세우고
# 가짜 워크스페이스를 /tmp/crc-demo에 만들어 개인 경로 노출 없이 오프라인으로 굽는다.

Output demo/demo.gif

Set FontSize 18
Set Width 1120
Set Height 720
Set Padding 24

# --- 격리 셋업(녹화에서 숨김) ---
Hide
Type "export XDG_CONFIG_HOME=$PWD/demo/.state XDG_STATE_HOME=$PWD/demo/.state" Enter
Type "export PATH=$PWD/demo/bin:$PATH" Enter
Type "rm -rf demo/.state && mkdir -p demo/.state" Enter
Type "D=/tmp/crc-demo; rm -rf $D; mkdir -p $D/proj-a $D/proj-b $D/proj-c" Enter
Type "for p in proj-a proj-b proj-c; do echo x > $D/$p/CLAUDE.md; done" Enter
Type "cd $D" Enter
Type "crc add proj-a >/dev/null; crc add proj-b >/dev/null" Enter
Type "clear" Enter
Show

# --- 상태판 ---
Type "crc" Enter
Sleep 1.8s

# 토글: proj-a 시작(초록 ●)
Enter
Sleep 1.6s

# 다음 항목으로 이동해 proj-b도 시작
Type "j"
Sleep 500ms
Enter
Sleep 1.6s

# 로그 보기(가짜 서버 로그) → 목록 복귀
Type "l"
Sleep 2.2s
Escape
Sleep 900ms

# 추가 화면: 스캔 후보에서 proj-c 골라 담기(진입 시 포커스는 이미 리스트)
Type "a"
Sleep 1.8s
Down
Down
Sleep 700ms
Space
Sleep 1s
Enter
Sleep 1.8s

# 전체 정지 후 종료
Type "x"
Sleep 1.2s
Type "q"
Sleep 1.2s
