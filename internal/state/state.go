// Package state는 실행 중 서버의 런타임 상태 파일(pid·log) 경로를 관리한다.
// 데이터는 ~/.local/state/crc/ (XDG_STATE_HOME 존중)에 둔다.
package state

import (
	"os"
	"path/filepath"
)

// Dir은 crc 상태 디렉토리(~/.local/state/crc)를 반환한다.
func Dir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "crc"), nil
}

// EnsureDir은 상태 디렉토리를 만들고 그 경로를 반환한다.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// PidPath는 <name>.pid의 전체 경로를 반환한다.
func PidPath(dir, name string) string {
	return filepath.Join(dir, name+".pid")
}

// LogPath는 <name>.log의 전체 경로를 반환한다.
func LogPath(dir, name string) string {
	return filepath.Join(dir, name+".log")
}
