package dns

import (
	"net"
	"strings"
	"sync"
)

// MagicDNS maps hostnames under *.web.karadul to virtual IPs.
type MagicDNS struct {
	mu      sync.RWMutex
	records map[string]net.IP // lowercase hostname → VIP
}

// NewMagicDNS creates an empty MagicDNS resolver.
func NewMagicDNS() *MagicDNS {
	return &MagicDNS{records: make(map[string]net.IP)}
}

// Update replaces the entire record set.
// entries is a map of hostname → IP (without the .web.karadul suffix).
func (m *MagicDNS) Update(entries map[string]net.IP) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = make(map[string]net.IP, len(entries))
	for host, ip := range entries {
		m.records[strings.ToLower(host)] = ip
	}
}

// Set adds or updates a single record.
func (m *MagicDNS) Set(hostname string, ip net.IP) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records[strings.ToLower(hostname)] = ip
}

// Delete removes the record for hostname.
func (m *MagicDNS) Delete(hostname string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.records, strings.ToLower(hostname))
}

// Lookup returns the virtual IP for the given hostname (without domain suffix).
// Returns nil if not found.
func (m *MagicDNS) Lookup(hostname string) net.IP {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.records[strings.ToLower(hostname)]
}

// All returns a snapshot of all hostname→IP mappings.
func (m *MagicDNS) All() map[string]net.IP {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]net.IP, len(m.records))
	for k, v := range m.records {
		out[k] = v
	}
	return out
}
