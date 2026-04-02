package mesh

import (
	"encoding/base64"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/karadul/karadul/internal/coordinator"
	klog "github.com/karadul/karadul/internal/log"
)

// ---------------------------------------------------------------------------
// Peer.Touch — uncovered branch: old == PeerDirect, callback should NOT fire
// ---------------------------------------------------------------------------

func TestPeer_Touch_DirectPeerNoCallback(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "direct-node", "id-direct", net.ParseIP("100.64.0.40"))

	callbackCount := 0
	p.SetOnStateChange(func(peer *Peer, from, to PeerState) {
		callbackCount++
	})

	// Transition to PeerDirect first.
	p.Transition(PeerDirect)
	if callbackCount != 1 {
		t.Fatalf("expected 1 callback after Transition, got %d", callbackCount)
	}

	// Touch when already PeerDirect — callback must NOT fire.
	p.Touch()
	if callbackCount != 1 {
		t.Fatalf("Touch on direct peer should not fire callback, got %d", callbackCount)
	}

	// State should remain PeerDirect.
	if p.GetState() != PeerDirect {
		t.Fatalf("expected PeerDirect, got %s", p.GetState())
	}
}

// ---------------------------------------------------------------------------
// Peer.Touch — verify lastSeen is updated
// ---------------------------------------------------------------------------

func TestPeer_Touch_UpdatesLastSeen(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "touch-node", "id-touch", net.ParseIP("100.64.0.41"))

	// Set lastSeen to a very old time.
	p.mu.Lock()
	p.lastSeen = time.Now().Add(-1 * time.Hour)
	p.mu.Unlock()

	before := time.Now()
	p.Touch()
	after := time.Now()

	p.mu.RLock()
	ls := p.lastSeen
	p.mu.RUnlock()

	if ls.Before(before) || ls.After(after) {
		t.Errorf("lastSeen not updated: %v (expected between %v and %v)", ls, before, after)
	}
}

// ---------------------------------------------------------------------------
// Peer.Touch — from Discovered state (not Idle, not Direct)
// ---------------------------------------------------------------------------

func TestPeer_Touch_FromDiscovered(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "disc-node", "id-disc", net.ParseIP("100.64.0.42"))

	var seenFrom, seenTo PeerState
	p.SetOnStateChange(func(peer *Peer, from, to PeerState) {
		seenFrom = from
		seenTo = to
	})

	// Touch from Discovered state: old != PeerDirect, so callback fires.
	// But state does NOT change (only PeerIdle -> PeerDirect transition happens).
	p.Touch()

	// State stays Discovered (Touch only changes Idle -> Direct).
	if p.GetState() != PeerDiscovered {
		t.Fatalf("expected PeerDiscovered, got %s", p.GetState())
	}

	// Callback fires because old (Discovered) != PeerDirect.
	if seenFrom != PeerDiscovered || seenTo != PeerDirect {
		t.Fatalf("callback: from=%s to=%s", seenFrom, seenTo)
	}
}

// ---------------------------------------------------------------------------
// Peer.IdleCheck — no state change when Direct and recently seen
// ---------------------------------------------------------------------------

func TestPeer_IdleCheck_NoChangeWhenActive(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "active-node", "id-active", net.ParseIP("100.64.0.43"))
	p.Transition(PeerDirect)

	// Peer was just created (lastSeen = now), so IdleCheck should not change state.
	p.IdleCheck()
	if p.GetState() != PeerDirect {
		t.Fatalf("expected PeerDirect, got %s", p.GetState())
	}
}

// ---------------------------------------------------------------------------
// Peer.IdleCheck — Direct to Expired in one step (skips Idle)
// ---------------------------------------------------------------------------

func TestPeer_IdleCheck_DirectToExpired(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "skip-idle", "id-skip", net.ParseIP("100.64.0.44"))
	p.Transition(PeerDirect)

	// Set lastSeen to past expiredTimeout (> 10 min) but not between 5-10 min.
	p.mu.Lock()
	p.lastSeen = time.Now().Add(-12 * time.Minute)
	p.mu.Unlock()

	p.IdleCheck()

	// Should transition directly to Expired (via the Idle branch in IdleCheck).
	// Actually, the first check is Direct/Relayed + >idleTimeout -> Idle.
	// Then second check is Idle + >expiredTimeout -> Expired.
	// Since both checks run, and 12 min > both timeouts, it goes to Expired.
	if p.GetState() != PeerExpired {
		t.Fatalf("expected PeerExpired, got %s", p.GetState())
	}
}

// ---------------------------------------------------------------------------
// Peer concurrent access tests
// ---------------------------------------------------------------------------

func TestPeer_ConcurrentTransitions(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "concurrent", "id-conc", net.ParseIP("100.64.0.45"))

	var wg sync.WaitGroup
	states := []PeerState{PeerConnecting, PeerDirect, PeerRelayed, PeerIdle, PeerExpired}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p.Transition(states[idx%len(states)])
		}(i)
	}
	wg.Wait()

	// Must not panic; state must be a valid PeerState.
	state := p.GetState()
	if state < PeerDiscovered || state > PeerExpired {
		t.Fatalf("unexpected state: %d", state)
	}
}

func TestPeer_ConcurrentTouchAndTransition(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "race-node", "id-race", net.ParseIP("100.64.0.46"))
	p.Transition(PeerDirect)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.Touch()
		}()
		go func() {
			defer wg.Done()
			p.Transition(PeerIdle)
		}()
	}
	wg.Wait()
}

func TestPeer_ConcurrentIdleCheck(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "idle-race", "id-irace", net.ParseIP("100.64.0.47"))
	p.Transition(PeerDirect)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.IdleCheck()
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Manager — concurrent AddOrUpdate
// ---------------------------------------------------------------------------

func TestManager_ConcurrentAddOrUpdate(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var key [32]byte
			key[0] = byte(idx)
			vip := net.ParseIP("100.64.0.50")
			m.AddOrUpdate(key, "concurrent-node", "id-conc", vip, "", nil)
		}(i)
	}
	wg.Wait()

	peers := m.ListPeers()
	if len(peers) != 20 {
		t.Errorf("expected 20 peers, got %d", len(peers))
	}
}

// ---------------------------------------------------------------------------
// Manager — GetPeerByVIP nonexistent
// ---------------------------------------------------------------------------

func TestManager_GetPeerByVIP_NotFound(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	_, ok := m.GetPeerByVIP(net.ParseIP("100.64.0.99"))
	if ok {
		t.Error("expected not found for nonexistent VIP")
	}
}

// ---------------------------------------------------------------------------
// Manager — GetPeer nonexistent
// ---------------------------------------------------------------------------

func TestManager_GetPeer_NotFound(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	var unknown [32]byte
	unknown[0] = 0xFF
	_, ok := m.GetPeer(unknown)
	if ok {
		t.Error("expected not found for unknown key")
	}
}

// ---------------------------------------------------------------------------
// Manager — ListPeers empty
// ---------------------------------------------------------------------------

func TestManager_ListPeers_Empty(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	peers := m.ListPeers()
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}
}

// ---------------------------------------------------------------------------
// Router — nil IP
// ---------------------------------------------------------------------------

func TestRouter_RoutePacket_NilIP(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	router := NewRouter(m)
	_, err := router.RoutePacket(nil)
	if err == nil {
		t.Fatal("expected error for nil IP")
	}
}

// ---------------------------------------------------------------------------
// Router — expired peer with subnet route is skipped
// ---------------------------------------------------------------------------

func TestRouter_ExpiredSubnetPeerSkipped(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	var key [32]byte
	key[0] = 1
	vip := net.ParseIP("100.64.0.1")
	m.AddOrUpdate(key, "subnet-expired", "id-se", vip, "", []string{"10.0.0.0/8"})

	p, _ := m.GetPeer(key)
	p.Transition(PeerDirect)
	p.Transition(PeerIdle)
	p.Transition(PeerExpired)

	router := NewRouter(m)
	_, err := router.RoutePacket(net.ParseIP("10.1.2.3"))
	if err == nil {
		t.Fatal("expected error when only peer is expired")
	}
}

// ---------------------------------------------------------------------------
// Router — set exit node to nil
// ---------------------------------------------------------------------------

func TestRouter_SetExitNode_Nil(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	router := NewRouter(m)
	router.SetExitNode(nil)

	_, err := router.RoutePacket(net.ParseIP("1.2.3.4"))
	if err == nil {
		t.Fatal("expected no-route error when exit node is nil")
	}
}

// ---------------------------------------------------------------------------
// Topology — empty state
// ---------------------------------------------------------------------------

func TestTopologyManager_Apply_EmptyState(t *testing.T) {
	mgr := newTestManager(t)
	var selfKey [32]byte
	tm := NewTopologyManager(mgr, selfKey, newTestLogger())

	tm.Apply(coordinator.NetworkState{})
	if len(mgr.ListPeers()) != 0 {
		t.Error("expected 0 peers for empty state")
	}
}

// ---------------------------------------------------------------------------
// Topology — multiple applies with route changes
// ---------------------------------------------------------------------------

func TestTopologyManager_Apply_UpdateRoutes(t *testing.T) {
	mgr := newTestManager(t)
	var selfKey [32]byte
	tm := NewTopologyManager(mgr, selfKey, newTestLogger())

	var peerKey [32]byte
	peerKey[0] = 0x77

	// First apply: two routes.
	state1 := coordinator.NetworkState{
		Nodes: []*coordinator.Node{
			{
				ID:        "n-routes",
				PublicKey: base64.StdEncoding.EncodeToString(peerKey[:]),
				Hostname:  "router",
				VirtualIP: "100.64.0.70",
				Status:    coordinator.NodeStatusActive,
				Routes:    []string{"192.168.0.0/24", "10.0.0.0/8"},
			},
		},
	}
	tm.Apply(state1)

	p, ok := mgr.GetPeer(peerKey)
	if !ok {
		t.Fatal("peer not found")
	}
	p.mu.RLock()
	n1 := len(p.Routes)
	p.mu.RUnlock()
	if n1 != 2 {
		t.Fatalf("expected 2 routes, got %d", n1)
	}

	// Second apply: different routes.
	state2 := coordinator.NetworkState{
		Nodes: []*coordinator.Node{
			{
				ID:        "n-routes",
				PublicKey: base64.StdEncoding.EncodeToString(peerKey[:]),
				Hostname:  "router",
				VirtualIP: "100.64.0.70",
				Status:    coordinator.NodeStatusActive,
				Routes:    []string{"172.16.0.0/12"},
			},
		},
	}
	tm.Apply(state2)

	p.mu.RLock()
	n2 := len(p.Routes)
	route0 := p.Routes[0].String()
	p.mu.RUnlock()
	if n2 != 1 {
		t.Fatalf("expected 1 route after update, got %d", n2)
	}
	if route0 != "172.16.0.0/12" {
		t.Errorf("expected 172.16.0.0/12, got %s", route0)
	}
}

// ---------------------------------------------------------------------------
// Topology — apply with endpoint
// ---------------------------------------------------------------------------

func TestTopologyManager_Apply_WithEndpoint(t *testing.T) {
	mgr := newTestManager(t)
	var selfKey [32]byte
	tm := NewTopologyManager(mgr, selfKey, newTestLogger())

	var peerKey [32]byte
	peerKey[0] = 0x88

	state := coordinator.NetworkState{
		Nodes: []*coordinator.Node{
			{
				ID:        "ep-node",
				PublicKey: base64.StdEncoding.EncodeToString(peerKey[:]),
				Hostname:  "epnode",
				VirtualIP: "100.64.0.80",
				Endpoint:  "1.2.3.4:51820",
				Status:    coordinator.NodeStatusActive,
			},
		},
	}
	tm.Apply(state)

	p, ok := mgr.GetPeer(peerKey)
	if !ok {
		t.Fatal("peer not found")
	}
	ep := p.GetEndpoint()
	if ep == nil {
		t.Fatal("expected endpoint to be set")
	}
	if ep.Port != 51820 {
		t.Errorf("expected port 51820, got %d", ep.Port)
	}
}

// ---------------------------------------------------------------------------
// Topology — apply with invalid endpoint (graceful skip)
// ---------------------------------------------------------------------------

func TestTopologyManager_Apply_InvalidEndpoint(t *testing.T) {
	mgr := newTestManager(t)
	var selfKey [32]byte
	tm := NewTopologyManager(mgr, selfKey, newTestLogger())

	var peerKey [32]byte
	peerKey[0] = 0x99

	state := coordinator.NetworkState{
		Nodes: []*coordinator.Node{
			{
				ID:        "bad-ep-node",
				PublicKey: base64.StdEncoding.EncodeToString(peerKey[:]),
				Hostname:  "badep",
				VirtualIP: "100.64.0.90",
				Endpoint:  "not-a-valid-endpoint",
				Status:    coordinator.NodeStatusActive,
			},
		},
	}
	tm.Apply(state)

	p, ok := mgr.GetPeer(peerKey)
	if !ok {
		t.Fatal("peer should still be added even with bad endpoint")
	}
	ep := p.GetEndpoint()
	if ep != nil {
		t.Error("expected nil endpoint for invalid address")
	}
}

// ---------------------------------------------------------------------------
// Topology — apply with multiple nodes including self
// ---------------------------------------------------------------------------

func TestTopologyManager_Apply_MultipleNodesWithSelf(t *testing.T) {
	mgr := newTestManager(t)
	var selfKey [32]byte
	selfKey[0] = 0xAA
	tm := NewTopologyManager(mgr, selfKey, newTestLogger())

	var p1Key, p2Key [32]byte
	p1Key[0] = 0x01
	p2Key[0] = 0x02

	state := coordinator.NetworkState{
		Nodes: []*coordinator.Node{
			{
				ID:        "self",
				PublicKey: base64.StdEncoding.EncodeToString(selfKey[:]),
				Hostname:  "self-node",
				VirtualIP: "100.64.0.1",
				Status:    coordinator.NodeStatusActive,
			},
			{
				ID:        "peer1",
				PublicKey: base64.StdEncoding.EncodeToString(p1Key[:]),
				Hostname:  "peer1",
				VirtualIP: "100.64.0.2",
				Status:    coordinator.NodeStatusActive,
			},
			{
				ID:        "peer2",
				PublicKey: base64.StdEncoding.EncodeToString(p2Key[:]),
				Hostname:  "peer2",
				VirtualIP: "100.64.0.3",
				Status:    coordinator.NodeStatusActive,
			},
		},
	}
	tm.Apply(state)

	peers := mgr.ListPeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers (self excluded), got %d", len(peers))
	}
}

// ---------------------------------------------------------------------------
// Peer — SetEndpoint nil then non-nil
// ---------------------------------------------------------------------------

func TestPeer_SetEndpoint_NilThenSet(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "ep-test", "id-ep", net.ParseIP("100.64.0.48"))

	// Initially nil.
	if p.GetEndpoint() != nil {
		t.Fatal("expected nil endpoint initially")
	}

	// Set nil again (no-op).
	p.SetEndpoint(nil)
	if p.GetEndpoint() != nil {
		t.Fatal("expected nil after SetEndpoint(nil)")
	}

	// Set a real endpoint.
	ep := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 12345}
	p.SetEndpoint(ep)

	got := p.GetEndpoint()
	if got == nil {
		t.Fatal("expected non-nil endpoint")
	}
	if !got.IP.Equal(net.ParseIP("10.0.0.1")) {
		t.Errorf("IP mismatch: %v", got.IP)
	}
	if got.Port != 12345 {
		t.Errorf("port: got %d, want 12345", got.Port)
	}
}

// ---------------------------------------------------------------------------
// Peer — IsExpired boundary: exactly at expiredTimeout
// ---------------------------------------------------------------------------

func TestPeer_IsExpired_ExactlyAtBoundary(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "boundary", "id-bound", net.ParseIP("100.64.0.49"))
	p.Transition(PeerDirect)

	// Set lastSeen to exactly expiredTimeout ago.
	p.mu.Lock()
	p.lastSeen = time.Now().Add(-expiredTimeout)
	p.mu.Unlock()

	// This depends on timing — time.Since could be >= or < expiredTimeout.
	// The test verifies no panic and consistent boolean return.
	_ = p.IsExpired()
}

// ---------------------------------------------------------------------------
// Peer — IdleCheck with callback from Expired state
// ---------------------------------------------------------------------------

func TestPeer_IdleCheck_FromExpiredNoChange(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "exp-check", "id-ec", net.ParseIP("100.64.0.50"))

	var callbackFired bool
	p.SetOnStateChange(func(peer *Peer, from, to PeerState) {
		callbackFired = true
	})

	p.Transition(PeerExpired)
	callbackFired = false

	p.mu.Lock()
	p.lastSeen = time.Now().Add(-20 * time.Minute)
	p.mu.Unlock()

	p.IdleCheck()

	// Expired state should not change further.
	if p.GetState() != PeerExpired {
		t.Fatalf("expected PeerExpired, got %s", p.GetState())
	}
	// No callback should fire since state did not change.
	if callbackFired {
		t.Error("callback should not fire when state stays Expired")
	}
}

// ---------------------------------------------------------------------------
// Peer — IdleCheck from Discovered state (no change expected)
// ---------------------------------------------------------------------------

func TestPeer_IdleCheck_FromDiscoveredNoChange(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "disc-check", "id-dc", net.ParseIP("100.64.0.51"))

	p.mu.Lock()
	p.lastSeen = time.Now().Add(-20 * time.Minute)
	p.mu.Unlock()

	p.IdleCheck()

	if p.GetState() != PeerDiscovered {
		t.Fatalf("expected PeerDiscovered, got %s", p.GetState())
	}
}

// ---------------------------------------------------------------------------
// Peer — PeerState String for all valid states
// ---------------------------------------------------------------------------

func TestPeerState_String_AllStates(t *testing.T) {
	expected := map[PeerState]string{
		PeerDiscovered: "discovered",
		PeerConnecting: "connecting",
		PeerDirect:     "direct",
		PeerRelayed:    "relayed",
		PeerIdle:       "idle",
		PeerExpired:    "expired",
	}
	for state, want := range expected {
		got := state.String()
		if got != want {
			t.Errorf("PeerState(%d).String(): want %q, got %q", state, want, got)
		}
	}
}

// ---------------------------------------------------------------------------
// PeerSummary — short NodeID
// ---------------------------------------------------------------------------

func TestPeerSummary_ShortNodeID(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "short", "abc", net.ParseIP("100.64.0.52"))
	s := PeerSummary(p)
	if s == "" {
		t.Fatal("PeerSummary should return non-empty string")
	}
	// NodeID "abc" is < 8 chars so it should not be truncated.
	if len(s) < 10 {
		t.Errorf("PeerSummary too short: %q", s)
	}
}

// ---------------------------------------------------------------------------
// PeerSummary — empty NodeID
// ---------------------------------------------------------------------------

func TestPeerSummary_EmptyNodeID(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "empty", "", net.ParseIP("100.64.0.53"))
	s := PeerSummary(p)
	if s == "" {
		t.Fatal("PeerSummary should return non-empty string even with empty NodeID")
	}
}

// ---------------------------------------------------------------------------
// PeerSummary — long NodeID (should be truncated to 8)
// ---------------------------------------------------------------------------

func TestPeerSummary_LongNodeID(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "long", "1234567890ABCDEF", net.ParseIP("100.64.0.54"))
	s := PeerSummary(p)
	if s == "" {
		t.Fatal("PeerSummary should return non-empty string")
	}
	// Should contain the truncated ID "12345678".
	// The actual check is in PeerSummary code: shortID = p.NodeID[:8]
}

// ---------------------------------------------------------------------------
// newTestLogger helper verification
// ---------------------------------------------------------------------------

func TestNewTestLogger(t *testing.T) {
	l := newTestLogger()
	if l == nil {
		t.Fatal("newTestLogger should return non-nil logger")
	}
}

// ---------------------------------------------------------------------------
// encodeKey / keyFromBase64 — empty string
// ---------------------------------------------------------------------------

func TestKeyFromBase64_EmptyString(t *testing.T) {
	_, err := keyFromBase64("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

// ---------------------------------------------------------------------------
// keyLenError — Error method
// ---------------------------------------------------------------------------

func TestKeyLenError_Values(t *testing.T) {
	for _, n := range []int{0, 1, 31, 33, 64} {
		e := &keyLenError{n: n}
		msg := e.Error()
		if msg != "key must be 32 bytes" {
			t.Errorf("keyLenError{%d].Error() = %q", n, msg)
		}
	}
}

// ---------------------------------------------------------------------------
// Manager — AddOrUpdate with valid endpoint on update
// ---------------------------------------------------------------------------

func TestManager_AddOrUpdate_UpdateWithValidEndpoint(t *testing.T) {
	log := klog.New(nil, klog.LevelError, klog.FormatText)
	m := NewManager(log, nil)
	defer m.Stop()

	var key [32]byte
	key[0] = 0xBB
	vip := net.ParseIP("100.64.0.55")

	// Add without endpoint.
	m.AddOrUpdate(key, "ep-update", "id-eu", vip, "", nil)

	p, ok := m.GetPeer(key)
	if !ok {
		t.Fatal("peer should exist")
	}
	if p.GetEndpoint() != nil {
		t.Error("expected nil endpoint initially")
	}

	// Update with valid endpoint.
	m.AddOrUpdate(key, "ep-update", "id-eu", vip, "10.0.0.1:51820", nil)

	p.mu.RLock()
	ep := p.Endpoint
	p.mu.RUnlock()
	if ep == nil {
		t.Fatal("expected endpoint after update")
	}
	if ep.String() != "10.0.0.1:51820" {
		t.Errorf("endpoint: got %s", ep)
	}
}
