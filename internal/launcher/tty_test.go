//go:build !windows

package launcher

import (
	"os"
	"syscall"
	"testing"
)

// skipIfNoTTY は /dev/tty が利用不可な環境でテストをスキップする
func skipIfNoTTY(t *testing.T) {
	t.Helper()
	fd, err := syscall.Open("/dev/tty", syscall.O_RDWR, 0)
	if err != nil {
		t.Skip("/dev/tty is not available in this environment")
	}
	syscall.Close(fd)
}

func TestReopenTTY_ReturnsNilWhenTTYAvailable(t *testing.T) {
	skipIfNoTTY(t)

	// 元の fd 0/1/2 を退避して復元する
	savedFds := saveFds(t)
	defer restoreFds(t, savedFds)

	err := reopenTTY()
	if err != nil {
		t.Fatalf("reopenTTY() returned error: %v", err)
	}
}

func TestReopenTTY_FdsAreBlockingAfterCall(t *testing.T) {
	// reopenTTY() 後の fd 0/1/2 に O_NONBLOCK が付いていないことを検証する
	skipIfNoTTY(t)

	savedFds := saveFds(t)
	defer restoreFds(t, savedFds)

	if err := reopenTTY(); err != nil {
		t.Fatalf("reopenTTY() returned error: %v", err)
	}

	for _, fd := range []int{0, 1, 2} {
		flags, err := fcntlGetFl(fd)
		if err != nil {
			t.Fatalf("fcntl(F_GETFL) on fd %d: %v", fd, err)
		}
		if flags&syscall.O_NONBLOCK != 0 {
			t.Errorf("fd %d has O_NONBLOCK set after reopenTTY()", fd)
		}
	}
}

func TestReopenTTY_FdsAreWritable(t *testing.T) {
	// reopenTTY() 後の fd 1/2 に書き込みできることを検証する
	skipIfNoTTY(t)

	savedFds := saveFds(t)
	defer restoreFds(t, savedFds)

	if err := reopenTTY(); err != nil {
		t.Fatalf("reopenTTY() returned error: %v", err)
	}

	for _, fd := range []int{1, 2} {
		// fd が有効かつ書き込み可能であることを確認
		_, err := syscall.Write(fd, []byte{})
		if err != nil {
			t.Errorf("fd %d is not writable after reopenTTY(): %v", fd, err)
		}
	}
}

func TestReopenTTY_OriginalFdIsClosed(t *testing.T) {
	// reopenTTY() が syscall.Open で取得した fd を閉じていることを検証する
	// fd 3 以降に TTY の fd が残っていないことを確認
	skipIfNoTTY(t)

	savedFds := saveFds(t)
	defer restoreFds(t, savedFds)

	// reopenTTY 前に開いている fd の上限を記録
	beforeFd := nextAvailableFd(t)

	if err := reopenTTY(); err != nil {
		t.Fatalf("reopenTTY() returned error: %v", err)
	}

	// reopenTTY 後、同じ fd 番号が再利用可能なら元 fd は閉じられている
	afterFd := nextAvailableFd(t)
	if afterFd > beforeFd {
		t.Errorf("reopenTTY() leaked a fd: before=%d, after=%d", beforeFd, afterFd)
	}
}

// saveFds は fd 0/1/2 を退避して、テスト後に復元できるようにする
func saveFds(t *testing.T) [3]int {
	t.Helper()
	var saved [3]int
	for i := 0; i < 3; i++ {
		fd, err := syscall.Dup(i)
		if err != nil {
			t.Fatalf("saving fd %d: %v", i, err)
		}
		saved[i] = fd
	}
	return saved
}

// restoreFds は退避した fd 0/1/2 を復元する
func restoreFds(t *testing.T, saved [3]int) {
	t.Helper()
	for i := 0; i < 3; i++ {
		if err := syscall.Dup2(saved[i], i); err != nil {
			// テスト中の復元失敗は致命的
			t.Errorf("restoring fd %d: %v", i, err)
		}
		syscall.Close(saved[i])
	}
	// os.Stdin/Stdout/Stderr を再構築
	os.Stdin = os.NewFile(0, "/dev/stdin")
	os.Stdout = os.NewFile(1, "/dev/stdout")
	os.Stderr = os.NewFile(2, "/dev/stderr")
}

// fcntlGetFl は fd のファイルステータスフラグを取得する
func fcntlGetFl(fd int) (int, error) {
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(syscall.F_GETFL), 0)
	if errno != 0 {
		return 0, errno
	}
	return int(flags), nil
}

// nextAvailableFd は次に利用可能な最小の fd 番号を返す
func nextAvailableFd(t *testing.T) int {
	t.Helper()
	fd, err := syscall.Open("/dev/null", syscall.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("opening /dev/null: %v", err)
	}
	syscall.Close(fd)
	return fd
}
