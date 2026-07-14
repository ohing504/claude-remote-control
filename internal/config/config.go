// Package config는 등록된 워크스페이스 목록(workspaces.json)의 로드·저장을 담당한다.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Workspace는 등록된 remote-control 서버 하나(= 디렉토리 하나)를 가리킨다.
type Workspace struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Config는 workspaces.json 전체 내용이다.
type Config struct {
	Workspaces []Workspace `json:"workspaces"`
}

// Dir은 crc 설정 디렉토리(~/.config/crc, XDG_CONFIG_HOME 존중)를 반환한다.
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "crc"), nil
}

// path는 workspaces.json의 전체 경로를 반환한다.
func path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "workspaces.json"), nil
}

// Load는 저장된 설정을 읽는다. 파일이 없으면 빈 Config를 돌려준다(최초 실행).
func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p) //nolint:gosec // p는 내부에서 구성한 고정 config 경로
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save는 설정을 atomic하게 쓴다(temp 파일 → rename). 중간 크래시에 파일이 깨지지 않게.
func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	p := filepath.Join(dir, "workspaces.json")
	tmp, err := os.CreateTemp(dir, "workspaces-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // rename 성공 시 no-op

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, p)
}

// Find는 이름으로 워크스페이스를 찾는다. 없으면 nil.
func (c *Config) Find(name string) *Workspace {
	for i := range c.Workspaces {
		if c.Workspaces[i].Name == name {
			return &c.Workspaces[i]
		}
	}
	return nil
}

// Add는 워크스페이스를 추가한다. 이름 중복이면 에러.
func (c *Config) Add(ws Workspace) error {
	if c.Find(ws.Name) != nil {
		return errors.New("이미 등록된 이름: " + ws.Name)
	}
	c.Workspaces = append(c.Workspaces, ws)
	return nil
}
