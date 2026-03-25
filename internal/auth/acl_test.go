package auth

import (
	"net"
	"testing"
)

func TestACL_DefaultAllowAll(t *testing.T) {
	e := NewEngine(ACLPolicy{}) // no rules → allow all
	if !e.Allow(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"), 80) {
		t.Fatal("empty policy should allow all")
	}
}

func TestACL_AllowRule(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"100.64.0.0/10"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)
	if !e.Allow(net.ParseIP("100.64.0.5"), net.ParseIP("8.8.8.8"), 53) {
		t.Fatal("should allow CGNAT src to any dst")
	}
}

func TestACL_DenyRule(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "deny", Src: []string{"*"}, Dst: []string{"192.168.99.0/24"}},
		},
	}
	e := NewEngine(policy)
	if e.Allow(net.ParseIP("100.64.0.1"), net.ParseIP("192.168.99.5"), 22) {
		t.Fatal("should deny access to 192.168.99.0/24")
	}
}

func TestACL_PortMatch(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"*"}, Dst: []string{"*"}, Ports: []string{"80", "443"}},
		},
	}
	e := NewEngine(policy)

	if !e.Allow(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"), 80) {
		t.Fatal("port 80 should be allowed")
	}
	if !e.Allow(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"), 443) {
		t.Fatal("port 443 should be allowed")
	}
	if e.Allow(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"), 22) {
		t.Fatal("port 22 should be denied (no matching rule)")
	}
}

func TestACL_PortRange(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"*"}, Dst: []string{"*"}, Ports: []string{"1000-2000"}},
		},
	}
	e := NewEngine(policy)

	if !e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 1500) {
		t.Fatal("port 1500 inside 1000-2000 should be allowed")
	}
	if e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 999) {
		t.Fatal("port 999 outside range should be denied")
	}
}

func TestACL_GroupMatch(t *testing.T) {
	policy := ACLPolicy{
		Groups: map[string][]string{
			"admins": {"100.64.0.1/32", "100.64.0.2/32"},
		},
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"group:admins"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)

	if !e.Allow(net.ParseIP("100.64.0.1"), net.ParseIP("10.0.0.1"), 0) {
		t.Fatal("group member should be allowed")
	}
	if e.Allow(net.ParseIP("100.64.0.9"), net.ParseIP("10.0.0.1"), 0) {
		t.Fatal("non-group member should be denied")
	}
}

func TestACL_WildcardSrc(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"*"}, Dst: []string{"10.0.0.0/8"}},
			{Action: "deny", Src: []string{"*"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)

	if !e.Allow(net.ParseIP("192.168.1.1"), net.ParseIP("10.5.5.5"), 0) {
		t.Fatal("should match first allow rule")
	}
	if e.Allow(net.ParseIP("192.168.1.1"), net.ParseIP("172.16.0.1"), 0) {
		t.Fatal("should match second deny rule")
	}
}

func TestACL_UpdatePolicy(t *testing.T) {
	e := NewEngine(ACLPolicy{})
	if !e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 0) {
		t.Fatal("empty policy should allow all")
	}

	// Update to deny-all.
	e.UpdatePolicy(ACLPolicy{
		Rules: []ACLRule{
			{Action: "deny", Src: []string{"*"}, Dst: []string{"*"}},
		},
	})
	if e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 0) {
		t.Fatal("updated deny-all policy should block")
	}
}

func TestACLPolicy_Validate(t *testing.T) {
	good := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"*"}, Dst: []string{"*"}},
		},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid policy: %v", err)
	}

	bad := ACLPolicy{
		Rules: []ACLRule{
			{Action: "permit", Src: []string{"*"}, Dst: []string{"*"}},
		},
	}
	if err := bad.Validate(); err == nil {
		t.Fatal("invalid action 'permit' should fail validation")
	}

	noSrc := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{}, Dst: []string{"*"}},
		},
	}
	if err := noSrc.Validate(); err == nil {
		t.Fatal("missing src should fail validation")
	}

	noDst := ACLPolicy{
		Rules: []ACLRule{
			{Action: "deny", Src: []string{"*"}, Dst: []string{}},
		},
	}
	if err := noDst.Validate(); err == nil {
		t.Fatal("missing dst should fail validation")
	}
}

func TestACL_PlainIPMatch(t *testing.T) {
	// Src and Dst as plain IPs (no CIDR slash).
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"192.168.1.10"}, Dst: []string{"10.0.0.1"}},
		},
	}
	e := NewEngine(policy)

	if !e.Allow(net.ParseIP("192.168.1.10"), net.ParseIP("10.0.0.1"), 0) {
		t.Fatal("exact IP match should allow")
	}
	if e.Allow(net.ParseIP("192.168.1.11"), net.ParseIP("10.0.0.1"), 0) {
		t.Fatal("non-matching src should be denied")
	}
}

func TestACL_InvalidCIDR(t *testing.T) {
	// An invalid CIDR entry should not cause a panic and simply not match.
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"not-a-cidr"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)
	if e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 0) {
		t.Fatal("invalid CIDR entry should not match")
	}
}

func TestACL_DefaultDeny(t *testing.T) {
	// Rules exist but none match → default deny.
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"10.0.0.0/8"}, Dst: []string{"10.0.0.0/8"}},
		},
	}
	e := NewEngine(policy)
	// Traffic from outside the rule should be denied.
	if e.Allow(net.ParseIP("192.168.1.1"), net.ParseIP("172.16.0.1"), 0) {
		t.Fatal("unmatched rules should default to deny")
	}
}

func TestACL_PortZero_NoPortFilter(t *testing.T) {
	// Port=0 with no Ports filter → any port is permitted.
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"*"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)
	if !e.Allow(net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2"), 0) {
		t.Fatal("should allow when no port filter")
	}
	if !e.Allow(net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2"), 443) {
		t.Fatal("should allow port 443 when no port filter")
	}
}

func TestACL_PortFilter_NoMatch(t *testing.T) {
	// Port filter set; packet port not in list → rule doesn't match → default deny.
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"*"}, Dst: []string{"*"}, Ports: []string{"80"}},
		},
	}
	e := NewEngine(policy)
	if !e.Allow(net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2"), 80) {
		t.Fatal("port 80 should be allowed")
	}
	if e.Allow(net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2"), 443) {
		t.Fatal("port 443 should be denied (not in port filter)")
	}
}

func TestACL_GroupMatchCIDR(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"group:admins"}, Dst: []string{"*"}},
			{Action: "deny", Src: []string{"*"}, Dst: []string{"*"}},
		},
		Groups: map[string][]string{
			"admins": {"10.10.10.0/24"},
		},
	}
	e := NewEngine(policy)

	if !e.Allow(net.ParseIP("10.10.10.5"), net.ParseIP("1.2.3.4"), 0) {
		t.Fatal("group member should be allowed")
	}
	if e.Allow(net.ParseIP("10.10.11.1"), net.ParseIP("1.2.3.4"), 0) {
		t.Fatal("non-group member should be denied")
	}
}

func TestACL_UnknownGroup(t *testing.T) {
	// group: entry that references a non-existent group should not match.
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"group:nobody"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)
	if e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 0) {
		t.Fatal("unknown group should not match")
	}
}

// TestACL_InvalidCIDRWithSlash covers the net.ParseCIDR error path in matchesCIDR
// (entry contains "/" but is not valid CIDR notation).
func TestACL_InvalidCIDRWithSlash(t *testing.T) {
	policy := ACLPolicy{
		Rules: []ACLRule{
			{Action: "allow", Src: []string{"not-valid/24"}, Dst: []string{"*"}},
		},
	}
	e := NewEngine(policy)
	if e.Allow(net.ParseIP("1.2.3.4"), net.ParseIP("5.6.7.8"), 0) {
		t.Fatal("invalid CIDR with slash should not match")
	}
}

func TestACL_ConcurrentReadWrite(t *testing.T) {
	e := NewEngine(ACLPolicy{})
	done := make(chan struct{})

	// Writer goroutine.
	go func() {
		for i := 0; i < 1000; i++ {
			e.UpdatePolicy(ACLPolicy{
				Rules: []ACLRule{
					{Action: "allow", Src: []string{"*"}, Dst: []string{"*"}},
				},
			})
		}
		close(done)
	}()

	// Reader goroutines.
	for i := 0; i < 4; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					e.Allow(net.ParseIP("1.1.1.1"), net.ParseIP("2.2.2.2"), 80)
				}
			}
		}()
	}
	<-done
}
