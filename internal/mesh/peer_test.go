package mesh

import (
	"net"
	"testing"
	"time"
)

func TestPeer_Transitions(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "test-node", "id-1", net.ParseIP("100.64.0.1"))

	if p.GetState() != PeerDiscovered {
		t.Fatalf("initial state: got %s, want discovered", p.GetState())
	}

	var transitions []PeerState
	p.SetOnStateChange(func(peer *Peer, from, to PeerState) {
		transitions = append(transitions, to)
	})

	p.Transition(PeerConnecting)
	p.Transition(PeerDirect)
	p.Transition(PeerIdle)
	p.Transition(PeerExpired)

	if len(transitions) != 4 {
		t.Fatalf("expected 4 transitions, got %d", len(transitions))
	}
	if transitions[3] != PeerExpired {
		t.Fatalf("last transition should be expired, got %s", transitions[3])
	}
}

func TestPeer_IdleCheck(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "test", "id", net.ParseIP("100.64.0.2"))
	p.Transition(PeerDirect)

	// Manually set lastSeen far in the past.
	p.mu.Lock()
	p.lastSeen = time.Now().Add(-6 * time.Minute)
	p.mu.Unlock()

	p.IdleCheck()
	if p.GetState() != PeerIdle {
		t.Fatalf("expected idle after 6 min, got %s", p.GetState())
	}
}

func TestPeer_Touch(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "test", "id", net.ParseIP("100.64.0.3"))
	p.Transition(PeerDirect)
	p.Transition(PeerIdle)

	// Touch should bring it back to direct.
	p.Touch()
	if p.GetState() != PeerDirect {
		t.Fatalf("touch should bring idle peer to direct, got %s", p.GetState())
	}
}

// TestPeerState_String verifies all named states and the default case.
func TestPeerState_String(t *testing.T) {
	cases := []struct {
		state PeerState
		want  string
	}{
		{PeerDiscovered, "discovered"},
		{PeerConnecting, "connecting"},
		{PeerDirect, "direct"},
		{PeerRelayed, "relayed"},
		{PeerIdle, "idle"},
		{PeerExpired, "expired"},
		{PeerState(99), "state(99)"},
	}
	for _, tc := range cases {
		got := tc.state.String()
		if got != tc.want {
			t.Errorf("PeerState(%d).String(): want %q, got %q", int(tc.state), tc.want, got)
		}
	}
}

// TestPeer_IdleCheck_RelayedToIdle verifies that a PeerRelayed peer with a stale
// lastSeen transitions to PeerIdle via IdleCheck.
func TestPeer_IdleCheck_RelayedToIdle(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "relay-node", "id-r", net.ParseIP("100.64.0.10"))
	p.Transition(PeerRelayed)

	// Simulate 6 minutes without traffic (> idleTimeout of 5 min).
	p.mu.Lock()
	p.lastSeen = time.Now().Add(-6 * time.Minute)
	p.mu.Unlock()

	p.IdleCheck()
	if p.GetState() != PeerIdle {
		t.Fatalf("expected PeerIdle after 6 min relayed, got %s", p.GetState())
	}
}

// TestPeer_IdleCheck_Expired verifies that a PeerIdle peer with lastSeen older
// than expiredTimeout (10 min) transitions to PeerExpired via IdleCheck.
func TestPeer_IdleCheck_Expired(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "idle-node", "id-e", net.ParseIP("100.64.0.11"))
	p.Transition(PeerDirect)
	p.Transition(PeerIdle)

	// Simulate 11 minutes without traffic (> expiredTimeout of 10 min).
	p.mu.Lock()
	p.lastSeen = time.Now().Add(-11 * time.Minute)
	p.mu.Unlock()

	p.IdleCheck()
	if p.GetState() != PeerExpired {
		t.Fatalf("expected PeerExpired after 11 min idle, got %s", p.GetState())
	}
}

func TestPeer_SetGetEndpoint(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "test", "id", net.ParseIP("100.64.0.4"))

	if p.GetEndpoint() != nil {
		t.Fatal("initial endpoint should be nil")
	}

	ep := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 51820}
	p.SetEndpoint(ep)

	got := p.GetEndpoint()
	if got == nil {
		t.Fatal("endpoint is nil after SetEndpoint")
	}
	if got.String() != ep.String() {
		t.Fatalf("endpoint mismatch: %s != %s", got, ep)
	}
	// Verify it's a copy (not the same pointer).
	if got == ep {
		t.Fatal("GetEndpoint should return a copy, not the original pointer")
	}
}

// TestPeer_TransitionSameState verifies that transitioning a peer to its
// current state does NOT invoke the onStateChange callback.
func TestPeer_TransitionSameState(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "same-state", "id-ss", net.ParseIP("100.64.0.20"))

	callbackCount := 0
	p.SetOnStateChange(func(peer *Peer, from, to PeerState) {
		callbackCount++
	})

	// First transition: Discovered -> Direct.
	p.Transition(PeerDirect)
	if callbackCount != 1 {
		t.Fatalf("expected 1 callback after first Transition, got %d", callbackCount)
	}

	// Second transition: Direct -> Direct (same state). Callback should NOT fire.
	p.Transition(PeerDirect)
	if callbackCount != 1 {
		t.Fatalf("expected callback count to remain 1 after same-state transition, got %d", callbackCount)
	}

	if p.GetState() != PeerDirect {
		t.Fatalf("expected PeerDirect, got %s", p.GetState())
	}
}

// TestPeer_TransitionSameStateNoCallback verifies that transitioning a peer to
// the state it is already in does NOT invoke the onStateChange callback.
// This is a focused variant of TestPeer_TransitionSameState that also inspects
// the (from, to) arguments passed to the callback.
func TestPeer_TransitionSameStateNoCallback(t *testing.T) {
	var key [32]byte
	p := NewPeer(key, "no-cb-node", "id-nc", net.ParseIP("100.64.0.30"))

	type transition struct{ from, to PeerState }
	var seen []transition
	p.SetOnStateChange(func(peer *Peer, from, to PeerState) {
		seen = append(seen, transition{from, to})
	})

	// First: Discovered -> Direct. Callback MUST fire.
	p.Transition(PeerDirect)
	if len(seen) != 1 {
		t.Fatalf("expected 1 callback after first Transition, got %d", len(seen))
	}
	if seen[0].from != PeerDiscovered || seen[0].to != PeerDirect {
		t.Fatalf("unexpected transition: from=%s to=%s", seen[0].from, seen[0].to)
	}

	// Second: Direct -> Direct. Callback must NOT fire.
	p.Transition(PeerDirect)
	if len(seen) != 1 {
		t.Fatalf("callback must not fire on same-state transition; got %d calls", len(seen))
	}
}
