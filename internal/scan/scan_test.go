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

// 상위가 마커를 가져도 그 아래 프로젝트를 가리지 않는다(서브트리 스킵 안 함).
func TestFindDoesNotHideNested(t *testing.T) {
	root := t.TempDir()
	mk := func(rel string) {
		dir := filepath.Join(root, rel)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mk("container") // 상위(예: ~/workspace)도 마커를 가질 수 있음
	mk("container/proj-a")
	mk("container/proj-b")

	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	got := names(cands)
	want := []string{"container", "proj-a", "proj-b"}
	if !slices.Equal(got, want) {
		t.Fatalf("후보=%v, %v 기대(하위도 보여야)", got, want)
	}
}

// 심링크가 프로젝트를 가리키면 후보에 포함한다(예: ~/Second Brain).
func TestFindFollowsSymlinkToProject(t *testing.T) {
	root := t.TempDir()
	// 실제 프로젝트는 root 밖에 두고, root 안에 심링크만 놓는다.
	realDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(realDir, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "brain")
	if err := os.Symlink(realDir, link); err != nil {
		t.Fatal(err)
	}
	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got := names(cands); !slices.Equal(got, []string{"brain"}) {
		t.Fatalf("후보=%v, [brain] 기대(심링크 프로젝트)", got)
	}
}

// NFD 파일명(자모 분해)이 후보 이름에서 NFC로 정규화되는지.
func TestFindNormalizesNFD(t *testing.T) {
	root := t.TempDir()
	// 파일명을 NFD(자모 분해)로 만들고 후보 이름이 NFC로 합쳐지는지 본다.
	nfd := "\u1100\u1161" // NFD: 초성 ㄱ + 중성 ㅏ → NFC로 U+AC00("가")
	dir := filepath.Join(root, nfd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cands, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("후보 %d개, 1개 기대", len(cands))
	}
	if cands[0].Name != "\uAC00" { // NFC "가"
		t.Fatalf("이름=%q(% x), NFC '가' 기대", cands[0].Name, cands[0].Name)
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
