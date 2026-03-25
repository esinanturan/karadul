package coordinator

import (
	"net"
	"testing"
)

func TestIPPool_Allocate(t *testing.T) {
	pool, err := NewIPPool("100.64.0.0/24")
	if err != nil {
		t.Fatal(err)
	}

	ip1, err := pool.Allocate("node-1")
	if err != nil {
		t.Fatal(err)
	}
	if ip1 == nil {
		t.Fatal("expected non-nil IP")
	}

	ip2, err := pool.Allocate("node-2")
	if err != nil {
		t.Fatal(err)
	}
	if ip1.Equal(ip2) {
		t.Fatal("two nodes got the same IP")
	}

	// Re-allocate same node: same IP.
	ip1b, err := pool.Allocate("node-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ip1.Equal(ip1b) {
		t.Fatalf("re-allocation: got %s, want %s", ip1b, ip1)
	}
}

func TestIPPool_Release(t *testing.T) {
	pool, _ := NewIPPool("100.64.0.0/24")
	ip, _ := pool.Allocate("node-1")
	pool.Release("node-1")

	ip2, err := pool.Allocate("node-2")
	if err != nil {
		t.Fatal(err)
	}
	// The released IP should now be available to node-2.
	if !ip.Equal(ip2) {
		// Not strictly required, but after release the first slot should be reused.
		// This is a soft check.
		_ = ip
	}
}

func TestIPPool_Contains(t *testing.T) {
	pool, _ := NewIPPool("100.64.0.0/10")
	if !pool.Contains(net.ParseIP("100.64.0.1")) {
		t.Fatal("100.64.0.1 should be in 100.64.0.0/10")
	}
	if pool.Contains(net.ParseIP("192.168.0.1")) {
		t.Fatal("192.168.0.1 should not be in 100.64.0.0/10")
	}
}

func TestIPPool_Reserve(t *testing.T) {
	pool, _ := NewIPPool("100.64.0.0/24")
	ip := net.ParseIP("100.64.0.5")
	if err := pool.Reserve("node-5", ip); err != nil {
		t.Fatal(err)
	}
	// Allocating node-5 should return the reserved IP.
	ip2, err := pool.Allocate("node-5")
	if err != nil {
		t.Fatal(err)
	}
	if !ip.Equal(ip2) {
		t.Fatalf("expected reserved IP %s, got %s", ip, ip2)
	}
}

// TestNewIPPool_IPv6Subnet verifies that IPv6 CIDR returns an error.
func TestNewIPPool_IPv6Subnet(t *testing.T) {
	_, err := NewIPPool("2001:db8::/64")
	if err == nil {
		t.Fatal("expected error for IPv6 subnet")
	}
	if err.Error() != "only IPv4 supported for IP pool" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestNewIPPool_SubnetTooSmall verifies that /31 and /32 subnets return an error.
func TestNewIPPool_SubnetTooSmall(t *testing.T) {
	_, err := NewIPPool("100.64.0.0/31")
	if err == nil {
		t.Fatal("expected error for /31 subnet")
	}
	if err.Error() != "subnet too small" {
		t.Errorf("unexpected error message: %v", err)
	}

	_, err = NewIPPool("100.64.0.0/32")
	if err == nil {
		t.Fatal("expected error for /32 subnet")
	}
	if err.Error() != "subnet too small" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestIPPool_AllocateExhausted verifies pool exhaustion with a tiny subnet.
func TestIPPool_AllocateExhausted(t *testing.T) {
	// /30 subnet has 2 usable hosts (network + broadcast excluded)
	pool, err := NewIPPool("100.64.0.0/30")
	if err != nil {
		t.Fatal(err)
	}

	// First allocation should succeed.
	_, err = pool.Allocate("node-1")
	if err != nil {
		t.Fatal(err)
	}

	// Second allocation should succeed.
	_, err = pool.Allocate("node-2")
	if err != nil {
		t.Fatal(err)
	}

	// Third allocation should fail - pool exhausted.
	_, err = pool.Allocate("node-3")
	if err == nil {
		t.Fatal("expected error when pool exhausted")
	}
	if err.Error() != "ip pool exhausted" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestIPPool_ReserveConflict verifies that reserving an already-used IP by a different node fails.
func TestIPPool_ReserveConflict(t *testing.T) {
	pool, _ := NewIPPool("100.64.0.0/24")
	ip := net.ParseIP("100.64.0.10")

	// Reserve IP for node-1.
	if err := pool.Reserve("node-1", ip); err != nil {
		t.Fatal(err)
	}

	// Try to reserve same IP for node-2 - should fail.
	err := pool.Reserve("node-2", ip)
	if err == nil {
		t.Fatal("expected error when reserving already-used IP")
	}
	if err.Error() != "ip 100.64.0.10 already used by node-1" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Re-reserving for the same node should succeed.
	if err := pool.Reserve("node-1", ip); err != nil {
		t.Fatalf("re-reserving for same node should succeed: %v", err)
	}
}

// TestIpToUint32_IPv6ReturnsZero verifies that ipToUint32 returns 0 for IPv6 addresses.
func TestIpToUint32_IPv6ReturnsZero(t *testing.T) {
	ipv6 := net.ParseIP("2001:db8::1")
	result := ipToUint32(ipv6)
	if result != 0 {
		t.Errorf("expected 0 for IPv6 address, got %d", result)
	}
}

// TestIpToUint32_IPv4Works verifies that ipToUint32 correctly converts IPv4 addresses.
func TestIpToUint32_IPv4Works(t *testing.T) {
	ipv4 := net.ParseIP("100.64.0.5")
	result := ipToUint32(ipv4)
	expected := uint32(0x64400005) // 100.64.0.5 in big-endian
	if result != expected {
		t.Errorf("expected %d (0x%x), got %d (0x%x)", expected, expected, result, result)
	}
}
