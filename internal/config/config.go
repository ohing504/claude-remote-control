// Package config는 등록된 워크스페이스 목록(workspaces.json)의 로드·저장을 담당한다.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// FindByPath는 경로로 워크스페이스를 찾는다. 없으면 nil.
func (c *Config) FindByPath(path string) *Workspace {
	for i := range c.Workspaces {
		if c.Workspaces[i].Path == path {
			return &c.Workspaces[i]
		}
	}
	return nil
}

// Add는 워크스페이스를 추가한다. 이름 또는 경로가 중복이면 에러.
// ValidateName은 워크스페이스 이름이 파일 경로로 쓰기에 안전한지 확인한다.
// 이름은 pid 파일명과 로그 디렉토리명이 되므로, 경로 구분자나 `..`가 들어가면
// 상태 디렉토리 밖을 가리키게 된다. 로그 디렉토리는 시작할 때 통째로 지워지므로
// 그 대상이 사용자 파일이 될 수 있다.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("이름이 비어 있습니다")
	}
	if strings.ContainsAny(name, `/\`) {
		return errors.New("이름에 경로 구분자를 쓸 수 없습니다: " + name)
	}
	if strings.HasPrefix(name, ".") {
		return errors.New("이름을 .으로 시작할 수 없습니다: " + name)
	}
	return nil
}

func (c *Config) Add(ws Workspace) error {
	if err := ValidateName(ws.Name); err != nil {
		return err
	}
	if c.Find(ws.Name) != nil {
		return errors.New("이미 등록된 이름: " + ws.Name)
	}
	if dup := c.FindByPath(ws.Path); dup != nil {
		return errors.New("이미 등록된 경로: " + ws.Path + " (name=" + dup.Name + ")")
	}
	c.Workspaces = append(c.Workspaces, ws)
	return nil
}

// UniqueName은 base가 이미 등록돼 있으면 부모 폴더명을 앞에 붙여 유니크한 이름을 만든다.
// 이름은 pid·log 파일명으로 쓰이므로 경로 구분자는 '-'로 잇는다.
// 예: budget-master가 이미 있으면 references-budget-master.
func (c *Config) UniqueName(path, base string) string {
	if c.Find(base) == nil {
		return base
	}
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	name := base
	// path의 마지막(=base) 바로 위 부모부터 하나씩 앞에 붙인다.
	for i := len(parts) - 2; i >= 0; i-- {
		if parts[i] == "" {
			continue
		}
		name = parts[i] + "-" + name
		if c.Find(name) == nil {
			return name
		}
	}
	// 조상까지 다 붙여도 겹치면 숫자 접미사.
	for n := 2; ; n++ {
		cand := base + "-" + strconv.Itoa(n)
		if c.Find(cand) == nil {
			return cand
		}
	}
}

// Remove는 이름으로 워크스페이스를 제거한다. 없으면 에러.
func (c *Config) Remove(name string) error {
	for i := range c.Workspaces {
		if c.Workspaces[i].Name == name {
			c.Workspaces = append(c.Workspaces[:i], c.Workspaces[i+1:]...)
			return nil
		}
	}
	return errors.New("등록되지 않은 이름: " + name)
}

// ValidatePath는 경로를 절대경로로 정규화하고, 실제 존재하는 디렉토리인지 검증한다.
func ValidatePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return "", errors.New("경로가 존재하지 않음: " + abs)
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("디렉토리가 아님: " + abs)
	}
	return abs, nil
}
