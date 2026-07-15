package scan

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// 트리를 구성한다: 마커 종류·스킵 대상·깊이 초과를 한 번에 검증.
func setupTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkProject := func(rel, marker string) {
		dir := filepath.Join(root, rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		switch marker {
		case "CLAUDE.md":
			if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		case ".claude":
			if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}

	mkProject("proj-md", "CLAUDE.md") // CLAUDE.md 후보
	mkProject("proj-dir", ".claude")  // .claude 디렉토리 후보
	mkProject("plain", "")            // 마커 없음 → 제외
	// .git 내부의 마커는 스킵돼야
	mkProject(".git/inner", "CLAUDE.md")
	// node_modules 내부도 스킵
	mkProject("node_modules/pkg", "CLAUDE.md")
	return root
}

func names(cands []Candidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.Name
	}
	slices.Sort(out)
	return out
}

func TestFindMarkersAndSkips(t *testing.T) {
	root := setupTree(t)
	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	got := names(cands)
	want := []string{"proj-dir", "proj-md"}
	if !slices.Equal(got, want) {
		t.Fatalf("후보=%v, %v 기대(.git·node_modules는 스킵)", got, want)
	}

	// 마커 근거도 확인.
	for _, c := range cands {
		if c.Name == "proj-md" && c.Marker != "CLAUDE.md" {
			t.Fatalf("proj-md marker=%q, CLAUDE.md 기대", c.Marker)
		}
		if c.Name == "proj-dir" && c.Marker != ".claude" {
			t.Fatalf("proj-dir marker=%q, .claude 기대", c.Marker)
		}
	}
}

func TestFindDepthLimit(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// 깊이 5의 마커는 상한 3에서 안 잡혀야.
	cands, err := Find(root, 3)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("깊이 상한 초과인데 후보 %d개 잡힘", len(cands))
	}
}

// 프로젝트 안에 중첩된 프로젝트(서브패키지·third-parties)는 최상위만 후보.
func TestFindSkipsNestedProjects(t *testing.T) {
	root := t.TempDir()
	// top이 프로젝트이고 그 아래 apps/sub, third-parties/lib도 마커를 가진다.
	mk := func(rel string) {
		dir := filepath.Join(root, rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mk("top")
	mk("top/apps/sub")
	mk("top/third-parties/lib")

	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got := names(cands); !slices.Equal(got, []string{"top"}) {
		t.Fatalf("후보=%v, [top]만 기대(중첩은 스킵)", got)
	}
}

// root 자신이 마커를 가져도(예: ~/workspace/.claude) 하위 프로젝트 탐색은 계속돼야.
func TestFindRootMarkerStillScansChildren(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "proj-a")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	// root와 proj-a 둘 다 후보(root 마커가 하위를 막지 않음).
	if len(cands) != 2 {
		t.Fatalf("후보 %d개, 2개 기대(root+proj-a): %v", len(cands), names(cands))
	}
	if !slices.Contains(names(cands), "proj-a") {
		t.Fatalf("하위 proj-a가 후보에 없음: %v", names(cands))
	}
}

func TestFindRootItself(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(cands) != 1 || cands[0].Path != filepath.Clean(root) {
		t.Fatalf("root 자신이 후보로 안 잡힘: %+v", cands)
	}
}
