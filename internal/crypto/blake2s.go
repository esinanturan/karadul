package crypto

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/blake2s"
)

// Hash computes a BLAKE2s-256 hash of data.
func Hash(data []byte) [32]byte {
	return blake2s.Sum256(data)
}

// HashMany computes BLAKE2s-256 over the concatenation of all inputs.
func HashMany(inputs ...[]byte) [32]byte {
	h, err := blake2s.New256(nil)
	if err != nil {
		panic(fmt.Sprintf("blake2s.New256: %v", err))
	}
	for _, b := range inputs {
		h.Write(b)
	}
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// HMAC computes BLAKE2s-keyed MAC: BLAKE2s(key, data).
// key must be 1–32 bytes.
func HMAC(key, data []byte) [32]byte {
	h, err := blake2s.New256(key)
	if err != nil {
		// Fallback: hash key to 32 bytes and retry
		kh := blake2s.Sum256(key)
		h, err = blake2s.New256(kh[:])
		if err != nil {
			panic(fmt.Sprintf("blake2s HMAC: %v", err))
		}
	}
	h.Write(data)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// HKDF derives a 32-byte key from secret, salt and info using BLAKE2s as PRF.
// This is a simplified Extract+Expand without the counter-based iteration
// required by full HKDF — suitable for single 32-byte output.
func HKDF(secret, salt, info []byte) [32]byte {
	// Extract: prk = HMAC(salt, secret)
	var k []byte
	if len(salt) == 0 {
		k = make([]byte, 32) // zero salt
	} else {
		k = salt
	}
	prk := HMAC(k, secret)

	// Expand: T(1) = HMAC(prk, info || 0x01)
	infoPlus := make([]byte, len(info)+1)
	copy(infoPlus, info)
	infoPlus[len(info)] = 0x01

	return HMAC(prk[:], infoPlus)
}

// MixKey mixes input into the chaining key using HKDF-style extract/expand.
// Returns (new chaining key, new temp key).
func MixKey(ck [32]byte, input []byte) ([32]byte, [32]byte) {
	// k1 = HMAC(ck, input || 0x01)
	// k2 = HMAC(ck, k1    || 0x02)
	k1 := HMAC(ck[:], appendByte(input, 0x01))
	k2 := HMAC(ck[:], appendByte(k1[:], 0x02))
	return k1, k2
}

// MixHash mixes data into the handshake hash state.
func MixHash(h [32]byte, data []byte) [32]byte {
	return HashMany(h[:], data)
}

// Split derives two 32-byte transport keys from the chaining key.
func Split(ck [32]byte) (k1, k2 [32]byte) {
	empty := []byte{}
	k1 = HMAC(ck[:], appendByte(empty, 0x01))
	k2 = HMAC(ck[:], appendByte(k1[:], 0x02))
	return
}

// Counter64 encodes n as little-endian 8 bytes.
func Counter64(n uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	return b
}

func appendByte(src []byte, b byte) []byte {
	out := make([]byte, len(src)+1)
	copy(out, src)
	out[len(src)] = b
	return out
}
