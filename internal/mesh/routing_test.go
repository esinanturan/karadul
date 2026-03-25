package mesh

import (
	"net"
	"testing"

	klog "github.com/ersinkoc/karadul/internal/log"
)

func TestRouter_DirectVIP(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 1
	vip := net.ParseIP("100.64.0.2")
	mgr.AddOrUpdate(pubKey, "node-a", "id-a", vip, "", nil)

	router := NewRouter(mgr)

	peer, err := router.RoutePacket(vip)
	if err != nil {
		t.Fatalf("route %s: %v", vip, err)
	}
	if peer.PublicKey != pubKey {
		t.Fatal("wrong peer returned for direct VIP")
	}
}

func TestRouter_SubnetRoute(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 2
	vip := net.ParseIP("100.64.0.3")
	mgr.AddOrUpdate(pubKey, "node-b", "id-b", vip, "", []string{"192.168.10.0/24"})

	router := NewRouter(mgr)

	// A host in the advertised subnet should route to node-b.
	target := net.ParseIP("192.168.10.55")
	peer, err := router.RoutePacket(target)
	if err != nil {
		t.Fatalf("route %s: %v", target, err)
	}
	if peer.PublicKey != pubKey {
		t.Fatal("wrong peer for subnet route")
	}
}

func TestRouter_LongestPrefixMatch(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var k1, k2 [32]byte
	k1[0], k2[0] = 1, 2

	// Node 1 advertises 10.0.0.0/8
	mgr.AddOrUpdate(k1, "n1", "id1", net.ParseIP("100.64.0.1"), "", []string{"10.0.0.0/8"})
	// Node 2 advertises 10.1.2.0/24 (more specific)
	mgr.AddOrUpdate(k2, "n2", "id2", net.ParseIP("100.64.0.2"), "", []string{"10.1.2.0/24"})

	router := NewRouter(mgr)

	// 10.1.2.5 matches both; /24 wins.
	peer, err := router.RoutePacket(net.ParseIP("10.1.2.5"))
	if err != nil {
		t.Fatal(err)
	}
	if peer.PublicKey != k2 {
		t.Fatal("longer prefix should win")
	}

	// 10.99.0.1 only matches /8.
	peer, err = router.RoutePacket(net.ParseIP("10.99.0.1"))
	if err != nil {
		t.Fatal(err)
	}
	if peer.PublicKey != k1 {
		t.Fatal("should match /8 route")
	}
}

func TestRouter_NoRoute(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	router := NewRouter(mgr)
	_, err := router.RoutePacket(net.ParseIP("8.8.8.8"))
	if err == nil {
		t.Fatal("expected error for unroutable address")
	}
}

// TestRouter_VIPExpired verifies that a VIP match for an expired peer is skipped
// and the packet falls through to subsequent rules (covers the false branch of
// "if p.GetState() != PeerExpired" in Rule 1).
func TestRouter_VIPExpired(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 9
	vip := net.ParseIP("100.64.0.9")
	mgr.AddOrUpdate(pubKey, "expired-node", "id-exp", vip, "", nil)

	peer, _ := mgr.GetPeer(pubKey)
	// Force the peer into the Expired state via the state machine.
	peer.Transition(PeerDirect)
	peer.Transition(PeerIdle)
	peer.Transition(PeerExpired)

	router := NewRouter(mgr)
	// Routing to the expired peer's VIP should fall through and return no-route.
	_, err := router.RoutePacket(vip)
	if err == nil {
		t.Fatal("expected no-route error when VIP peer is expired")
	}
}

// TestRouter_ExitNodeExpired verifies that an expired exit node is not used as
// a default route (covers the false branch of exit-node guard in Rule 3).
func TestRouter_ExitNodeExpired(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 8
	vip := net.ParseIP("100.64.0.8")
	mgr.AddOrUpdate(pubKey, "ex-node", "id-ex2", vip, "", nil)

	peer, _ := mgr.GetPeer(pubKey)
	// Expire the peer before setting it as exit node.
	peer.Transition(PeerDirect)
	peer.Transition(PeerIdle)
	peer.Transition(PeerExpired)

	router := NewRouter(mgr)
	router.SetExitNode(peer)

	_, err := router.RoutePacket(net.ParseIP("8.8.8.8"))
	if err == nil {
		t.Fatal("expected no-route error when exit node is expired")
	}
}

func TestRouter_ExitNode(t *testing.T) {
	mgr := NewManager(klog.New(nil, klog.LevelError, klog.FormatText), nil)
	defer mgr.Stop()

	var pubKey [32]byte
	pubKey[0] = 5
	vip := net.ParseIP("100.64.0.5")
	mgr.AddOrUpdate(pubKey, "exit-node", "id-ex", vip, "", nil)

	peer, _ := mgr.GetPeer(pubKey)
	peer.Transition(PeerDirect)

	router := NewRouter(mgr)
	router.SetExitNode(peer)

	// Any random IP should route via exit node.
	got, err := router.RoutePacket(net.ParseIP("1.2.3.4"))
	if err != nil {
		t.Fatalf("expected exit node route: %v", err)
	}
	if got.PublicKey != pubKey {
		t.Fatal("wrong peer for exit node route")
	}
}
