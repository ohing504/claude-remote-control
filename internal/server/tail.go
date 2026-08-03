package server

import (
	"io"
	"os"
	"time"
)

// tailPollInterval은 EOF 도달 후 새 로그를 다시 확인하기까지의 간격이다.
// fsnotify 같은 런타임 의존성을 피하려 폴링으로 follow한다(의존성 0 원칙).
const tailPollInterval = 200 * time.Millisecond

// TailBytes는 추적을 시작할 때 거슬러 올라가 읽는 최대 크기다. 진단 로그는 재시작
// 사이에 이어 쌓여 수 MB가 되므로, 처음부터 다 쏟으면 화면이 과거 기록으로 덮인다.
const TailBytes = 64 << 10

// Tail은 logPath의 마지막 TailBytes를 출력하고, 이후 새로 쓰이는 내용을 계속
// 따라간다(tail -f). stop이 닫히면 반환한다. 로그 파일이 없으면 즉시 에러.
func Tail(logPath string, w io.Writer, stop <-chan struct{}) error {
	f, err := os.Open(logPath) //nolint:gosec // logPath는 내부에서 구성한 상태 경로
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if fi, err := f.Stat(); err == nil && fi.Size() > TailBytes {
		if _, err := f.Seek(-TailBytes, io.SeekEnd); err != nil {
			return err
		}
	}

	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		switch {
		case err == io.EOF:
			// 더 읽을 게 없으면 잠깐 쉬며 새 내용을 기다린다.
			select {
			case <-stop:
				return nil
			case <-time.After(tailPollInterval):
			}
		case err != nil:
			return err
		}
	}
}
