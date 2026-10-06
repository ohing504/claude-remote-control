package server

import (
	"os"
	"testing"

	"crc/internal/state"
)

// server.err의 공백이 아닌 마지막 줄을 종료 원인으로 돌려주는지.
func TestExitReasonLastNonBlankLine(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(state.LogDir(dir, "a"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "warn: something\nError: Workspace not trusted.  \n\n  \n"
	if err := os.WriteFile(state.ErrPath(dir, "a"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, want := ExitReason(dir, "a"), "Error: Workspace not trusted."; got != want {
		t.Fatalf("%q, %q 기대", got, want)
	}
}

// server.err가 없거나 비어 있으면 빈 문자열.
func TestExitReasonMissingOrEmpty(t *testing.T) {
	dir := t.TempDir()
	if got := ExitReason(dir, "none"); got != "" {
		t.Fatalf("파일 없음: %q, 빈 문자열 기대", got)
	}
	if err := os.MkdirAll(state.LogDir(dir, "e"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state.ErrPath(dir, "e"), []byte("\n \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ExitReason(dir, "e"); got != "" {
		t.Fatalf("공백뿐: %q, 빈 문자열 기대", got)
	}
}
