package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("hello, karadul!")
	aad := []byte("additional-data")

	ct, err := EncryptAEAD(key, 0, plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct[:len(plaintext)], plaintext) {
		t.Fatal("ciphertext looks like plaintext")
	}

	pt, err := DecryptAEAD(key, 0, ct, aad)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("decrypted: %q, want: %q", pt, plaintext)
	}
}

func TestEncryptCounterChangesOutput(t *testing.T) {
	var key [32]byte
	plaintext := []byte("test packet")

	ct0, _ := EncryptAEAD(key, 0, plaintext, nil)
	ct1, _ := EncryptAEAD(key, 1, plaintext, nil)
	if bytes.Equal(ct0, ct1) {
		t.Fatal("different counters produced same ciphertext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	var key, badKey [32]byte
	key[0] = 1
	badKey[0] = 2

	ct, _ := EncryptAEAD(key, 0, []byte("secret"), nil)
	_, err := DecryptAEAD(badKey, 0, ct, nil)
	if err == nil {
		t.Fatal("decryption with wrong key should fail")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	var key [32]byte
	ct, _ := EncryptAEAD(key, 0, []byte("secret"), nil)
	ct[0] ^= 0xFF // tamper
	_, err := DecryptAEAD(key, 0, ct, nil)
	if err == nil {
		t.Fatal("tampered ciphertext should fail authentication")
	}
}
