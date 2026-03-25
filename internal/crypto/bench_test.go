package crypto

import (
	"testing"
)

var (
	benchKey     [32]byte
	benchPayload = make([]byte, 1400) // typical MTU-sized packet
)

func init() {
	for i := range benchKey {
		benchKey[i] = byte(i)
	}
	for i := range benchPayload {
		benchPayload[i] = byte(i & 0xFF)
	}
}

// BenchmarkEncryptAEAD measures ChaCha20-Poly1305 encryption of a 1400-byte packet.
func BenchmarkEncryptAEAD(b *testing.B) {
	b.SetBytes(int64(len(benchPayload)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptAEAD(benchKey, uint64(i), benchPayload, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptAEAD measures ChaCha20-Poly1305 decryption of a 1400-byte packet.
func BenchmarkDecryptAEAD(b *testing.B) {
	ct, err := EncryptAEAD(benchKey, 0, benchPayload, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(benchPayload)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptAEAD(benchKey, 0, ct, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReplayWindow_Check measures Check+Advance for sequential counters.
func BenchmarkReplayWindow_Check(b *testing.B) {
	w := &ReplayWindow{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := uint64(i)
		if w.Check(counter) {
			w.Advance(counter)
		}
	}
}

// BenchmarkReplayWindow_CheckDuplicate measures the duplicate-rejection fast path.
func BenchmarkReplayWindow_CheckDuplicate(b *testing.B) {
	w := &ReplayWindow{}
	// Pre-fill with sequential counters.
	for i := uint64(0); i < 100; i++ {
		w.Advance(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Counter 50 is already in the window — should be rejected quickly.
		w.Check(50)
	}
}

// BenchmarkNoiseHandshake measures a complete Noise IK handshake round-trip.
func BenchmarkNoiseHandshake(b *testing.B) {
	ikp, err := GenerateKeyPair()
	if err != nil {
		b.Fatal(err)
	}
	rkp, err := GenerateKeyPair()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ini, err := InitiatorHandshake(ikp, rkp.Public)
		if err != nil {
			b.Fatal(err)
		}
		res, err := ResponderHandshake(rkp)
		if err != nil {
			b.Fatal(err)
		}

		msg1, err := ini.WriteMessage1()
		if err != nil {
			b.Fatal(err)
		}
		if err := res.ReadMessage1(msg1); err != nil {
			b.Fatal(err)
		}
		msg2, err := res.WriteMessage2()
		if err != nil {
			b.Fatal(err)
		}
		if err := ini.ReadMessage2(msg2); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateKeyPair measures X25519 key pair generation.
func BenchmarkGenerateKeyPair(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := GenerateKeyPair(); err != nil {
			b.Fatal(err)
		}
	}
}
