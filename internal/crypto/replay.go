package crypto

import "sync"

const (
	// WindowSize is the number of packets tracked in the sliding window.
	// Must be a multiple of 64.
	WindowSize = 2048
	windowWords = WindowSize / 64
)

// ReplayWindow is a thread-safe sliding window replay filter.
// It tracks which counter values have been seen and rejects duplicates
// or packets that fall behind the window floor.
type ReplayWindow struct {
	mu      sync.Mutex
	bitmap  [windowWords]uint64
	floor   uint64 // lowest counter still inside the window
}

// Check returns true if counter is acceptable (not a replay, not too old).
// It does NOT advance the window; call Advance after authenticating the packet.
func (w *ReplayWindow) Check(counter uint64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if counter < w.floor {
		return false // too old
	}
	// Use subtraction to avoid overflow when floor is near MaxUint64.
	if counter-w.floor >= WindowSize {
		return true // ahead of window — will be accepted when advanced
	}
	idx := (counter % WindowSize) / 64
	bit := counter % 64
	return w.bitmap[idx]&(1<<bit) == 0
}

// Advance marks counter as seen and advances the window if necessary.
// Should be called only after the packet has been authenticated.
func (w *ReplayWindow) Advance(counter uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// If counter is behind the floor, it was a replay — ignore.
	if counter < w.floor {
		return
	}

	// If counter is far ahead, slide the window.
	// Use subtraction to avoid overflow when floor is near MaxUint64.
	if counter-w.floor >= WindowSize {
		advance := counter - w.floor - WindowSize + 1
		w.slideBy(advance)
	}

	// Mark counter as seen.
	idx := (counter % WindowSize) / 64
	bit := counter % 64
	w.bitmap[idx] |= 1 << bit
}

// slideBy advances the window floor by n positions, zeroing vacated bits.
func (w *ReplayWindow) slideBy(n uint64) {
	if n >= WindowSize {
		// Clear entire bitmap.
		w.bitmap = [windowWords]uint64{}
		w.floor += n
		return
	}

	wordAdvance := n / 64
	bitAdvance := n % 64

	// Shift words.
	if wordAdvance > 0 {
		copy(w.bitmap[:], w.bitmap[wordAdvance:])
		for i := windowWords - int(wordAdvance); i < windowWords; i++ {
			w.bitmap[i] = 0
		}
	}

	// Shift bits within words.
	if bitAdvance > 0 {
		var carry uint64
		for i := windowWords - 1; i >= 0; i-- {
			newCarry := w.bitmap[i] >> (64 - bitAdvance)
			w.bitmap[i] = (w.bitmap[i] << bitAdvance) | carry
			carry = newCarry
		}
	}

	w.floor += n
}

// Reset clears the window and resets the floor counter.
func (w *ReplayWindow) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bitmap = [windowWords]uint64{}
	w.floor = 0
}
