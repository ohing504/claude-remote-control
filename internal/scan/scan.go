// Package scan은 root 아래에서 CLAUDE.md/.claude를 가진 디렉토리를 등록 후보로 찾는다.
package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DefaultMaxDepth는 root 기준 탐색 깊이 상한이다. 워크스페이스 루트는 보통 얕게 놓인다.
const DefaultMaxDepth = 4

// Candidate는 등록 후보 디렉토리다.
type Candidate struct {
	Path   string // 절대경로
	Name   string // basename (등록 시 기본 이름)
	Marker string // 발견 근거: "CLAUDE.md" 또는 ".claude"
}

// Find는 root 아래를 훑어 프로젝트 마커를 가진 디렉토리를 후보로 모은다.
// .git·node_modules는 통째로 건너뛰고, maxDepth를 넘는 하위는 탐색하지 않는다.
func Find(root string, maxDepth int) ([]Candidate, error) {
	root = filepath.Clean(root)
	var out []Candidate
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 권한 없는 하위 등은 조용히 건너뜀
		}
		if !d.IsDir() {
			return nil
		}
		if path != root {
			switch d.Name() {
			case ".git", "node_modules":
				return fs.SkipDir
			}
		}
		if depthFrom(root, path) > maxDepth {
			return fs.SkipDir
		}
		if marker := projectMarker(path); marker != "" {
			out = append(out, Candidate{Path: path, Name: filepath.Base(path), Marker: marker})
			// 하위에서 프로젝트를 찾으면 그 서브트리는 더 안 본다 — 최상위 프로젝트만
			// 후보로 남겨 서브패키지·third-parties의 중첩 CLAUDE.md를 거른다.
			// root 자신이 마커를 가져도(예: ~/workspace/.claude) 하위 탐색은 계속한다.
			if path != root {
				return fs.SkipDir
			}
		}
		return nil
	})
	return out, err
}

// projectMarker는 디렉토리가 프로젝트임을 나타내는 표식을 반환한다(없으면 "").
func projectMarker(dir string) string {
	if fi, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil && !fi.IsDir() {
		return "CLAUDE.md"
	}
	if fi, err := os.Stat(filepath.Join(dir, ".claude")); err == nil && fi.IsDir() {
		return ".claude"
	}
	return ""
}

// depthFrom은 root로부터 path까지의 디렉토리 깊이다(root 자신은 0).
func depthFrom(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
