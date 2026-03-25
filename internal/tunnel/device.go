package tunnel

import "net"

// Device is the interface for a TUN/TAP virtual network device.
// Implementations are platform-specific (tun_linux.go, tun_darwin.go).
type Device interface {
	// Name returns the OS interface name, e.g. "tun0" or "utun3".
	Name() string

	// Read reads one IP packet from the device into buf.
	// Returns the number of bytes read.
	Read(buf []byte) (int, error)

	// Write writes one IP packet to the device.
	Write(buf []byte) (int, error)

	// MTU returns the current MTU.
	MTU() int

	// SetMTU sets the MTU.
	SetMTU(mtu int) error

	// SetAddr assigns the given IP address and prefix length to the interface.
	SetAddr(ip net.IP, prefixLen int) error

	// AddRoute adds a route through this device.
	AddRoute(dst *net.IPNet) error

	// Close closes the device.
	Close() error
}
