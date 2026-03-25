package dns

import (
	"net"
	"testing"
)

func TestMagicDNS_SetLookup(t *testing.T) {
	m := NewMagicDNS()
	ip := net.ParseIP("100.64.0.5")
	m.Set("node-a", ip)

	got := m.Lookup("node-a")
	if got == nil || !got.Equal(ip) {
		t.Fatalf("Lookup(node-a) = %v, want %v", got, ip)
	}
}

func TestMagicDNS_LookupMissing(t *testing.T) {
	m := NewMagicDNS()
	if m.Lookup("nonexistent") != nil {
		t.Fatal("lookup of nonexistent name should return nil")
	}
}

func TestMagicDNS_Delete(t *testing.T) {
	m := NewMagicDNS()
	m.Set("node-b", net.ParseIP("100.64.0.6"))
	m.Delete("node-b")
	if m.Lookup("node-b") != nil {
		t.Fatal("deleted entry should not be found")
	}
}

func TestMagicDNS_Update(t *testing.T) {
	m := NewMagicDNS()
	m.Set("old-node", net.ParseIP("100.64.0.1"))

	newEntries := map[string]net.IP{
		"node-x": net.ParseIP("100.64.0.10"),
		"node-y": net.ParseIP("100.64.0.11"),
	}
	m.Update(newEntries)

	if m.Lookup("old-node") != nil {
		t.Fatal("old entry should be gone after Update")
	}
	if m.Lookup("node-x") == nil {
		t.Fatal("node-x should exist after Update")
	}
	if m.Lookup("node-y") == nil {
		t.Fatal("node-y should exist after Update")
	}
}

func TestMagicDNS_All(t *testing.T) {
	m := NewMagicDNS()
	m.Set("host1", net.ParseIP("100.64.0.1"))
	m.Set("host2", net.ParseIP("100.64.0.2"))

	all := m.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d entries, want 2", len(all))
	}
}
