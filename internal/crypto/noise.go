// Package crypto implements the Noise IK handshake pattern used by WireGuard.
//
// Pattern: Noise_IK_25519_ChaChaPoly_BLAKE2s
//
// Message layout:
//
//	-> e, es, s, ss   (initiator sends message 1: 32+32+16+16 = 96 bytes + payload)
//	<- e, ee, se      (responder sends message 2: 32+16 = 48 bytes + payload)
package crypto

import (
	"fmt"
)

const (
	// NoiseProtocolName is used to initialise the hash/chaining key.
	NoiseProtocolName = "Noise_IK_25519_ChaChaPoly_BLAKE2s"
	// NoisePrologue is the karadul-specific prologue.
	NoisePrologue = "karadul-v1"
)

// HandshakeState holds the state of an in-progress Noise IK handshake.
type HandshakeState struct {
	isInitiator bool

	// Static key pair of local side.
	localStatic KeyPair

	// Static public key of remote side (initiator knows this upfront; responder learns it).
	remoteStatic Key

	// Ephemeral key pair of local side.
	localEphemeral KeyPair

	// Ephemeral public key of remote (learned during handshake).
	remoteEphemeral Key

	// Noise symmetric state.
	ck   [32]byte // chaining key
	h    [32]byte // handshake hash
	k    [32]byte // cipher key (zero if not yet established)
	nk   bool     // whether k is valid
	nMsg int      // next expected message index
}

// InitiatorHandshake creates a HandshakeState for the initiator role.
// localStatic is our long-term key pair; remoteStatic is the peer's public key.
func InitiatorHandshake(localStatic KeyPair, remoteStatic Key) (*HandshakeState, error) {
	hs := &HandshakeState{
		isInitiator:  true,
		localStatic:  localStatic,
		remoteStatic: remoteStatic,
	}
	if err := hs.initialize(); err != nil {
		return nil, err
	}
	return hs, nil
}

// ResponderHandshake creates a HandshakeState for the responder role.
func ResponderHandshake(localStatic KeyPair) (*HandshakeState, error) {
	hs := &HandshakeState{
		isInitiator: false,
		localStatic: localStatic,
	}
	if err := hs.initialize(); err != nil {
		return nil, err
	}
	return hs, nil
}

// initialize sets up the initial hash/chaining key state.
func (hs *HandshakeState) initialize() error {
	// ck = BLAKE2s(protocol_name)
	hs.ck = Hash([]byte(NoiseProtocolName))

	// h = BLAKE2s(ck || prologue)
	hs.h = HashMany(hs.ck[:], []byte(NoisePrologue))

	// Mix remote static public key into hash (both sides know it upfront in IK).
	if hs.isInitiator {
		hs.h = MixHash(hs.h, hs.remoteStatic[:])
	} else {
		hs.h = MixHash(hs.h, hs.localStatic.Public[:])
	}

	// Generate ephemeral key.
	var err error
	hs.localEphemeral, err = GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate ephemeral key: %w", err)
	}
	return nil
}

// --- Message 1: Initiator → Responder ---
// Layout: e(32) || encrypt(s)(32+16) || encrypt("" with ss)(16)

// WriteMessage1 produces the initiator's first handshake message.
// Returns the 96-byte message.
func (hs *HandshakeState) WriteMessage1() ([]byte, error) {
	if !hs.isInitiator {
		return nil, fmt.Errorf("WriteMessage1 called on responder")
	}
	if hs.nMsg != 0 {
		return nil, fmt.Errorf("wrong state for WriteMessage1")
	}

	msg := make([]byte, 0, 96)

	// -> e: send ephemeral public key, mix into hash
	hs.h = MixHash(hs.h, hs.localEphemeral.Public[:])
	msg = append(msg, hs.localEphemeral.Public[:]...)

	// -> es: DH(e, rs) — mix shared secret into ck/k
	es, err := ECDH(hs.localEphemeral.Private, hs.remoteStatic)
	if err != nil {
		return nil, fmt.Errorf("DH(e, rs): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, es[:])
	hs.nk = true

	// -> s: encrypt our static public key
	encStatic, err := EncryptZeroNonce(hs.k, hs.localStatic.Public[:], hs.h[:])
	if err != nil {
		return nil, err
	}
	hs.h = MixHash(hs.h, encStatic)
	msg = append(msg, encStatic...) // 32 + 16 bytes

	// -> ss: DH(s, rs)
	ss, err := ECDH(hs.localStatic.Private, hs.remoteStatic)
	if err != nil {
		return nil, fmt.Errorf("DH(s, rs): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, ss[:])

	// Encrypt empty payload
	encPayload, err := EncryptZeroNonce(hs.k, nil, hs.h[:])
	if err != nil {
		return nil, err
	}
	hs.h = MixHash(hs.h, encPayload)
	msg = append(msg, encPayload...) // 0 + 16 bytes

	hs.nMsg = 1
	return msg, nil
}

// ReadMessage1 processes the initiator's first handshake message on the responder.
func (hs *HandshakeState) ReadMessage1(msg []byte) error {
	if hs.isInitiator {
		return fmt.Errorf("ReadMessage1 called on initiator")
	}
	if hs.nMsg != 0 {
		return fmt.Errorf("wrong state for ReadMessage1")
	}
	if len(msg) < 96 {
		return fmt.Errorf("message1 too short: %d < 96", len(msg))
	}

	// <- e
	copy(hs.remoteEphemeral[:], msg[:32])
	hs.h = MixHash(hs.h, hs.remoteEphemeral[:])

	// <- es: DH(s, e_remote)
	es, err := ECDH(hs.localStatic.Private, hs.remoteEphemeral)
	if err != nil {
		return fmt.Errorf("DH(s, e_remote): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, es[:])
	hs.nk = true

	// <- s: decrypt remote static
	encStatic := msg[32:80] // 32 + 16
	remoteStatic, err := DecryptZeroNonce(hs.k, encStatic, hs.h[:])
	if err != nil {
		return fmt.Errorf("decrypt remote static: %w", err)
	}
	copy(hs.remoteStatic[:], remoteStatic)
	hs.h = MixHash(hs.h, encStatic)

	// <- ss: DH(s, s_remote)
	ss, err := ECDH(hs.localStatic.Private, hs.remoteStatic)
	if err != nil {
		return fmt.Errorf("DH(s, s_remote): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, ss[:])

	// Decrypt empty payload (auth tag only)
	encPayload := msg[80:96]
	if _, err := DecryptZeroNonce(hs.k, encPayload, hs.h[:]); err != nil {
		return fmt.Errorf("decrypt payload: %w", err)
	}
	hs.h = MixHash(hs.h, encPayload)

	hs.nMsg = 1
	return nil
}

// --- Message 2: Responder → Initiator ---
// Layout: e(32) || encrypt("" with ee+se)(16)

// WriteMessage2 produces the responder's reply message.
func (hs *HandshakeState) WriteMessage2() ([]byte, error) {
	if hs.isInitiator {
		return nil, fmt.Errorf("WriteMessage2 called on initiator")
	}
	if hs.nMsg != 1 {
		return nil, fmt.Errorf("wrong state for WriteMessage2")
	}

	msg := make([]byte, 0, 48)

	// <- e: send ephemeral, mix into hash
	hs.h = MixHash(hs.h, hs.localEphemeral.Public[:])
	msg = append(msg, hs.localEphemeral.Public[:]...)

	// <- ee: DH(e, e_remote)
	ee, err := ECDH(hs.localEphemeral.Private, hs.remoteEphemeral)
	if err != nil {
		return nil, fmt.Errorf("DH(ee): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, ee[:])
	hs.nk = true

	// <- se: DH(s, e_remote) — responder static, initiator ephemeral
	se, err := ECDH(hs.localStatic.Private, hs.remoteEphemeral)
	if err != nil {
		return nil, fmt.Errorf("DH(se): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, se[:])

	// Encrypt empty payload
	encPayload, err := EncryptZeroNonce(hs.k, nil, hs.h[:])
	if err != nil {
		return nil, err
	}
	hs.h = MixHash(hs.h, encPayload)
	msg = append(msg, encPayload...)

	hs.nMsg = 2
	return msg, nil
}

// ReadMessage2 processes the responder's reply on the initiator.
func (hs *HandshakeState) ReadMessage2(msg []byte) error {
	if !hs.isInitiator {
		return fmt.Errorf("ReadMessage2 called on responder")
	}
	if hs.nMsg != 1 {
		return fmt.Errorf("wrong state for ReadMessage2")
	}
	if len(msg) < 48 {
		return fmt.Errorf("message2 too short: %d < 48", len(msg))
	}

	// -> e
	copy(hs.remoteEphemeral[:], msg[:32])
	hs.h = MixHash(hs.h, hs.remoteEphemeral[:])

	// -> ee
	ee, err := ECDH(hs.localEphemeral.Private, hs.remoteEphemeral)
	if err != nil {
		return fmt.Errorf("DH(ee): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, ee[:])
	hs.nk = true

	// -> se: DH(e_local, s_remote) — initiator ephemeral, responder static
	se, err := ECDH(hs.localEphemeral.Private, hs.remoteStatic)
	if err != nil {
		return fmt.Errorf("DH(se): %w", err)
	}
	hs.ck, hs.k = MixKey(hs.ck, se[:])

	// Decrypt payload
	encPayload := msg[32:48]
	if _, err := DecryptZeroNonce(hs.k, encPayload, hs.h[:]); err != nil {
		return fmt.Errorf("decrypt payload: %w", err)
	}
	hs.h = MixHash(hs.h, encPayload)

	hs.nMsg = 2
	return nil
}

// TransportKeys derives the final send/recv symmetric keys.
// Call only after a complete handshake (nMsg == 2).
func (hs *HandshakeState) TransportKeys() (send, recv [32]byte, err error) {
	if hs.nMsg < 2 {
		return send, recv, fmt.Errorf("handshake not complete")
	}
	k1, k2 := Split(hs.ck)
	if hs.isInitiator {
		return k1, k2, nil
	}
	return k2, k1, nil
}

// RemoteStaticKey returns the authenticated remote static public key.
// Valid only after ReadMessage1 (responder) or after completing the handshake.
func (hs *HandshakeState) RemoteStaticKey() Key {
	return hs.remoteStatic
}
