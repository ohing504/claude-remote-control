package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain은 testscript가 "crc"라는 in-process 커맨드로 run()을 부를 수 있게 등록한다.
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"crc": func() {
			if err := run(os.Args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "crc:", err)
				os.Exit(1)
			}
		},
	})
}

// TestScripts는 testdata/script/*.txtar의 CLI 시나리오를 실행한다.
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(e *testscript.Env) error {
			// 각 스크립트를 격리 — config/state를 작업 디렉토리 밑으로.
			e.Setenv("XDG_CONFIG_HOME", e.WorkDir+"/cfg")
			e.Setenv("XDG_STATE_HOME", e.WorkDir+"/state")
			return nil
		},
	})
}
