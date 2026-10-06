package server

import (
	"os"
	"strings"

	"crc/internal/state"
)

// ExitReason은 서버 stderr(server.err)의 공백이 아닌 마지막 줄을 돌려준다. dead 서버가
// 왜 기동 직후 종료했는지(trust 미수락, 429 등) 보여주는 용도다. 기록이 없으면 "".
// server.err는 시작마다 새로 쓰고 정상 기동이면 비어 있어 통째로 읽어도 작다.
func ExitReason(dir, name string) string {
	data, err := os.ReadFile(state.ErrPath(dir, name)) //nolint:gosec // 내부에서 구성한 상태 경로
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return ""
}
