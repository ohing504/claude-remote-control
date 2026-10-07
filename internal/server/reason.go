package server

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"crc/internal/state"
)

// exitReasonTailBytes는 종료 원인을 찾으려 server.err 끝에서 읽는 최대 크기다.
// server.err는 시작 때만 비우고 실행 중에는 크기 상한이 없어, 오래 켜둔 서버가
// stderr를 많이 남긴 뒤 죽으면 커질 수 있다. TUI는 이 함수를 주기적으로 부른다.
const exitReasonTailBytes = 4 << 10

// ExitReason은 서버 stderr(server.err)의 공백이 아닌 마지막 줄을 돌려준다. dead 서버가
// 왜 기동 직후 종료했는지(trust 미수락, 429 등) 보여주는 용도다. 기록이 없으면 "".
// 터미널 표시를 깨뜨리는 ANSI 시퀀스는 지우고, 캐리지 리턴으로 덮어쓴 줄은 마지막
// 덮어쓴 내용만 남긴다.
func ExitReason(dir, name string) string {
	f, err := os.Open(state.ErrPath(dir, name)) //nolint:gosec // 내부에서 구성한 상태 경로
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	if fi, err := f.Stat(); err == nil && fi.Size() > exitReasonTailBytes {
		if _, err := f.Seek(-exitReasonTailBytes, io.SeekEnd); err != nil {
			return ""
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(ansi.Strip(string(data)))
	s = s[strings.LastIndexByte(s, '\n')+1:]
	s = s[strings.LastIndexByte(s, '\r')+1:]
	return strings.TrimSpace(s)
}
