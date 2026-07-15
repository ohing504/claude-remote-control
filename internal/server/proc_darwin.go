//go:build darwin

package server

import "golang.org/x/sys/unix"

// sZomb는 macOS sys/proc.h의 SZOMB — 종료했으나 부모가 아직 reap하지 않은 좀비.
const sZomb = 5

// isZombie는 pid가 좀비 상태인지 sysctl로 확인한다.
// kill(pid,0)은 좀비도 성공하므로, 좀비를 Running으로 오판하지 않으려면 이 확인이 필요하다.
// crc가 Setsid로 띄운 자식을 wait하지 않아 생기는 좀비를 dead로 판정하기 위함이다.
func isZombie(pid int) bool {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return false // 조회 실패 시 판정 보류(좀비 아님으로)
	}
	return kp.Proc.P_stat == sZomb
}
