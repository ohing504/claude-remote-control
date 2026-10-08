package server

import (
	"os"
	"strings"
	"testing"

	"crc/internal/state"
)

// server.err에서 "Error:"로 시작하는 마지막 줄을, 없으면 공백이 아닌 마지막 줄을 종료 원인으로
// 돌려주고, 기록이 없으면 ""인지.
func TestExitReason(t *testing.T) {
	cases := []struct {
		name string
		body *string // nil이면 파일 없음
		want string
	}{
		{"마지막 줄", new("warn: something\nError: Workspace not trusted.  \n\n  \n"), "Error: Workspace not trusted."},
		{"한 줄", new("Error: x"), "Error: x"},
		{"Error 줄 뒤 안내 줄", new("Error: already served. Stop it first.\nExiting in about 57 seconds.\n"), "Error: already served. Stop it first."},
		{"Error 줄 여럿", new("Error: a\nwarn: b\nError: c\nExiting soon.\n"), "Error: c"},
		{"Error 줄 없음", new("warn: a\nconnection lost\n"), "connection lost"},
		{"덮어쓴 줄 뒤 Error", new("Connecting...\rError: x\nExiting soon.\n"), "Error: x"},
		{"공백뿐", new("\n \n"), ""},
		{"캐리지 리턴으로 덮어쓴 줄", new("Connecting...\rError: x\n"), "Error: x"},
		{"ANSI 색 코드", new("\x1b[31mError: x\x1b[0m\n"), "Error: x"},
		{"앞부분이 큰 파일", new(strings.Repeat("warn: noise\n", 10000) + "Error: x\n"), "Error: x"},
		{"파일 없음", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if c.body != nil {
				if err := os.MkdirAll(state.LogDir(dir, "a"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(state.ErrPath(dir, "a"), []byte(*c.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := ExitReason(dir, "a"); got != c.want {
				t.Fatalf("%q, %q 기대", got, c.want)
			}
		})
	}
}
