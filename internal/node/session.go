package node

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ersinkoc/karadul/internal/crypto"
)

const (
	// sessionLifetime is how long a session key is valid before re-keying.
	sessionLifetime = 2 * time.Minute

	// sessionGracePeriod is the extra time old keys are accepted after rotation.
	sessionGracePeriod = 30 * time.Second
)

// Session represents an established encrypted channel to a single peer.
// Each session has independent send/recv keys derived from the Noise handshake.
type Session struct {
	mu sync.RWMutex

	sendKey [32]byte
	recvKey [32]byte

	sendCounter atomic.Uint64
	replayWin   crypto.ReplayWindow

	createdAt time.Time
	lastUsed  time.Time

	// onRekey is called when the session needs a new handshake.
	onRekey func()
}

// NewSession creates a Session from transport keys produced by a Noise handshake.
// sendKey is used for encryption; recvKey for decryption.
func NewSession(sendKey, recvKey [32]byte, onRekey func()) *Session {
	s := &Session{
		sendKey:   sendKey,
		recvKey:   recvKey,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		onRekey:   onRekey,
	}
	s.replayWin.Reset()
	return s
}

// Encrypt encrypts a plaintext IP packet for transmission.
// Returns the counter prepended ciphertext (counter || ciphertext || tag).
// The counter is advanced atomically.
func (s *Session) Encrypt(packet []byte) (counter uint64, ciphertext []byte, err error) {
	s.mu.RLock()
	sendKey := s.sendKey
	s.mu.RUnlock()

	counter = s.sendCounter.Add(1) - 1 // post-increment, get pre-value
	ct, err := crypto.EncryptAEAD(sendKey, counter, packet, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("session encrypt: %w", err)
	}

	s.mu.Lock()
	s.lastUsed = time.Now()
	s.mu.Unlock()

	// Trigger re-key if lifetime exceeded.
	if time.Since(s.createdAt) > sessionLifetime && s.onRekey != nil {
		go s.onRekey()
	}

	return counter, ct, nil
}

// Decrypt decrypts and authenticates a received packet.
func (s *Session) Decrypt(counter uint64, ciphertext []byte) ([]byte, error) {
	if !s.replayWin.Check(counter) {
		return nil, fmt.Errorf("replay detected or packet too old: counter %d", counter)
	}

	s.mu.RLock()
	recvKey := s.recvKey
	s.mu.RUnlock()

	plain, err := crypto.DecryptAEAD(recvKey, counter, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("session decrypt: %w", err)
	}

	s.replayWin.Advance(counter)

	s.mu.Lock()
	s.lastUsed = time.Now()
	s.mu.Unlock()

	return plain, nil
}

// Rotate atomically replaces the session keys (after a new handshake).
func (s *Session) Rotate(newSendKey, newRecvKey [32]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendKey = newSendKey
	s.recvKey = newRecvKey
	s.createdAt = time.Now()
	s.sendCounter.Store(0)
	s.replayWin.Reset()
}

// IsExpired reports whether the session has exceeded its lifetime plus grace period.
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.createdAt) > sessionLifetime+sessionGracePeriod
}

// NeedsRekey reports whether the session should be rekeyed soon.
func (s *Session) NeedsRekey() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.createdAt) > sessionLifetime
}

// LastUsed returns the time the session last sent or received a packet.
func (s *Session) LastUsed() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUsed
}
