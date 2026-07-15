package tui

import "testing"

func TestWindowBounds(t *testing.T) {
	tests := []struct {
		name                   string
		cursor, total, visible int
		wantStart, wantEnd     int
	}{
		{"전부 보임", 0, 3, 10, 0, 3},
		{"딱 맞음", 2, 5, 5, 0, 5},
		{"위쪽 커서 — 0에서 시작", 1, 100, 10, 0, 10},
		{"가운데 커서 — 중앙 정렬", 50, 100, 10, 45, 55},
		{"아래쪽 커서 — 끝에 고정", 99, 100, 10, 90, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := windowBounds(tt.cursor, tt.total, tt.visible)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("windowBounds(%d,%d,%d)=(%d,%d), (%d,%d) 기대",
					tt.cursor, tt.total, tt.visible, start, end, tt.wantStart, tt.wantEnd)
			}
			// 불변식: 커서는 항상 창 안.
			if tt.total > 0 && (tt.cursor < start || tt.cursor >= end) {
				t.Fatalf("커서 %d가 창 [%d,%d) 밖", tt.cursor, start, end)
			}
		})
	}
}

func TestVisibleCount(t *testing.T) {
	if got := visibleCount(0, 5); got != 19 { // 미측정 시 기본 24 - 5
		t.Fatalf("height 0 → %d, 19 기대", got)
	}
	if got := visibleCount(24, 5); got != 19 {
		t.Fatalf("height 24 → %d, 19 기대", got)
	}
	if got := visibleCount(6, 5); got != 3 { // 하한 3
		t.Fatalf("작은 높이 → %d, 3 기대(하한)", got)
	}
}
