// Package scan은 root 아래에서 CLAUDE.md/.claude를 가진 디렉토리를 등록 후보로 찾는다.
package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
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
		// 심링크가 프로젝트 디렉토리를 가리키면 후보에 넣되, 그 안으로는 진입하지 않는다
		// (WalkDir는 심링크를 따라가지 않고, 따라가면 순환 위험이 있다). 예: ~/notes.
		if d.Type()&fs.ModeSymlink != 0 {
			if fi, e := os.Stat(path); e == nil && fi.IsDir() {
				if marker := ProjectMarker(path); marker != "" {
					out = append(out, newCandidate(path, marker))
				}
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root {
			name := d.Name()
			// node_modules와 모든 숨김 디렉토리(.git·.cache·.Trash 등)는 진입하지 않는다.
			// .claude 마커는 부모에서 Stat으로 감지하므로 그 안으로 들어갈 필요가 없다.
			if name == "node_modules" || strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
		}
		if depthFrom(root, path) > maxDepth {
			return fs.SkipDir
		}
		if marker := ProjectMarker(path); marker != "" {
			out = append(out, newCandidate(path, marker))
			// 서브트리를 스킵하지 않는다 — 상위가 마커를 가져도(예: ~/workspace/.claude)
			// 그 아래 프로젝트를 가리지 않게. 중첩 노이즈는 깊이 상한으로만 제한한다.
		}
		return nil
	})
	return out, err
}

// newCandidate는 후보를 만들며 이름을 NFC로 정규화한다.
// macOS 파일명은 NFD(자모 분해)라 그대로 두면 한글이 깨져 보이고 이름 매칭도 어긋난다.
func newCandidate(path, marker string) Candidate {
	return Candidate{Path: path, Name: norm.NFC.String(filepath.Base(path)), Marker: marker}
}

// ProjectMarker는 디렉토리가 프로젝트임을 나타내는 표식을 반환한다(없으면 "").
func ProjectMarker(dir string) string {
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
