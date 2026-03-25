package protocol

// Packet type identifiers (first byte of every karadul UDP packet).
const (
	TypeHandshakeInit uint8 = 0x01
	TypeHandshakeResp uint8 = 0x02
	TypeData          uint8 = 0x03
	TypeKeepalive     uint8 = 0x04
)

// Packet size constants.
const (
	// HandshakeInitSize is the fixed wire size of a HandshakeInit message.
	// 1 (type) + 3 (reserved) + 4 (senderIndex) + 32 (ephemeral) + 48 (static enc) + 16 (payload tag) + 28 (mac1+mac2 fields, 0 for now) = varies
	// Simplified: type(1) + reserved(3) + senderID(4) + e(32) + s_enc(48) + tag(16) = 104
	HandshakeInitSize = 104

	// HandshakeRespSize: type(1) + reserved(3) + senderID(4) + receiverID(4) + e(32) + tag(16) = 60 → round to 64
	HandshakeRespSize = 60

	// DataHeaderSize: type(1) + reserved(3) + receiverID(4) + counter(8) = 16
	DataHeaderSize = 16

	// AuthTagSize is the size of the ChaCha20-Poly1305 authentication tag.
	AuthTagSize = 16

	// MaxMTU is the maximum TUN MTU we support.
	MaxMTU = 1420

	// MaxPacketSize is the maximum UDP payload we ever allocate.
	MaxPacketSize = DataHeaderSize + MaxMTU + AuthTagSize + 64
)

// Magic bytes identify karadul packets on the wire.
var Magic = [4]byte{'K', 'R', 'D', 'L'}
