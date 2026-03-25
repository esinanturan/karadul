package crypto

import (
	"encoding/binary"
	"testing"
)

// TestHash_Deterministic verifies the same input always produces the same hash.
func TestHash_Deterministic(t *testing.T) {
	a := Hash([]byte("hello"))
	b := Hash([]byte("hello"))
	if a != b {
		t.Fatal("Hash is not deterministic")
	}
}

// TestHash_Differs verifies different inputs produce different hashes.
func TestHash_Differs(t *testing.T) {
	a := Hash([]byte("hello"))
	b := Hash([]byte("world"))
	if a == b {
		t.Fatal("Hash collision for different inputs")
	}
}

// TestHMAC_Deterministic verifies HMAC is deterministic.
func TestHMAC_Deterministic(t *testing.T) {
	key := []byte("secret")
	data := []byte("message")
	a := HMAC(key, data)
	b := HMAC(key, data)
	if a != b {
		t.Fatal("HMAC is not deterministic")
	}
}

// TestHMAC_KeySensitive verifies different keys produce different MACs.
func TestHMAC_KeySensitive(t *testing.T) {
	a := HMAC([]byte("key1"), []byte("data"))
	b := HMAC([]byte("key2"), []byte("data"))
	if a == b {
		t.Fatal("HMAC should differ for different keys")
	}
}

// TestHKDF_Deterministic verifies HKDF always returns the same key for the same inputs.
func TestHKDF_Deterministic(t *testing.T) {
	secret := []byte("my-secret")
	salt := []byte("my-salt")
	info := []byte("my-info")

	k1 := HKDF(secret, salt, info)
	k2 := HKDF(secret, salt, info)
	if k1 != k2 {
		t.Fatal("HKDF is not deterministic")
	}
}

// TestHKDF_DifferentInputs verifies distinct inputs produce distinct outputs.
func TestHKDF_DifferentInputs(t *testing.T) {
	k1 := HKDF([]byte("secret1"), []byte("salt"), []byte("info"))
	k2 := HKDF([]byte("secret2"), []byte("salt"), []byte("info"))
	if k1 == k2 {
		t.Fatal("HKDF should differ for different secrets")
	}

	k3 := HKDF([]byte("secret"), []byte("salt1"), []byte("info"))
	k4 := HKDF([]byte("secret"), []byte("salt2"), []byte("info"))
	if k3 == k4 {
		t.Fatal("HKDF should differ for different salts")
	}

	k5 := HKDF([]byte("secret"), []byte("salt"), []byte("info1"))
	k6 := HKDF([]byte("secret"), []byte("salt"), []byte("info2"))
	if k5 == k6 {
		t.Fatal("HKDF should differ for different info strings")
	}
}

// TestHKDF_EmptySalt verifies HKDF works with an empty salt (uses zero-key internally).
func TestHKDF_EmptySalt(t *testing.T) {
	k := HKDF([]byte("secret"), nil, []byte("info"))
	var zero [32]byte
	if k == zero {
		t.Fatal("HKDF with empty salt should not produce zero key")
	}
}

// TestHKDF_NonZeroOutput verifies the output is non-zero for typical inputs.
func TestHKDF_NonZeroOutput(t *testing.T) {
	k := HKDF([]byte("secret"), []byte("salt"), []byte("info"))
	var zero [32]byte
	if k == zero {
		t.Fatal("HKDF should produce non-zero output")
	}
}

// TestCounter64_Encoding verifies Counter64 encodes values as little-endian 8 bytes.
func TestCounter64_Encoding(t *testing.T) {
	cases := []uint64{0, 1, 255, 256, 0xDEADBEEF, ^uint64(0)}
	for _, n := range cases {
		b := Counter64(n)
		if len(b) != 8 {
			t.Fatalf("Counter64(%d): want 8 bytes, got %d", n, len(b))
		}
		got := binary.LittleEndian.Uint64(b)
		if got != n {
			t.Fatalf("Counter64(%d): round-trip got %d", n, got)
		}
	}
}

// TestHMAC_LongKey verifies the fallback path when key exceeds 32 bytes.
// BLAKE2s rejects keys > 32 bytes; HMAC falls back to hashing the key first.
func TestHMAC_LongKey(t *testing.T) {
	longKey := make([]byte, 64) // 64 bytes > BLAKE2s max key size of 32
	for i := range longKey {
		longKey[i] = byte(i)
	}
	data := []byte("test data")
	// Should not panic; fallback path hashes the key first.
	result := HMAC(longKey, data)
	var zero [32]byte
	if result == zero {
		t.Fatal("HMAC with long key should produce non-zero result")
	}
	// Deterministic.
	result2 := HMAC(longKey, data)
	if result != result2 {
		t.Fatal("HMAC with long key should be deterministic")
	}
}

// TestMixKey_Deterministic verifies MixKey is deterministic.
func TestMixKey_Deterministic(t *testing.T) {
	var ck [32]byte
	ck[0] = 0xAB
	input := []byte("test-input")
	k1a, k1b := MixKey(ck, input)
	k2a, k2b := MixKey(ck, input)
	if k1a != k2a || k1b != k2b {
		t.Fatal("MixKey is not deterministic")
	}
}

// TestSplit_Deterministic verifies Split is deterministic and the two keys differ.
func TestSplit_Deterministic(t *testing.T) {
	var ck [32]byte
	ck[1] = 0x7F
	k1a, k1b := Split(ck)
	k2a, k2b := Split(ck)
	if k1a != k2a || k1b != k2b {
		t.Fatal("Split is not deterministic")
	}
	if k1a == k1b {
		t.Fatal("Split should produce two different keys")
	}
}
