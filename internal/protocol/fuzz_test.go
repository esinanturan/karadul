package protocol

import (
	"testing"
)

// FuzzUnmarshalHandshakeInit fuzzes the HandshakeInit wire parser.
// The goal is to ensure no input can cause a panic or out-of-bounds access.
func FuzzUnmarshalHandshakeInit(f *testing.F) {
	// Seed: a valid-sized but zeroed message.
	f.Add(make([]byte, HandshakeInitSize))

	// Seed: correct type byte, rest zero.
	valid := make([]byte, HandshakeInitSize)
	valid[0] = TypeHandshakeInit
	f.Add(valid)

	// Seed: too-short input.
	f.Add([]byte{TypeHandshakeInit, 0x00, 0x00})

	// Seed: empty input.
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, b []byte) {
		// Must never panic regardless of input.
		msg, err := UnmarshalMsgHandshakeInit(b)
		if err != nil {
			return
		}
		// If parsing succeeded, re-encode and verify size.
		wire := msg.MarshalBinary()
		if len(wire) != HandshakeInitSize {
			t.Fatalf("re-encoded length %d != %d", len(wire), HandshakeInitSize)
		}
	})
}

// FuzzUnmarshalHandshakeResp fuzzes the HandshakeResp wire parser.
func FuzzUnmarshalHandshakeResp(f *testing.F) {
	f.Add(make([]byte, HandshakeRespSize))

	valid := make([]byte, HandshakeRespSize)
	valid[0] = TypeHandshakeResp
	f.Add(valid)

	f.Add([]byte{})
	f.Add([]byte{TypeHandshakeResp})

	f.Fuzz(func(t *testing.T, b []byte) {
		msg, err := UnmarshalMsgHandshakeResp(b)
		if err != nil {
			return
		}
		wire := msg.MarshalBinary()
		if len(wire) != HandshakeRespSize {
			t.Fatalf("re-encoded length %d != %d", len(wire), HandshakeRespSize)
		}
	})
}

// FuzzUnmarshalMsgData fuzzes the data message parser.
func FuzzUnmarshalMsgData(f *testing.F) {
	// Minimal valid data message: header + 16-byte auth tag.
	seed := make([]byte, DataHeaderSize+AuthTagSize)
	seed[0] = TypeData
	f.Add(seed)

	f.Add([]byte{})
	f.Add([]byte{TypeData})
	f.Add(make([]byte, DataHeaderSize)) // header only, no auth tag

	f.Fuzz(func(t *testing.T, b []byte) {
		msg, err := UnmarshalMsgData(b)
		if err != nil {
			return
		}
		wire := msg.MarshalBinary()
		if len(wire) < DataHeaderSize {
			t.Fatalf("re-encoded too short: %d", len(wire))
		}
	})
}

// FuzzParseType fuzzes the packet type parser.
func FuzzParseType(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{TypeHandshakeInit})
	f.Add([]byte{TypeHandshakeResp})
	f.Add([]byte{TypeData})
	f.Add([]byte{TypeKeepalive})
	f.Add([]byte{0xFF})

	f.Fuzz(func(t *testing.T, b []byte) {
		// Must never panic.
		_, _ = ParseType(b)
	})
}
