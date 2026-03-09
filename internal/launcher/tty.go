//go:build !windows

package launcher

import (
	"fmt"
	"syscall"
)

// reopenTTY は /dev/tty を直接開いて fd 0/1/2 に複製する。
// syscall.Open を使い、Go ランタイムのネットポーラー登録をバイパスする。
// os.OpenFile を使うと O_NONBLOCK が設定され、syscall.Exec 後の子プロセスに引き継がれてしまう。
func reopenTTY() error {
	fd, err := syscall.Open("/dev/tty", syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("opening /dev/tty: %w", err)
	}
	for _, targetFd := range []int{0, 1, 2} {
		if err := syscall.Dup2(fd, targetFd); err != nil {
			syscall.Close(fd)
			return fmt.Errorf("dup2(%d, %d): %w", fd, targetFd, err)
		}
	}
	// fd が 0, 1, 2 自体の場合、Dup2 先と同一なので Close すると壊れる
	if fd > 2 {
		syscall.Close(fd)
	}
	return nil
}
