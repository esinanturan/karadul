package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Key is a 32-byte Curve25519 key.
type Key [32]byte

// KeyPair holds a Curve25519 private/public key pair.
type KeyPair struct {
	Private Key
	Public  Key
}

// GenerateKeyPair creates a new random Curve25519 key pair.
func GenerateKeyPair() (KeyPair, error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("generate x25519 key: %w", err)
	}
	var kp KeyPair
	copy(kp.Private[:], priv.Bytes())
	copy(kp.Public[:], priv.PublicKey().Bytes())
	return kp, nil
}

// ECDH performs a Diffie-Hellman exchange: returns shared secret.
func ECDH(priv, pub Key) (Key, error) {
	curve := ecdh.X25519()

	privKey, err := curve.NewPrivateKey(priv[:])
	if err != nil {
		return Key{}, fmt.Errorf("parse private key: %w", err)
	}
	pubKey, err := curve.NewPublicKey(pub[:])
	if err != nil {
		return Key{}, fmt.Errorf("parse public key: %w", err)
	}
	shared, err := privKey.ECDH(pubKey)
	if err != nil {
		return Key{}, fmt.Errorf("ecdh: %w", err)
	}
	var out Key
	copy(out[:], shared)
	return out, nil
}

// PublicFromPrivate derives the public key from a private key.
func PublicFromPrivate(priv Key) (Key, error) {
	curve := ecdh.X25519()
	privKey, err := curve.NewPrivateKey(priv[:])
	if err != nil {
		return Key{}, fmt.Errorf("parse private key: %w", err)
	}
	var pub Key
	copy(pub[:], privKey.PublicKey().Bytes())
	return pub, nil
}

// String returns the key as a base64-encoded string.
func (k Key) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

// Hex returns the key as a lowercase hex string.
func (k Key) Hex() string {
	return hex.EncodeToString(k[:])
}

// IsZero reports whether the key is all zeros.
func (k Key) IsZero() bool {
	for _, b := range k {
		if b != 0 {
			return false
		}
	}
	return true
}

// KeyFromBase64 parses a base64-encoded key.
func KeyFromBase64(s string) (Key, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return Key{}, fmt.Errorf("decode key: %w", err)
	}
	if len(b) != 32 {
		return Key{}, fmt.Errorf("key must be 32 bytes, got %d", len(b))
	}
	var k Key
	copy(k[:], b)
	return k, nil
}

// KeyFromHex parses a hex-encoded key.
func KeyFromHex(s string) (Key, error) {
	b, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return Key{}, fmt.Errorf("decode key hex: %w", err)
	}
	if len(b) != 32 {
		return Key{}, fmt.Errorf("key must be 32 bytes, got %d", len(b))
	}
	var k Key
	copy(k[:], b)
	return k, nil
}

// SaveKeyPair writes private and public keys to files under dir.
// Private key is in <dir>/private.key, public in <dir>/public.key.
func SaveKeyPair(kp KeyPair, dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	privPath := filepath.Join(dir, "private.key")
	pubPath := filepath.Join(dir, "public.key")

	if err := os.WriteFile(privPath, []byte(kp.Private.String()+"\n"), 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, []byte(kp.Public.String()+"\n"), 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	return nil
}

// LoadKeyPair reads key files from dir.
func LoadKeyPair(dir string) (KeyPair, error) {
	privData, err := os.ReadFile(filepath.Join(dir, "private.key"))
	if err != nil {
		return KeyPair{}, fmt.Errorf("read private key: %w", err)
	}
	priv, err := KeyFromBase64(strings.TrimSpace(string(privData)))
	if err != nil {
		return KeyPair{}, err
	}
	pub, err := PublicFromPrivate(priv)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{Private: priv, Public: pub}, nil
}
