package crypto

import (
	"testing"
)

func TestReplayWindow_Basic(t *testing.T) {
	var w ReplayWindow

	// Fresh window accepts any counter.
	if !w.Check(0) {
		t.Fatal("counter 0 should be accepted on fresh window")
	}
	w.Advance(0)
	// Replay of counter 0 should be rejected.
	if w.Check(0) {
		t.Fatal("counter 0 should be rejected after advance")
	}

	// New counter accepted.
	if !w.Check(1) {
		t.Fatal("counter 1 should be accepted")
	}
	w.Advance(1)
}

func TestReplayWindow_OldPacket(t *testing.T) {
	var w ReplayWindow

	// Advance far ahead.
	w.Advance(WindowSize + 10)

	// Counter 0 should be rejected (behind window).
	if w.Check(0) {
		t.Fatal("counter 0 should be rejected (too old)")
	}
}

func TestReplayWindow_FarAhead(t *testing.T) {
	var w ReplayWindow

	// A counter far ahead of the window should be accepted.
	if !w.Check(WindowSize * 3) {
		t.Fatal("counter far ahead should be accepted")
	}
}

func TestReplayWindow_Slide(t *testing.T) {
	var w ReplayWindow

	// Fill a contiguous range.
	for i := uint64(0); i < 100; i++ {
		if !w.Check(i) {
			t.Fatalf("counter %d should be accepted", i)
		}
		w.Advance(i)
	}

	// All 0..99 should be replays.
	for i := uint64(0); i < 100; i++ {
		if w.Check(i) {
			t.Fatalf("counter %d should be rejected after advance", i)
		}
	}

	// Counter 100 should be accepted.
	if !w.Check(100) {
		t.Fatal("counter 100 should be accepted")
	}
}

func TestReplayWindow_OutOfOrder(t *testing.T) {
	var w ReplayWindow

	// Accept and advance 10 first.
	if !w.Check(10) {
		t.Fatal("expected accept 10")
	}
	w.Advance(10)

	// Accept counter 5 (within window, not yet seen).
	if !w.Check(5) {
		t.Fatal("expected accept 5 (out of order but within window)")
	}
	w.Advance(5)

	// Replay of 5.
	if w.Check(5) {
		t.Fatal("counter 5 should be rejected")
	}
}

func TestReplayWindow_Reset(t *testing.T) {
	var w ReplayWindow

	// Advance the window significantly.
	for i := uint64(0); i < 100; i++ {
		w.Advance(i)
	}

	// Verify counter 0 is rejected before reset.
	if w.Check(0) {
		t.Fatal("counter 0 should be rejected before reset")
	}

	w.Reset()

	// After reset, counter 0 should be accepted again.
	if !w.Check(0) {
		t.Fatal("counter 0 should be accepted after reset")
	}
	// Floor should be back to 0 (counter WindowSize*3 accepted from fresh window).
	if !w.Check(WindowSize * 3) {
		t.Fatal("far-ahead counter should be accepted on reset window")
	}
}

// TestReplayWindow_Advance_BehindFloor verifies that Advance with a counter below
// the current floor is silently ignored (no panic, no change).
func TestReplayWindow_Advance_BehindFloor(t *testing.T) {
	var w ReplayWindow
	// Slide the window well past 0.
	w.Advance(WindowSize + 100)
	// floor is now > 0; advancing with counter 0 should be a no-op.
	w.Advance(0) // must not panic
	// Counter 0 should still be rejected (behind floor).
	if w.Check(0) {
		t.Fatal("counter behind floor should still be rejected after no-op Advance")
	}
}

func TestReplayWindow_Reset_ClearsMarked(t *testing.T) {
	var w ReplayWindow

	w.Advance(42)
	if w.Check(42) {
		t.Fatal("counter 42 should be rejected after Advance")
	}

	w.Reset()

	// After reset, 42 should be acceptable again.
	if !w.Check(42) {
		t.Fatal("counter 42 should be accepted after Reset")
	}
}
