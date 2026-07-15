//go:build !darwin

package server

// isZombie는 darwin 외 플랫폼에선 좀비 감지를 하지 않는다(signal 0 판정만).
// 필요해지면 Linux는 /proc/<pid>/stat의 상태 필드로 구현할 수 있다.
func isZombie(int) bool { return false }
