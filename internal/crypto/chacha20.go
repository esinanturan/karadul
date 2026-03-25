package crypto

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptAEAD encrypts plaintext with ChaCha20-Poly1305.
// nonce is a 64-bit counter encoded as little-endian in the 12-byte nonce (bytes 4–11).
// aad is additional authenticated data (may be nil).
// Returns ciphertext || 16-byte auth tag.
func EncryptAEAD(key [32]byte, nonce uint64, plaintext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("chacha20poly1305.New: %w", err)
	}
	n := makeNonce12(nonce)
	return aead.Seal(nil, n[:], plaintext, aad), nil
}

// DecryptAEAD decrypts and authenticates ciphertext (including 16-byte tag).
func DecryptAEAD(key [32]byte, nonce uint64, ciphertext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("chacha20poly1305.New: %w", err)
	}
	n := makeNonce12(nonce)
	plain, err := aead.Open(nil, n[:], ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("chacha20poly1305 decrypt: %w", err)
	}
	return plain, nil
}

// EncryptZeroNonce encrypts with a zero nonce (used during Noise handshake).
func EncryptZeroNonce(key [32]byte, plaintext, aad []byte) ([]byte, error) {
	return EncryptAEAD(key, 0, plaintext, aad)
}

// DecryptZeroNonce decrypts with a zero nonce.
func DecryptZeroNonce(key [32]byte, ciphertext, aad []byte) ([]byte, error) {
	return DecryptAEAD(key, 0, ciphertext, aad)
}

// makeNonce12 converts a uint64 counter into a 12-byte nonce.
// Layout: 4 zero bytes || 8-byte little-endian counter (matches WireGuard convention).
func makeNonce12(counter uint64) [12]byte {
	var n [12]byte
	binary.LittleEndian.PutUint64(n[4:], counter)
	return n
}
