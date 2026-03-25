package crypto

import (
	"testing"
)

// FuzzDecryptAEAD ensures that DecryptAEAD never panics on arbitrary input.
func FuzzDecryptAEAD(f *testing.F) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	pt := []byte("hello karadul fuzzer")
	var k32 [32]byte
	copy(k32[:], key)
	ct, _ := EncryptAEAD(k32, 0, pt, nil)

	f.Add(key, uint64(0), ct, []byte(nil))
	f.Add(key, uint64(1), ct, []byte("aad"))
	f.Add(key, uint64(0), []byte("short"), []byte(nil))
	f.Add(key, uint64(0), []byte{}, []byte(nil))

	f.Fuzz(func(t *testing.T, keyBytes []byte, nonce uint64, ciphertext, aad []byte) {
		if len(keyBytes) == 0 {
			return
		}
		var k [32]byte
		copy(k[:], keyBytes)

		plain, err := DecryptAEAD(k, nonce, ciphertext, aad)
		if err != nil {
			return
		}
		// Re-encrypt must reproduce same-length ciphertext.
		reenc, err := EncryptAEAD(k, nonce, plain, aad)
		if err != nil {
			t.Fatalf("re-encrypt failed: %v", err)
		}
		if len(reenc) != len(ciphertext) {
			t.Fatalf("length mismatch after re-encrypt: %d vs %d", len(reenc), len(ciphertext))
		}
	})
}

// FuzzEncryptDecryptRoundtrip ensures Encrypt→Decrypt is an identity for any plaintext.
func FuzzEncryptDecryptRoundtrip(f *testing.F) {
	f.Add(uint64(0), []byte("hello"))
	f.Add(uint64(1<<32), []byte{})
	f.Add(^uint64(0), make([]byte, 256))

	f.Fuzz(func(t *testing.T, nonce uint64, plaintext []byte) {
		var k [32]byte
		ct, err := EncryptAEAD(k, nonce, plaintext, nil)
		if err != nil {
			t.Skip()
		}
		got, err := DecryptAEAD(k, nonce, ct, nil)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if string(got) != string(plaintext) {
			t.Fatal("plaintext mismatch")
		}
	})
}

// FuzzReplayWindowCheck ensures ReplayWindow never panics.
func FuzzReplayWindowCheck(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(WindowSize - 1))
	f.Add(uint64(WindowSize))
	f.Add(^uint64(0))

	f.Fuzz(func(t *testing.T, counter uint64) {
		w := &ReplayWindow{}
		accepted := w.Check(counter)
		if accepted {
			w.Advance(counter)
			if w.Check(counter) {
				t.Fatalf("duplicate counter %d accepted", counter)
			}
		}
	})
}
