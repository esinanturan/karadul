package coordinator

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

// IPPool manages virtual IP allocation within a CGNAT subnet (default 100.64.0.0/10).
type IPPool struct {
	mu       sync.Mutex
	subnet   *net.IPNet
	base     uint32 // first host address (subnet base + 1)
	size     uint32 // number of usable host addresses
	used     map[uint32]string // ip → nodeID
	byNode   map[string]uint32 // nodeID → ip
}

// NewIPPool creates an IPPool for the given CIDR string (e.g. "100.64.0.0/10").
func NewIPPool(cidr string) (*IPPool, error) {
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("parse cidr %q: %w", cidr, err)
	}
	ones, bits := subnet.Mask.Size()
	if bits != 32 {
		return nil, fmt.Errorf("only IPv4 supported for IP pool")
	}
	hostBits := uint32(bits - ones)
	if hostBits < 2 {
		return nil, fmt.Errorf("subnet too small")
	}
	base := ipToUint32(subnet.IP) + 1 // skip network address
	size := (uint32(1) << hostBits) - 2 // exclude broadcast

	return &IPPool{
		subnet: subnet,
		base:   base,
		size:   size,
		used:   make(map[uint32]string),
		byNode: make(map[string]uint32),
	}, nil
}

// Allocate assigns a virtual IP to nodeID. Returns the same IP if already allocated.
func (p *IPPool) Allocate(nodeID string) (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n, ok := p.byNode[nodeID]; ok {
		return uint32ToIP(n), nil
	}

	for i := uint32(0); i < p.size; i++ {
		addr := p.base + i
		if _, taken := p.used[addr]; !taken {
			p.used[addr] = nodeID
			p.byNode[nodeID] = addr
			return uint32ToIP(addr), nil
		}
	}
	return nil, fmt.Errorf("ip pool exhausted")
}

// Release frees the IP assigned to nodeID.
func (p *IPPool) Release(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if addr, ok := p.byNode[nodeID]; ok {
		delete(p.used, addr)
		delete(p.byNode, nodeID)
	}
}

// Contains reports whether ip is within the pool's subnet.
func (p *IPPool) Contains(ip net.IP) bool {
	return p.subnet.Contains(ip)
}

// Reserve marks ip as taken by nodeID (used during restore from disk).
func (p *IPPool) Reserve(nodeID string, ip net.IP) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	addr := ipToUint32(ip.To4())
	if other, taken := p.used[addr]; taken && other != nodeID {
		return fmt.Errorf("ip %s already used by %s", ip, other)
	}
	p.used[addr] = nodeID
	p.byNode[nodeID] = addr
	return nil
}

func ipToUint32(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip4)
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
