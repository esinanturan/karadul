package nat

import (
	"fmt"
	"net"
)

// NATType classifies the type of NAT in front of this node.
type NATType int

const (
	NATUnknown        NATType = iota
	NATDirect                 // No NAT; public IP is directly reachable
	NATFullCone               // Full Cone: any external host can send to mapped port
	NATRestrictedCone         // Restricted Cone: external host must have received a packet first
	NATPortRestricted         // Port Restricted: external host+port must match
	NATSymmetric              // Symmetric: each destination gets a different mapping
)

func (n NATType) String() string {
	switch n {
	case NATDirect:
		return "direct"
	case NATFullCone:
		return "full-cone"
	case NATRestrictedCone:
		return "restricted-cone"
	case NATPortRestricted:
		return "port-restricted"
	case NATSymmetric:
		return "symmetric"
	default:
		return "unknown"
	}
}

// DefaultSTUNServers are public STUN servers used for NAT detection.
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
}

// DetectNATType uses two STUN servers to classify the local NAT type.
// conn is the local UDP socket to use (or nil to create a new one).
func DetectNATType(conn *net.UDPConn) (NATType, *net.UDPAddr, error) {
	if conn == nil {
		var err error
		conn, err = net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
		if err != nil {
			return NATUnknown, nil, fmt.Errorf("bind udp: %w", err)
		}
		defer conn.Close()
	}

	// Query first STUN server.
	r1, err := BindingRequest(conn, DefaultSTUNServers[0])
	if err != nil {
		return NATUnknown, nil, fmt.Errorf("stun1: %w", err)
	}

	// Query second STUN server.
	r2, err := BindingRequest(conn, DefaultSTUNServers[1])
	if err != nil {
		// Can't compare — assume port-restricted.
		return NATPortRestricted, r1.PublicAddr, nil
	}

	// Get local address to check if we have a direct/public IP.
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if localAddr.IP.Equal(r1.PublicAddr.IP) {
		return NATDirect, r1.PublicAddr, nil
	}

	// Compare the mapped addresses from the two servers.
	// Same port from both → cone NAT. Different port → symmetric.
	if r1.PublicAddr.Port == r2.PublicAddr.Port {
		return NATFullCone, r1.PublicAddr, nil
	}
	return NATSymmetric, r1.PublicAddr, nil
}
