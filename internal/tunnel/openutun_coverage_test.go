//go:build darwin

package tunnel

import (
	"strings"
	"syscall"
	"testing"
)

// TestOpenUtun_SocketErrorViaRlimit exercises the unix.Socket error path in
// openUtun by temporarily lowering the process file-descriptor limit so that
// the AF_SYSTEM socket allocation fails with EMFILE.
func TestOpenUtun_SocketErrorViaRlimit(t *testing.T) {
	var oldLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &oldLimit); err != nil {
		t.Skipf("getrlimit: %v", err)
	}

	newLimit := syscall.Rlimit{
		Cur: 3, // stdin, stdout, stderr only
		Max: oldLimit.Max,
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &newLimit); err != nil {
		t.Skipf("setrlimit: %v", err)
	}
	defer syscall.Setrlimit(syscall.RLIMIT_NOFILE, &oldLimit)

	fd, name, err := openUtun(-1)
	if err == nil {
		t.Fatalf("expected error from openUtun with exhausted fds, got fd=%d name=%q", fd, name)
	}
	if fd != -1 {
		t.Errorf("expected fd=-1, got %d", fd)
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
	if !strings.Contains(err.Error(), "socket AF_SYSTEM") {
		t.Errorf("error should mention 'socket AF_SYSTEM': %v", err)
	}
}

// TestCreateTUN_SocketErrorViaRlimit exercises CreateTUN when the underlying
// openUtun call fails due to socket allocation failure (EMFILE).
func TestCreateTUN_SocketErrorViaRlimit(t *testing.T) {
	var oldLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &oldLimit); err != nil {
		t.Skipf("getrlimit: %v", err)
	}

	newLimit := syscall.Rlimit{
		Cur: 3,
		Max: oldLimit.Max,
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &newLimit); err != nil {
		t.Skipf("setrlimit: %v", err)
	}
	defer syscall.Setrlimit(syscall.RLIMIT_NOFILE, &oldLimit)

	dev, err := CreateTUN("")
	if err == nil {
		dev.Close()
		t.Fatal("expected error from CreateTUN with exhausted fds")
	}
}
