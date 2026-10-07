package server

import (
	"os"
	"testing"

	"crc/internal/state"
)

// server.err의 공백이 아닌 마지막 줄을 종료 원인으로 돌려주고, 기록이 없으면 ""인지.
func TestExitReason(t *testing.T) {
	cases := []struct {
		name string
		body *string // nil이면 파일 없음
		want string
	}{
		{"마지막 줄", ptr("warn: something\nError: Workspace not trusted.  \n\n  \n"), "Error: Workspace not trusted."},
		{"한 줄", ptr("Error: x"), "Error: x"},
		{"공백뿐", ptr("\n \n"), ""},
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

func ptr(s string) *string { return &s }
