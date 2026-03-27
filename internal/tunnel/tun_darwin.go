//go:build darwin

package tunnel

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"unsafe"

	"golang.org/x/sys/unix"
)

// macOS utun control name for SYSPROTO_CONTROL.
const utunControlName = "com.apple.net.utun_control"

// macOS-specific socket constants not in golang.org/x/sys/unix.
const (
	sysProtoControl = 2  // SYSPROTO_CONTROL
	afSystem        = 32 // AF_SYSTEM
)

// linuxTUN on macOS: utun device via SYSPROTO_CONTROL.
type darwinTUN struct {
	file *os.File
	name string
	mtu  int
}

// CreateTUN opens (or creates) a utunN device on macOS.
// If name is empty or "utun", the OS assigns the next available unit.
// Specify "utun3" to request a specific unit.
func CreateTUN(name string) (Device, error) {
	unit := -1 // -1 = let OS choose
	if name != "" && name != "utun" {
		var n int
		if _, err := fmt.Sscanf(name, "utun%d", &n); err == nil {
			unit = n
		}
	}

	fd, devName, err := openUtun(unit)
	if err != nil {
		return nil, err
	}

	// Set non-blocking
	if err := unix.SetNonblock(fd, false); err != nil {
		unix.Close(fd)
		return nil, err
	}

	f := os.NewFile(uintptr(fd), devName)
	return &darwinTUN{file: f, name: devName, mtu: 1420}, nil
}

// openUtun opens a utun device. unit == -1 means auto.
func openUtun(unit int) (int, string, error) {
	// Open a SYSPROTO_CONTROL socket.
	fd, err := unix.Socket(afSystem, unix.SOCK_DGRAM, sysProtoControl)
	if err != nil {
		return -1, "", fmt.Errorf("socket AF_SYSTEM: %w", err)
	}

	// Look up the control ID for utun.
	ctlInfo := struct {
		ctlID   uint32
		ctlName [96]byte
	}{}
	copy(ctlInfo.ctlName[:], utunControlName)

	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		// CTLIOCGINFO = 0xc0644e03
		0xc0644e03,
		uintptr(unsafe.Pointer(&ctlInfo)),
	); errno != 0 {
		unix.Close(fd)
		return -1, "", fmt.Errorf("CTLIOCGINFO: %w", errno)
	}

	// Connect to the control.
	addr := unix.SockaddrCtl{
		ID:   ctlInfo.ctlID,
		Unit: uint32(unit + 1), // unit 0 → utunN where N = Unit-1; 0 means auto
	}
	if unit < 0 {
		addr.Unit = 0
	}

	if err := unix.Connect(fd, &addr); err != nil {
		unix.Close(fd)
		return -1, "", fmt.Errorf("connect utun: %w", err)
	}

	// Find the assigned interface name.
	var ifName [unix.IFNAMSIZ]byte
	ifNameLen := uint32(len(ifName))
	if _, _, errno := unix.Syscall6(
		unix.SYS_GETSOCKOPT,
		uintptr(fd),
		// SYSPROTO_CONTROL
		sysProtoControl,
		// UTUN_OPT_IFNAME = 2
		2,
		uintptr(unsafe.Pointer(&ifName[0])),
		uintptr(unsafe.Pointer(&ifNameLen)),
		0,
	); errno != 0 {
		unix.Close(fd)
		return -1, "", fmt.Errorf("UTUN_OPT_IFNAME: %w", errno)
	}

	name := unix.ByteSliceToString(ifName[:])
	return fd, name, nil
}

func (t *darwinTUN) Name() string { return t.name }
func (t *darwinTUN) MTU() int     { return t.mtu }
func (t *darwinTUN) Close() error { return t.file.Close() }

// macOS utun prepends a 4-byte AF header to every packet.
// Read strips it; Write prepends it.

func (t *darwinTUN) Read(buf []byte) (int, error) {
	// Allocate extra space for the 4-byte AF header.
	tmp := make([]byte, len(buf)+4)
	n, err := t.file.Read(tmp)
	if err != nil {
		return 0, err
	}
	if n < 4 {
		return 0, fmt.Errorf("utun read: short read %d", n)
	}
	n = copy(buf, tmp[4:n])
	return n, nil
}

func (t *darwinTUN) Write(buf []byte) (int, error) {
	// Prepend 4-byte AF header.
	var af [4]byte
	if len(buf) > 0 {
		version := buf[0] >> 4
		var afNum uint32
		if version == 6 {
			afNum = unix.AF_INET6
		} else {
			afNum = unix.AF_INET
		}
		binary.BigEndian.PutUint32(af[:], afNum)
	}
	pkt := make([]byte, 4+len(buf))
	copy(pkt, af[:])
	copy(pkt[4:], buf)
	_, err := t.file.Write(pkt)
	return len(buf), err
}

func (t *darwinTUN) SetMTU(mtu int) error {
	cmd := exec.Command("ifconfig", t.name, "mtu", strconv.Itoa(mtu))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ifconfig mtu: %w: %s", err, out)
	}
	t.mtu = mtu
	return nil
}

func (t *darwinTUN) SetAddr(ip net.IP, prefixLen int) error {
	mask := net.CIDRMask(prefixLen, 32)
	if ip.To4() == nil {
		mask = net.CIDRMask(prefixLen, 128)
	}
	var cmd *exec.Cmd
	if ip.To4() != nil {
		// IPv4: ifconfig utunN inet <ip> <ip> netmask <mask>
		// On macOS utun, the peer address is the same as the local address for point-to-point.
		cmd = exec.Command("ifconfig", t.name, "inet",
			ip.String(), ip.String(),
			"netmask", fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3]))
	} else {
		cmd = exec.Command("ifconfig", t.name, "inet6", fmt.Sprintf("%s/%d", ip.String(), prefixLen))
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ifconfig addr: %w: %s", err, out)
	}
	return nil
}

func (t *darwinTUN) AddRoute(dst *net.IPNet) error {
	var cmd *exec.Cmd
	if dst.IP.To4() != nil {
		cmd = exec.Command("route", "-q", "-n", "add", "-inet", dst.String(), "-interface", t.name)
	} else {
		cmd = exec.Command("route", "-q", "-n", "add", "-inet6", dst.String(), "-interface", t.name)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("route add: %w: %s", err, out)
	}
	return nil
}
