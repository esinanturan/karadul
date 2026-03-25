package node

import (
	"bytes"
	"testing"
	"time"
)

func TestSession_EncryptDecrypt(t *testing.T) {
	var sendKey, recvKey [32]byte
	for i := range sendKey {
		sendKey[i] = byte(i)
		recvKey[i] = byte(i + 1)
	}

	alice := NewSession(sendKey, recvKey, nil)
	// Bob has mirrored keys: alice sends → bob receives.
	bob := NewSession(recvKey, sendKey, nil)

	msg := []byte("hello from alice")
	counter, ct, err := alice.Encrypt(msg)
	if err != nil {
		t.Fatal(err)
	}

	plain, err := bob.Decrypt(counter, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, msg) {
		t.Fatalf("decrypted: %q, want %q", plain, msg)
	}
}

func TestSession_ReplayProtection(t *testing.T) {
	var k [32]byte
	s := NewSession(k, k, nil)

	_, ct, err := s.Encrypt([]byte("pkt"))
	if err != nil {
		t.Fatal(err)
	}

	// First decrypt — should succeed.
	if _, err := s.Decrypt(0, ct); err != nil {
		t.Fatal(err)
	}

	// Second decrypt of same counter — should fail.
	if _, err := s.Decrypt(0, ct); err == nil {
		t.Fatal("replay should be rejected")
	}
}

func TestSession_CounterMonotonic(t *testing.T) {
	var k [32]byte
	s := NewSession(k, k, nil)

	var counters []uint64
	for i := 0; i < 5; i++ {
		c, _, _ := s.Encrypt([]byte("x"))
		counters = append(counters, c)
	}
	for i := 1; i < len(counters); i++ {
		if counters[i] <= counters[i-1] {
			t.Fatalf("counter not monotonic: %d then %d", counters[i-1], counters[i])
		}
	}
}

// TestSession_IsExpired_Fresh verifies a newly-created session is not expired.
func TestSession_IsExpired_Fresh(t *testing.T) {
	var k1, k2 [32]byte
	s := NewSession(k1, k2, nil)
	if s.IsExpired() {
		t.Error("fresh session should not be expired")
	}
}

// TestSession_IsExpired_Old verifies that a session past its lifetime+grace is expired.
func TestSession_IsExpired_Old(t *testing.T) {
	var k1, k2 [32]byte
	s := NewSession(k1, k2, nil)
	// Age the session beyond lifetime+grace.
	s.mu.Lock()
	s.createdAt = time.Now().Add(-(sessionLifetime + sessionGracePeriod + time.Second))
	s.mu.Unlock()
	if !s.IsExpired() {
		t.Error("aged session should be expired")
	}
}

// TestSession_NeedsRekey_Fresh verifies a fresh session does not need a rekey.
func TestSession_NeedsRekey_Fresh(t *testing.T) {
	var k1, k2 [32]byte
	s := NewSession(k1, k2, nil)
	if s.NeedsRekey() {
		t.Error("fresh session should not need rekey")
	}
}

// TestSession_NeedsRekey_Old verifies a session past its lifetime needs a rekey.
func TestSession_NeedsRekey_Old(t *testing.T) {
	var k1, k2 [32]byte
	s := NewSession(k1, k2, nil)
	s.mu.Lock()
	s.createdAt = time.Now().Add(-(sessionLifetime + time.Second))
	s.mu.Unlock()
	if !s.NeedsRekey() {
		t.Error("old session should need rekey")
	}
}

// TestSession_LastUsed verifies LastUsed advances after Encrypt/Decrypt.
func TestSession_LastUsed(t *testing.T) {
	var k1, k2 [32]byte
	k1[0], k2[0] = 0xAA, 0xBB
	s := NewSession(k1, k2, nil)

	before := time.Now()
	_, _, err := s.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	lu := s.LastUsed()
	if lu.Before(before) {
		t.Error("LastUsed should be >= time before Encrypt")
	}
}

// TestSession_Encrypt_TriggersRekey verifies the onRekey callback is invoked when
// the session is older than sessionLifetime (covers the rekey branch in Encrypt).
func TestSession_Encrypt_TriggersRekey(t *testing.T) {
	var k [32]byte
	rekey := make(chan struct{}, 1)
	s := NewSession(k, k, func() { rekey <- struct{}{} })

	// Age the session beyond sessionLifetime so the rekey branch fires.
	s.mu.Lock()
	s.createdAt = time.Now().Add(-(sessionLifetime + time.Second))
	s.mu.Unlock()

	if _, _, err := s.Encrypt([]byte("packet")); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	select {
	case <-rekey:
		// onRekey was called — good.
	case <-time.After(time.Second):
		t.Fatal("onRekey was not called after session lifetime exceeded")
	}
}

// TestSession_Decrypt_AuthFailure verifies Decrypt returns an error when the
// ciphertext authentication tag is wrong (DecryptAEAD error path).
func TestSession_Decrypt_AuthFailure(t *testing.T) {
	var k [32]byte
	s := NewSession(k, k, nil)

	// Corrupt ciphertext: supply random bytes that are too short to have a valid tag.
	badCT := make([]byte, 5) // too small; will fail AEAD auth
	if _, err := s.Decrypt(0, badCT); err == nil {
		t.Fatal("expected authentication error for corrupted ciphertext")
	}
}

func TestSession_Rotate(t *testing.T) {
	var k1, k2, k3, k4 [32]byte
	k1[0], k2[0], k3[0], k4[0] = 1, 2, 3, 4

	alice := NewSession(k1, k2, nil)
	bob := NewSession(k2, k1, nil)

	// Send one packet under old keys.
	c, ct, _ := alice.Encrypt([]byte("before rotate"))
	plain, err := bob.Decrypt(c, ct)
	if err != nil || string(plain) != "before rotate" {
		t.Fatal("pre-rotate session failed")
	}

	// Rotate both sides to new keys.
	alice.Rotate(k3, k4)
	bob.Rotate(k4, k3)

	// Old-key ciphertext should now fail on bob (counter was reset).
	// But a fresh packet under new keys should work.
	c2, ct2, _ := alice.Encrypt([]byte("after rotate"))
	plain2, err := bob.Decrypt(c2, ct2)
	if err != nil {
		t.Fatalf("post-rotate decrypt: %v", err)
	}
	if string(plain2) != "after rotate" {
		t.Fatalf("wrong plaintext: %q", plain2)
	}
}
