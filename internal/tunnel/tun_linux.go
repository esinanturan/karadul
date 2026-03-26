//go:build linux

package tunnel

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	cloneDevTUN = "/dev/net/tun"
	ifnameSize  = 16
)

// linuxTUN implements Device on Linux via /dev/net/tun.
type linuxTUN struct {
	file *os.File
	name string
	mtu  int
}

// CreateTUN opens (or creates) a TUN device named name.
// If name is empty, the OS chooses (e.g. "tun0").
func CreateTUN(name string) (Device, error) {
	fd, err := unix.Open(cloneDevTUN, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", cloneDevTUN, err)
	}

	// struct ifreq layout for TUNSETIFF:
	// char ifr_name[16]; short ifr_flags; (padding to 40 bytes total)
	var ifr [40]byte
	if name != "" {
		copy(ifr[:ifnameSize], name)
	}
	// IFF_TUN | IFF_NO_PI
	flags := uint16(unix.IFF_TUN | unix.IFF_NO_PI)
	binary.LittleEndian.PutUint16(ifr[16:], flags)

	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		unix.TUNSETIFF,
		uintptr(unsafe.Pointer(&ifr[0])),
	); errno != 0 {
		unix.Close(fd)
		return nil, fmt.Errorf("ioctl TUNSETIFF: %w", errno)
	}

	// Read back the interface name (may differ if we asked for "").
	devName := unix.ByteSliceToString(ifr[:ifnameSize])

	// Bring the interface up via a SOCK_DGRAM ioctl.
	if err := bringUp(devName); err != nil {
		unix.Close(fd)
		return nil, err
	}

	f := os.NewFile(uintptr(fd), cloneDevTUN)
	return &linuxTUN{file: f, name: devName, mtu: 1420}, nil
}

func (t *linuxTUN) Name() string { return t.name }
func (t *linuxTUN) MTU() int     { return t.mtu }

func (t *linuxTUN) Read(buf []byte) (int, error)  { return t.file.Read(buf) }
func (t *linuxTUN) Write(buf []byte) (int, error) { return t.file.Write(buf) }
func (t *linuxTUN) Close() error                  { return t.file.Close() }

func (t *linuxTUN) SetMTU(mtu int) error {
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(sock)

	var ifr [40]byte
	copy(ifr[:ifnameSize], t.name)
	binary.LittleEndian.PutUint32(ifr[16:], uint32(mtu))

	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(sock),
		unix.SIOCSIFMTU,
		uintptr(unsafe.Pointer(&ifr[0])),
	); errno != 0 {
		return fmt.Errorf("SIOCSIFMTU: %w", errno)
	}
	t.mtu = mtu
	return nil
}

func (t *linuxTUN) SetAddr(ip net.IP, prefixLen int) error {
	ip4 := ip.To4()
	if ip4 == nil {
		return t.setAddr6(ip, prefixLen)
	}
	return t.setAddr4(ip4, prefixLen)
}

func (t *linuxTUN) setAddr4(ip net.IP, prefixLen int) error {
	// Use netlink RTM_NEWADDR to assign the address.
	return netlinkAddrAdd(t.name, ip, prefixLen, unix.AF_INET)
}

func (t *linuxTUN) setAddr6(ip net.IP, prefixLen int) error {
	return netlinkAddrAdd(t.name, ip, prefixLen, unix.AF_INET6)
}

func (t *linuxTUN) AddRoute(dst *net.IPNet) error {
	return netlinkRouteAdd(t.name, dst)
}

// bringUp sets the IFF_UP flag on the interface.
func bringUp(name string) error {
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(sock)

	// SIOCGIFFLAGS
	var ifr [40]byte
	copy(ifr[:ifnameSize], name)
	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(sock),
		unix.SIOCGIFFLAGS,
		uintptr(unsafe.Pointer(&ifr[0])),
	); errno != 0 {
		return fmt.Errorf("SIOCGIFFLAGS: %w", errno)
	}

	// Set IFF_UP
	flags := binary.LittleEndian.Uint16(ifr[16:])
	flags |= unix.IFF_UP
	binary.LittleEndian.PutUint16(ifr[16:], flags)

	// SIOCSIFFLAGS
	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(sock),
		unix.SIOCSIFFLAGS,
		uintptr(unsafe.Pointer(&ifr[0])),
	); errno != 0 {
		return fmt.Errorf("SIOCSIFFLAGS: %w", errno)
	}
	return nil
}

// netlinkAddrAdd assigns an IP address to a named interface using netlink.
func netlinkAddrAdd(name string, ip net.IP, prefixLen int, family int) error {
	// Resolve interface index.
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", name, err)
	}

	sock, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("netlink socket: %w", err)
	}
	defer unix.Close(sock)

	// Build RTM_NEWADDR message.
	msg := buildNlAddrMsg(iface.Index, ip, prefixLen, family)
	if err := unix.Send(sock, msg, 0); err != nil {
		return fmt.Errorf("netlink send RTM_NEWADDR: %w", err)
	}
	return recvNlAck(sock)
}

// netlinkRouteAdd adds a route via a named interface.
func netlinkRouteAdd(name string, dst *net.IPNet) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", name, err)
	}

	sock, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("netlink socket: %w", err)
	}
	defer unix.Close(sock)

	msg := buildNlRouteMsg(iface.Index, dst)
	if err := unix.Send(sock, msg, 0); err != nil {
		return fmt.Errorf("netlink send RTM_NEWROUTE: %w", err)
	}
	return recvNlAck(sock)
}

// buildNlAddrMsg constructs a RTM_NEWADDR netlink message.
func buildNlAddrMsg(ifIndex int, ip net.IP, prefixLen int, family int) []byte {
	var ipBytes []byte
	if family == unix.AF_INET {
		ipBytes = ip.To4()
	} else {
		ipBytes = ip.To16()
	}

	// RTA_LOCAL attribute
	rtaLen := rtaAlign(unix.SizeofRtAttr + len(ipBytes))
	rtaLocal := make([]byte, rtaLen)
	binary.LittleEndian.PutUint16(rtaLocal[0:], uint16(unix.SizeofRtAttr+len(ipBytes)))
	binary.LittleEndian.PutUint16(rtaLocal[2:], unix.IFA_LOCAL)
	copy(rtaLocal[4:], ipBytes)

	// ifaddrmsg
	ifaMsg := unix.IfAddrmsg{
		Family:    uint8(family),
		Prefixlen: uint8(prefixLen),
		Index:     uint32(ifIndex),
	}
	ifaMsgBytes := (*[unix.SizeofIfAddrmsg]byte)(unsafe.Pointer(&ifaMsg))[:]

	// nlmsghdr
	msgLen := unix.SizeofNlMsghdr + unix.SizeofIfAddrmsg + len(rtaLocal)
	nlh := unix.NlMsghdr{
		Len:   uint32(msgLen),
		Type:  unix.RTM_NEWADDR,
		Flags: unix.NLM_F_REQUEST | unix.NLM_F_CREATE | unix.NLM_F_REPLACE | unix.NLM_F_ACK,
		Seq:   1,
	}
	nlhBytes := (*[unix.SizeofNlMsghdr]byte)(unsafe.Pointer(&nlh))[:]

	msg := make([]byte, 0, msgLen)
	msg = append(msg, nlhBytes...)
	msg = append(msg, ifaMsgBytes...)
	msg = append(msg, rtaLocal...)
	return msg
}

// buildNlRouteMsg constructs a RTM_NEWROUTE netlink message.
func buildNlRouteMsg(ifIndex int, dst *net.IPNet) []byte {
	prefixLen, _ := dst.Mask.Size()
	ip := dst.IP

	var family uint8
	var ipBytes []byte
	if ip4 := ip.To4(); ip4 != nil {
		family = unix.AF_INET
		ipBytes = ip4
	} else {
		family = unix.AF_INET6
		ipBytes = ip.To16()
	}

	// RTA_DST
	rtaLen := rtaAlign(unix.SizeofRtAttr + len(ipBytes))
	rtaDst := make([]byte, rtaLen)
	binary.LittleEndian.PutUint16(rtaDst[0:], uint16(unix.SizeofRtAttr+len(ipBytes)))
	binary.LittleEndian.PutUint16(rtaDst[2:], unix.RTA_DST)
	copy(rtaDst[4:], ipBytes)

	// RTA_OIF
	oif := make([]byte, rtaAlign(unix.SizeofRtAttr+4))
	binary.LittleEndian.PutUint16(oif[0:], uint16(unix.SizeofRtAttr+4))
	binary.LittleEndian.PutUint16(oif[2:], unix.RTA_OIF)
	binary.LittleEndian.PutUint32(oif[4:], uint32(ifIndex))

	rtmMsg := unix.RtMsg{
		Family:   family,
		Dst_len:  uint8(prefixLen),
		Protocol: unix.RTPROT_STATIC,
		Scope:    unix.RT_SCOPE_UNIVERSE,
		Type:     unix.RTN_UNICAST,
	}
	rtmBytes := (*[unix.SizeofRtMsg]byte)(unsafe.Pointer(&rtmMsg))[:]

	msgLen := unix.SizeofNlMsghdr + unix.SizeofRtMsg + len(rtaDst) + len(oif)
	nlh := unix.NlMsghdr{
		Len:   uint32(msgLen),
		Type:  unix.RTM_NEWROUTE,
		Flags: unix.NLM_F_REQUEST | unix.NLM_F_CREATE | unix.NLM_F_REPLACE | unix.NLM_F_ACK,
		Seq:   2,
	}
	nlhBytes := (*[unix.SizeofNlMsghdr]byte)(unsafe.Pointer(&nlh))[:]

	msg := make([]byte, 0, msgLen)
	msg = append(msg, nlhBytes...)
	msg = append(msg, rtmBytes...)
	msg = append(msg, rtaDst...)
	msg = append(msg, oif...)
	return msg
}

// recvNlAck reads the ACK from the netlink socket.
func recvNlAck(sock int) error {
	buf := make([]byte, 4096)
	n, _, err := syscall.Recvfrom(sock, buf, 0)
	if err != nil {
		return fmt.Errorf("netlink recv: %w", err)
	}
	msgs, err := parseNetlinkMessage(buf[:n])
	if err != nil {
		return fmt.Errorf("parse netlink: %w", err)
	}
	for _, msg := range msgs {
		if msg.Header.Type == unix.NLMSG_ERROR {
			errno := int32(binary.LittleEndian.Uint32(msg.Data[:4]))
			if errno != 0 {
				return fmt.Errorf("netlink error: %w", unix.Errno(-errno))
			}
		}
	}
	return nil
}

// rtaAlign rounds length up to RTA_ALIGNTO (4 bytes)
func rtaAlign(length int) int {
	return (length + 3) & ^3
}

// parseNetlinkMessage parses netlink messages from a byte slice.
// This is a compatibility function for cross-compilation.
func parseNetlinkMessage(b []byte) ([]syscall.NetlinkMessage, error) {
	var msgs []syscall.NetlinkMessage
	for len(b) >= unix.SizeofNlMsghdr {
		h := (*syscall.NlMsghdr)(unsafe.Pointer(&b[0]))
		if int(h.Len) < unix.SizeofNlMsghdr || int(h.Len) > len(b) {
			return nil, fmt.Errorf("invalid netlink message length: %d", h.Len)
		}
		m := syscall.NetlinkMessage{
			Header: *h,
			Data:   b[unix.SizeofNlMsghdr:h.Len],
		}
		msgs = append(msgs, m)
		b = b[h.Len:]
	}
	return msgs, nil
}
