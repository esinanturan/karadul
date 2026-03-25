# Karadul — SPECIFICATION.md

> **Karadul** — Self-hosted, zero-dependency mesh VPN system written in Go.
> Ağ ören, mesh kuran, dokunduğu her noktayı birbirine bağlayan sistem.

---

## 1. Project Overview

### 1.1 What is Karadul?

Karadul is a WireGuard-compatible, self-hosted mesh VPN system that enables secure peer-to-peer connectivity across NAT boundaries. It combines a coordination server, encrypted tunnels, NAT traversal, and relay infrastructure into a single Go binary with zero external dependencies.

Think of it as: **Tailscale + Headscale in one binary, built from scratch.**

### 1.2 Core Philosophy

| Principle | Description |
|---|---|
| **Zero External Dependencies** | Only Go stdlib + extended stdlib (`golang.org/x/crypto`, `golang.org/x/sys` evaluated as "extended stdlib" for crypto safety — see §2.8 for rationale). All other components hand-written. |
| **Single Binary** | One binary serves all roles: node, coordination server, DERP relay. Role determined by CLI flags. |
| **Self-Hosted First** | No SaaS dependency. User owns coordination server, DERP relays, all keys. |
| **Modular Architecture** | Each subsystem (crypto, tunnel, nat, mesh, coordination) is an independent internal package with clean interfaces. |
| **Cross-Platform** | Linux (primary), macOS, Windows. Platform-specific code isolated behind build tags. |
| **WireGuard-Compatible Protocol** | Uses Noise IK handshake, X25519, ChaCha20-Poly1305, BLAKE2s — identical to WireGuard's cryptographic choices. |

### 1.3 Non-Goals (Explicit Exclusions)

- GUI application (CLI only, GUI is a separate future project)
- Mobile support (iOS/Android — future phase, not in scope)
- Kubernetes CNI plugin (future project)
- Commercial SaaS features (multi-tenant billing, etc.)
- Backward compatibility with actual WireGuard clients (protocol-compatible concepts, not wire-compatible)

---

## 2. Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        karadul binary                       │
│                                                             │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────────┐  │
│  │  CLI    │  │  Node   │  │ Coord   │  │  DERP Relay  │  │
│  │ Engine  │  │ Engine  │  │ Server  │  │   Server     │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └──────┬───────┘  │
│       │            │            │               │          │
│  ┌────┴────────────┴────────────┴───────────────┴───────┐  │
│  │                    Core Libraries                     │  │
│  │                                                       │  │
│  │  ┌─────────┐ ┌─────────┐ ┌──────┐ ┌──────────────┐  │  │
│  │  │ crypto  │ │ tunnel  │ │ nat  │ │    mesh      │  │  │
│  │  │         │ │         │ │      │ │              │  │  │
│  │  │-noise   │ │-tun_lin │ │-stun │ │-peer_manager │  │  │
│  │  │-chacha  │ │-tun_mac │ │-hole │ │-topology     │  │  │
│  │  │-blake2s │ │-tun_win │ │-derp │ │-routing      │  │  │
│  │  │-x25519  │ │-packet  │ │      │ │              │  │  │
│  │  └─────────┘ └─────────┘ └──────┘ └──────────────┘  │  │
│  │                                                       │  │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌───────────┐ │  │
│  │  │  auth   │ │  dns    │ │  config  │ │    log    │ │  │
│  │  │         │ │         │ │          │ │           │ │  │
│  │  │-keys    │ │-resolver│ │-file     │ │-structured│ │  │
│  │  │-acl     │ │-magic   │ │-validate │ │-levels    │ │  │
│  │  └─────────┘ └─────────┘ └──────────┘ └───────────┘ │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Execution Modes

Single binary, mode determined by subcommand:

```bash
karadul up                    # Start as mesh node (default mode)
karadul server                # Start as coordination server
karadul relay                 # Start as DERP relay server
karadul server --with-relay   # Coordination + DERP relay combined
karadul keygen                # Generate node keypair
karadul peers                 # List connected peers
karadul status                # Show node status, latency, throughput
karadul ping <peer>           # Ping a specific peer through the mesh
karadul exit-node enable      # Enable this node as exit node
karadul exit-node use <peer>  # Route all traffic through peer
karadul dns                   # Show MagicDNS entries
```

### 2.3 Single Binary Role Architecture

```
karadul binary
├── cmd/karadul/main.go          # Entry point, CLI router
├── internal/node/               # Node engine (mesh participant)
├── internal/coordinator/        # Coordination server
├── internal/relay/              # DERP relay server
├── internal/crypto/             # All cryptographic operations
├── internal/tunnel/             # TUN device + packet I/O
├── internal/nat/                # STUN + hole punching
├── internal/mesh/               # Peer management + topology
├── internal/auth/               # Key management + ACL
├── internal/dns/                # MagicDNS resolver
├── internal/config/             # Configuration management
├── internal/protocol/           # Wire protocol definitions
└── internal/log/                # Structured logging
```

### 2.4 Component Interaction Flow

**Node startup sequence:**

```
1. Load config + keys (or generate if first run)
2. Open TUN device (requires root/admin)
3. Connect to coordination server (HTTPS long-poll)
4. Receive peer list + network topology
5. For each peer:
   a. Attempt STUN to discover own public endpoint
   b. Exchange endpoint info via coordination server
   c. Attempt UDP hole punch (direct connection)
   d. If hole punch fails → connect via DERP relay
   e. Perform Noise IK handshake
   f. Establish encrypted tunnel
6. Configure routes (virtual subnet + optional exit node)
7. Start DNS resolver (MagicDNS)
8. Begin packet forwarding: TUN ↔ encrypted UDP
```

### 2.5 Network Topology Model

```
                    ┌──────────────────┐
                    │  Coordination    │
                    │  Server          │
                    │  (HTTPS API)     │
                    └────────┬─────────┘
                             │ control plane
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────┴────┐   ┌────┴────┐   ┌────┴────┐
         │ Node A  │   │ Node B  │   │ Node C  │
         │ 10.0.0.1│   │ 10.0.0.2│   │ 10.0.0.3│
         └────┬────┘   └────┬────┘   └────┬────┘
              │              │              │
              │   data plane (encrypted UDP)│
              ├──────────────┤              │
              │              ├──────────────┤
              ├──────────────┼──────────────┤
              │              │              │
         Direct P2P    Direct P2P    Via DERP
         (hole punch)  (hole punch)  (relay)
```

**Control Plane:** Coordination server distributes topology, peer endpoints, ACLs. Lightweight, stateless-friendly.

**Data Plane:** All traffic flows directly between nodes (P2P). DERP relay only when direct connection impossible. Coordination server NEVER sees data traffic.

### 2.6 Virtual Network Addressing

Each Karadul network (called a **"Web"** — örümcek ağı) gets a virtual subnet:

- Default: `100.64.0.0/10` (CGNAT range, same as Tailscale)
- Configurable per-web
- Each node gets a stable virtual IP assigned by coordination server
- IPs persist across reconnections (stored in coordination server DB)

### 2.7 Platform Support Matrix

| Feature | Linux | macOS | Windows |
|---|---|---|---|
| TUN device | `/dev/net/tun` via `ioctl` | `utun` via `sys/socket` | Wintun driver via DLL |
| Route manipulation | Netlink (`RTM_NEWROUTE`) | `route` command / syscall | `netsh` / Windows API |
| DNS override | `resolvconf` / systemd-resolved | `scutil` | `netsh` / NRPT |
| Exit node NAT | `iptables` / `nftables` | `pfctl` | Windows NAT API |
| Auto-start | systemd service | launchd plist | Windows service |
| Privileges | `CAP_NET_ADMIN` or root | root (sudo) | Administrator |

### 2.8 Zero-Dependency Policy & Exceptions

**Hard rule:** No third-party Go modules.

**Extended stdlib evaluation:** The following `golang.org/x/` packages are evaluated:

| Package | Decision | Rationale |
|---|---|---|
| `golang.org/x/crypto` | **ACCEPTED as extended stdlib** | ChaCha20-Poly1305, BLAKE2s, Curve25519 implementations are security-critical. Hand-writing crypto primitives introduces unacceptable risk of timing attacks, side-channel vulnerabilities. Go team maintains these. |
| `golang.org/x/sys` | **ACCEPTED as extended stdlib** | Low-level syscall wrappers for TUN device, netlink, route manipulation. Platform-specific code that mirrors kernel interfaces. |
| `golang.org/x/net` | **REJECTED** | Not needed. `net` stdlib sufficient for UDP, TCP, HTTP. |
| Everything else | **REJECTED** | No ORM, no web framework, no config library, no logging framework. |

**Rationale for x/crypto exception:** WireGuard's security model depends on correct implementation of ChaCha20-Poly1305 and BLAKE2s. A single timing vulnerability in hand-written crypto can silently compromise every tunnel. The `x/crypto` package is audited, fuzz-tested, and maintained by the Go security team. This is the one place where "zero dependency" purism would be irresponsible.

---

## 3. Cryptographic Protocol

### 3.1 Protocol Overview

Karadul uses a protocol inspired by WireGuard, based on the **Noise Protocol Framework** (IK pattern).

**Cipher suite (non-negotiable, no cipher agility):**

| Primitive | Algorithm | Source |
|---|---|---|
| Key Exchange | X25519 (Curve25519 ECDH) | `crypto/ecdh` (stdlib Go 1.20+) |
| Symmetric Encryption | ChaCha20-Poly1305 (AEAD) | `golang.org/x/crypto/chacha20poly1305` |
| Hash | BLAKE2s-256 | `golang.org/x/crypto/blake2s` |
| MAC | Keyed BLAKE2s | `golang.org/x/crypto/blake2s` |
| KDF | HKDF via BLAKE2s | Hand-written using BLAKE2s |

### 3.2 Key Types

```
Identity Key Pair (long-term):
  - Private: 32 bytes, generated once, stored on disk
  - Public:  32 bytes, registered with coordination server

Ephemeral Key Pair (per-handshake):
  - Generated fresh for every new handshake
  - Never stored, exists only in memory

Pre-shared Key (optional):
  - 32 bytes, for post-quantum resistance layer
  - Shared out-of-band between two peers
```

### 3.3 Noise IK Handshake

The Noise IK pattern assumes the initiator knows the responder's static public key (obtained from coordination server).

```
Initiator (I) knows: I_static, R_static_pub
Responder (R) knows: R_static

=== Message 1: Initiator → Responder ===

I generates: I_ephemeral
I sends:
  - I_ephemeral.pub                          (32 bytes, cleartext)
  - AEAD_encrypt(I_static.pub)               (32 + 16 bytes)
  - AEAD_encrypt(timestamp)                  (12 + 16 bytes)

=== Message 2: Responder → Initiator ===

R generates: R_ephemeral
R sends:
  - R_ephemeral.pub                          (32 bytes, cleartext)
  - AEAD_encrypt(empty)                      (0 + 16 bytes)

=== After Handshake ===

Both derive:
  - T_send key (initiator→responder transport key)
  - T_recv key (responder→initiator transport key)
  
Transport keys are used for all subsequent data packets.
```

### 3.4 Transport Packet Format

After handshake, every data packet:

```
┌─────────────┬────────┬──────────────────┬──────────┐
│ Receiver ID │ Counter│ Encrypted Payload│ Auth Tag │
│  (4 bytes)  │(8 bytes)│   (variable)    │(16 bytes)│
└─────────────┴────────┴──────────────────┴──────────┘

- Receiver ID: identifies which tunnel this packet belongs to
- Counter: 64-bit nonce, monotonically increasing (replay protection)
- Encrypted Payload: ChaCha20-Poly1305(transport_key, counter, IP_packet)
- Auth Tag: part of AEAD output, authentication
```

### 3.5 Key Rotation

- Transport keys rotate every **2 minutes** or every **2^64 - 1** packets (whichever first)
- Rotation = new handshake with new ephemeral keys
- Old keys kept for **30 seconds** after rotation (in-flight packet grace period)
- Initiator for re-keying: the peer who originally initiated

### 3.6 Replay Protection

- 64-bit counter per direction per tunnel
- Sliding window bitmap (2048 packets wide) for out-of-order tolerance
- Packets with counter below window floor are dropped
- Duplicate counters within window are dropped

---

## 4. TUN Device Layer

### 4.1 Interface Design

```go
// tunnel.Device is the platform-abstracted TUN interface
type Device interface {
    // Name returns the OS interface name (e.g., "utun5", "karadul0")
    Name() string
    
    // Read reads one IP packet from the TUN device (blocks)
    Read(buf []byte) (n int, err error)
    
    // Write writes one IP packet to the TUN device
    Write(buf []byte) (n int, err error)
    
    // MTU returns the device MTU
    MTU() int
    
    // SetMTU configures the device MTU
    SetMTU(mtu int) error
    
    // Close destroys the TUN device
    Close() error
}
```

### 4.2 Platform Implementations

**Linux (`tunnel/tun_linux.go`):**
```
- Open /dev/net/tun
- ioctl(TUNSETIFF) with IFF_TUN | IFF_NO_PI
- Interface name: "karadul0" (configurable)
- Route manipulation via netlink sockets (RTM_NEWROUTE, RTM_DELROUTE)
- IP address assignment via netlink (RTM_NEWADDR)
```

**macOS (`tunnel/tun_darwin.go`):**
```
- Create utun device via sys/socket with SYSPROTO_CONTROL
- Interface name: auto-assigned "utunN"
- 4-byte AF header prepended to each packet (macOS utun quirk)
- Route manipulation via route(4) socket or exec("route")
- IP address via ifconfig or ioctl
```

**Windows (`tunnel/tun_windows.go`):**
```
- Load wintun.dll dynamically (bundled or downloaded)
- WintunCreateAdapter / WintunStartSession / WintunAllocateSendPacket
- Interface name: "Karadul"
- Route manipulation via CreateIpForwardEntry2 or exec("netsh")
- IP address via Windows API
```

### 4.3 MTU Considerations

```
Standard Ethernet MTU:          1500 bytes
WireGuard overhead:             -80 bytes (IPv4) / -80 bytes (IPv6)
  - Outer UDP header:           8 bytes
  - Outer IP header:            20 bytes (IPv4)
  - WireGuard header:           16 bytes (receiver_id + counter)
  - AEAD auth tag:              16 bytes
  - Padding:                    ~0-15 bytes

Karadul TUN MTU:                1420 bytes (default, matches WireGuard)
```

---

## 5. NAT Traversal

### 5.1 Strategy Hierarchy

NAT traversal attempts in order:

```
1. Direct connection (both peers have public IP)
   └→ Success? Use direct UDP tunnel

2. STUN discovery + UDP hole punching
   └→ Both peers discover their public endpoint via STUN
   └→ Exchange endpoints via coordination server
   └→ Simultaneously send UDP packets to each other
   └→ Success? Use direct UDP tunnel

3. UPnP / NAT-PMP port mapping (optional)
   └→ Request port mapping from router
   └→ Success? Advertise mapped port, use direct tunnel

4. DERP relay (guaranteed fallback)
   └→ Both peers connect to nearest DERP relay (TCP/WebSocket)
   └→ Encrypted packets relayed through server
   └→ Always works, higher latency
```

### 5.2 STUN Implementation

Minimal STUN client (RFC 5389) — no full RFC compliance needed, only Binding Request:

```
STUN Binding Request:
┌──────────────┬──────────┬──────────────────┐
│ Type (2B)    │ Length   │ Magic Cookie     │
│ 0x0001       │ 0x0000  │ 0x2112A442       │
├──────────────┴──────────┴──────────────────┤
│ Transaction ID (12 bytes, random)          │
└────────────────────────────────────────────┘

STUN Binding Response:
- Contains XOR-MAPPED-ADDRESS attribute
- XOR with magic cookie to get public IP:port
```

**STUN servers used:** Configurable list, defaults to public Google/Cloudflare STUN servers:
```
stun.l.google.com:19302
stun.cloudflare.com:3478
```

Self-hosted STUN server support: coordination server can also serve STUN.

### 5.3 UDP Hole Punching

```
Timeline:
  t=0:  Node A sends STUN → learns A_public = 1.2.3.4:50000
  t=0:  Node B sends STUN → learns B_public = 5.6.7.8:60000
  t=1:  Coordination server tells A about B's endpoint and vice versa
  t=2:  A sends UDP to 5.6.7.8:60000 (may be dropped by B's NAT, but opens A's NAT)
  t=2:  B sends UDP to 1.2.3.4:50000 (may be dropped by A's NAT, but opens B's NAT)
  t=3:  A retries → B's NAT now has mapping → packet arrives!
  t=3:  B retries → A's NAT now has mapping → packet arrives!
  t=4:  Bidirectional UDP flow established → proceed to Noise handshake
```

**Hole punch timing:**
- Simultaneous send with jitter (±50ms random)
- Retry up to 10 times with 500ms intervals
- Total timeout: 5 seconds
- If fails → fall back to DERP

**NAT type detection:**
- Cone NAT (Full/Restricted/Port-Restricted): hole punch works
- Symmetric NAT: hole punch usually fails → DERP fallback
- Detection via STUN response analysis (same/different mapped ports from different STUN servers)

### 5.4 DERP Relay

**DERP = Designated Encrypted Relay for Packets**

DERP is the guaranteed fallback when direct connection is impossible.

**Key properties:**
- Relay sees only encrypted packets — cannot decrypt (no transport keys)
- Runs over TCP/443 (HTTPS upgrade to WebSocket) — works through any corporate firewall
- Stateless packet forwarding — relay just routes based on receiver public key
- Multiple DERP regions for latency optimization

**DERP protocol (simplified):**

```
Client → Relay:
  1. HTTPS upgrade to WebSocket on /derp
  2. Send ClientInfo { public_key, version }
  3. Send packets: FrameSendPacket { dest_public_key, encrypted_payload }

Relay → Client:
  1. Forward packets: FrameRecvPacket { src_public_key, encrypted_payload }
  2. Notify: FramePeerPresent { public_key } / FramePeerGone { public_key }
```

**DERP server is embedded in karadul binary:**
```bash
karadul relay --addr=:443 --tls-cert=cert.pem --tls-key=key.pem
# or combined with coordination server:
karadul server --with-relay --addr=:443
```

### 5.5 Connection Upgrade Path

Even after establishing a DERP relay connection, Karadul continuously attempts direct connection:

```
1. Connected via DERP relay
2. Background goroutine retries hole punching every 30 seconds
3. If direct UDP path discovered:
   a. Perform latency comparison (direct vs DERP)
   b. If direct is faster → migrate tunnel to direct
   c. Keep DERP as fallback (don't disconnect)
4. If direct path lost → seamlessly fall back to DERP
```

---

## 6. Coordination Server

### 6.1 Responsibilities

```
- Node registration and authentication (public key based)
- Virtual IP assignment and management
- Peer list distribution (which nodes exist, their endpoints)
- ACL distribution (who can talk to whom)
- DERP relay map distribution (which DERP servers are available)
- Endpoint exchange for NAT traversal
- Network ("Web") management
```

### 6.2 API Design

REST/JSON over HTTPS. Authentication via node's static key (signed requests).

```
POST   /api/v1/register          # Register node, get virtual IP
POST   /api/v1/poll              # Long-poll for updates (peers, ACLs, DERP map)
POST   /api/v1/update-endpoint   # Report current public endpoint
GET    /api/v1/peers             # Get current peer list
GET    /api/v1/derp-map          # Get DERP relay server list
POST   /api/v1/ping              # Coordination server health check

Admin endpoints:
GET    /api/v1/admin/nodes       # List all registered nodes
DELETE /api/v1/admin/nodes/:id   # Remove a node
PUT    /api/v1/admin/acl         # Update ACL policy
GET    /api/v1/admin/web         # Get network info
```

### 6.3 Authentication Model

**No username/password.** Pure public-key authentication.

```
1. Node generates keypair: karadul keygen → creates ~/.karadul/private.key
2. Node registers with coordination server:
   POST /register { public_key: "...", hostname: "ersin-macbook" }
   - First registration: server assigns virtual IP, returns auth token
   - Token = HMAC(server_secret, node_public_key)
3. Subsequent requests signed with node's private key:
   - Request includes: X-Karadul-Key: <public_key>
   - Request includes: X-Karadul-Sig: <signature_of_request_body>
   - Server verifies signature against registered public key
```

**Node approval modes:**
- `auto-approve`: Any node with valid key can join (default for testing)
- `manual-approve`: Admin must approve new nodes via CLI or API
- `pre-auth-keys`: Generate single-use or reusable keys for onboarding

```bash
# Generate a pre-auth key
karadul server auth create-key --reusable --expiry=24h
# → karadul-authkey-abc123def456

# Node uses it to register
karadul up --auth-key=karadul-authkey-abc123def456
```

### 6.4 Long-Poll Mechanism

Instead of WebSocket (to avoid complexity), coordination server uses HTTP long-poll:

```
Client:
  POST /api/v1/poll
  Body: { map_version: 42 }  # Client's current version
  
Server:
  - If server version > 42 → immediately return new state
  - If server version == 42 → hold connection open (up to 60s)
  - When state changes → respond with new state
  - On timeout → respond with empty update, client re-polls

Response:
{
  "map_version": 43,
  "peers": [...],
  "derp_map": {...},
  "acl": {...},
  "your_ip": "100.64.0.5",
  "dns_config": {...}
}
```

### 6.5 Data Storage

Coordination server uses an embedded file-based store (no external database):

```
~/.karadul-server/
├── state.json           # Full network state (nodes, IPs, ACLs)
├── state.json.bak       # Previous state (crash recovery)
├── auth-keys/           # Pre-auth keys
└── acl.json             # Access control policy
```

Format: JSON, loaded into memory at startup, written on mutation with fsync. Sufficient for networks up to ~1000 nodes. For larger scale, future phase adds SQLite option (still zero-dep via pure Go SQLite).

---

## 7. Mesh Networking

### 7.1 Topology Management

**On-demand mesh:** Not full mesh. Tunnels established only when nodes need to communicate.

```
Peer states:
  - DISCOVERED: Known from coordination server, no tunnel
  - CONNECTING: Handshake in progress
  - DIRECT: Active tunnel via direct UDP
  - RELAYED: Active tunnel via DERP relay
  - IDLE: Tunnel exists but no recent traffic (will timeout)
  - EXPIRED: Tunnel torn down after idle timeout
```

**Idle timeout:** 5 minutes. If no packets for 5 min, tunnel moves to EXPIRED, resources freed. Re-established on next packet.

### 7.2 Peer Selection & Routing

```go
// Routing decision for outgoing IP packet
func (m *Mesh) RoutePacket(dstIP net.IP) (Peer, error) {
    // 1. Direct peer match — destination IS a peer's virtual IP
    if peer, ok := m.peerByIP[dstIP]; ok {
        return peer, nil
    }
    
    // 2. Subnet route — peer advertises a subnet (e.g., office LAN)
    if peer, ok := m.routeTable.Lookup(dstIP); ok {
        return peer, nil
    }
    
    // 3. Exit node — default route if exit node configured
    if m.exitNode != nil {
        return m.exitNode, nil
    }
    
    return nil, ErrNoRoute
}
```

### 7.3 Subnet Routing

Nodes can advertise local subnets to the mesh:

```bash
# Node at office advertises LAN
karadul up --advertise-routes=192.168.1.0/24,10.0.50.0/24

# Other nodes can now reach 192.168.1.x through this node
# (requires accept-routes on other nodes)
karadul up --accept-routes
```

Coordination server distributes route advertisements to all peers.

### 7.4 Exit Node

Any node can serve as internet exit node:

```bash
# On the exit node (e.g., VPS with clean IP):
karadul up --exit-node-enable

# On client nodes:
karadul exit-node use node-vps-amsterdam
# → Default route changed: 0.0.0.0/0 → via karadul tunnel
# → DNS changed to tunnel DNS to prevent leaks
```

**Exit node implementation:**
1. Client sets default route through TUN device
2. All traffic (except karadul control traffic) goes into tunnel
3. Exit node receives decrypted packets
4. Exit node performs SNAT/masquerade (iptables on Linux, pfctl on macOS)
5. Responses come back, encrypted, sent to client

**Split tunneling (optional):**
```bash
karadul up --exit-node=vps --exit-node-allow-lan
# → LAN traffic (192.168.x.x) stays local
# → Everything else goes through exit node
```

---

## 8. DNS System (MagicDNS)

### 8.1 Overview

Each node gets a human-readable hostname on the mesh:

```
ersin-macbook.web.karadul  →  100.64.0.1
office-server.web.karadul  →  100.64.0.2
vps-amsterdam.web.karadul  →  100.64.0.3
```

### 8.2 Implementation

- Karadul runs a lightweight DNS resolver on `100.64.0.53:53` (or localhost)
- Intercepts DNS queries:
  - `*.web.karadul` → resolve from coordination server's peer list
  - Everything else → forward to upstream DNS (system default or configured)
- DNS override:
  - Linux: write to `/etc/resolv.conf` or talk to systemd-resolved
  - macOS: `scutil` to set DNS for the karadul interface
  - Windows: NRPT (Name Resolution Policy Table) for split DNS

### 8.3 DNS Leak Prevention

When using exit node, ALL DNS queries must go through the tunnel:

```
- Override system DNS to karadul's local resolver
- Local resolver forwards non-karadul queries through tunnel to exit node
- Exit node resolves and returns through tunnel
- No DNS query ever leaves unencrypted
```

---

## 9. Access Control (ACL)

### 9.1 Policy Model

JSON-based ACL policy on coordination server:

```json
{
  "acl": [
    {
      "action": "accept",
      "src": ["ersin-macbook", "ersin-phone"],
      "dst": ["*:*"]
    },
    {
      "action": "accept",
      "src": ["office-team"],
      "dst": ["office-server:22,80,443"]
    },
    {
      "action": "deny",
      "src": ["*"],
      "dst": ["*:*"]
    }
  ],
  "groups": {
    "office-team": ["alice-laptop", "bob-desktop", "charlie-macbook"]
  },
  "tags": {
    "tag:server": ["office-server", "vps-amsterdam"],
    "tag:admin": ["ersin-macbook"]
  }
}
```

### 9.2 ACL Enforcement

ACLs are enforced at the node level:
- Coordination server distributes ACL to all nodes
- Each node filters packets based on src/dst IP and port
- Unauthorized packets are silently dropped (no ICMP response)
- ACL changes propagated via long-poll, applied within seconds

---

## 10. Configuration

### 10.1 Node Configuration

```yaml
# ~/.karadul/config.yaml
server:
  url: https://coord.example.com
  auth_key: karadul-authkey-abc123  # Only for first registration

node:
  hostname: ersin-macbook           # Optional, defaults to OS hostname
  advertise_routes:                 # Subnets to share with mesh
    - 192.168.1.0/24
  accept_routes: true               # Accept routes from other nodes
  exit_node: ""                     # Peer name to use as exit node
  exit_node_enable: false           # Serve as exit node
  exit_node_allow_lan: true         # Allow LAN access when using exit node

network:
  listen_port: 0                    # 0 = random (default), or fixed port
  mtu: 1420                         # TUN device MTU

dns:
  magic_dns: true                   # Enable *.web.karadul resolution
  upstream:                         # Upstream DNS servers
    - 1.1.1.1
    - 8.8.8.8

log:
  level: info                       # debug, info, warn, error
  format: text                      # text or json
```

### 10.2 Coordination Server Configuration

```yaml
# ~/.karadul-server/config.yaml
server:
  addr: :443
  tls_cert: /path/to/cert.pem
  tls_key: /path/to/key.pem
  
  # or use auto TLS (Let's Encrypt via ACME)
  auto_tls: true
  domain: coord.example.com

approval:
  mode: auto-approve                # auto-approve, manual, pre-auth-key

network:
  subnet: 100.64.0.0/10
  dns_domain: web.karadul

derp:
  enabled: true                     # Run embedded DERP relay
  stun_port: 3478                   # Embedded STUN server port

storage:
  path: ~/.karadul-server/state.json
```

### 10.3 Key Storage

```
~/.karadul/
├── config.yaml          # Node configuration
├── private.key          # Node's X25519 private key (chmod 600)
├── public.key           # Node's X25519 public key
└── cached-state.json    # Last known peer list (offline bootstrap)
```

---

## 11. Observability

### 11.1 Structured Logging

```
level=info msg="handshake complete" peer=office-server latency=23ms via=direct
level=info msg="tunnel established" peer=vps-amsterdam via=derp region=eu-west
level=warn msg="hole punch failed" peer=charlie-macbook nat_type=symmetric
level=debug msg="packet" dir=tx peer=office-server size=1420 counter=483729
```

### 11.2 Status API

Each node exposes a local status API (unix socket or localhost):

```bash
karadul status
# Node: ersin-macbook (100.64.0.1)
# Web:  my-network
# Coordination: connected (coord.example.com)
#
# Peers:
#   office-server  100.64.0.2  direct   23ms   ↑12MB ↓8MB
#   vps-amsterdam  100.64.0.3  relay    89ms   ↑1MB  ↓500KB
#   charlie-mac    100.64.0.4  idle     -      -
```

### 11.3 Metrics

Optional Prometheus-compatible metrics endpoint:

```
karadul_peers_connected{} 3
karadul_peers_direct{} 2
karadul_peers_relayed{} 1
karadul_tunnel_tx_bytes{peer="office-server"} 12582912
karadul_tunnel_rx_bytes{peer="office-server"} 8388608
karadul_tunnel_latency_ms{peer="office-server"} 23
karadul_handshakes_total{} 47
karadul_derp_fallback_total{} 3
```

---

## 12. Security Model

### 12.1 Threat Model

| Threat | Mitigation |
|---|---|
| MITM on data plane | Noise IK handshake with pre-known public keys |
| Replay attacks | 64-bit monotonic counter + sliding window |
| Key compromise | 2-minute key rotation, ephemeral keys per session |
| Coordination server compromise | Server only sees metadata (endpoints, public keys), never traffic content |
| DERP relay compromise | Relay only sees encrypted packets, no transport keys |
| Rogue node | ACL enforcement, admin-approved node registration |
| DDoS on coordination | Rate limiting, node authentication required for all endpoints |
| DNS leak | Forced tunnel DNS when exit node active |

### 12.2 Key Security Properties

```
✓ Perfect forward secrecy (ephemeral keys per handshake)
✓ Identity hiding (static key encrypted in handshake)
✓ Post-quantum optional (PSK layer)
✓ Zero-trust data plane (encryption mandatory, no plaintext mode)
✓ Minimal attack surface (fixed cipher suite, no negotiation)
✓ Coordination server sees no traffic content
✓ DERP relay sees no traffic content
```

---

## 13. Development Phases

### Phase 1 — Point-to-Point Tunnel (Foundation)
- TUN device (Linux first, then macOS)
- X25519 key generation and exchange
- Noise IK handshake
- ChaCha20-Poly1305 packet encryption
- UDP transport between two hardcoded endpoints
- Basic IP packet forwarding
- **Milestone:** `ping` works between two machines through encrypted tunnel

### Phase 2 — Coordination Server
- HTTP API (register, poll, update-endpoint)
- Virtual IP assignment
- Peer list distribution via long-poll
- Public-key authentication
- File-based state storage
- Pre-auth key generation
- **Milestone:** Nodes auto-discover each other via coordination server

### Phase 3 — NAT Traversal
- STUN client implementation
- UDP hole punching with coordination
- NAT type detection
- DERP relay server (embedded in binary)
- DERP client in node
- Automatic fallback: direct → DERP
- Background upgrade: DERP → direct
- **Milestone:** Two nodes behind NAT connect without port forwarding

### Phase 4 — Full Mesh
- Multi-peer support (concurrent tunnels)
- On-demand tunnel creation
- Idle tunnel cleanup
- Peer state machine (DISCOVERED → CONNECTING → DIRECT/RELAYED → IDLE → EXPIRED)
- Route table management
- Subnet advertising
- **Milestone:** 5+ nodes in mesh, any-to-any communication

### Phase 5 — Exit Node & DNS
- Exit node enable/use
- NAT masquerade on exit node
- DNS resolver (MagicDNS)
- DNS leak prevention
- Split tunneling
- **Milestone:** Browse internet through exit node with zero DNS leaks

### Phase 6 — Security & Polish
- ACL system
- Key rotation automation
- Pre-auth keys with expiry
- Windows TUN support (Wintun)
- systemd/launchd service files
- Prometheus metrics
- Comprehensive logging
- **Milestone:** Production-ready for personal/small-team use

---

## 14. Testing Strategy

### 14.1 Unit Tests
- Crypto primitives: test vectors from RFC 7539 (ChaCha20-Poly1305), RFC 7693 (BLAKE2s)
- Noise handshake: test vectors from Noise spec
- Packet parsing: fuzz testing with `go test -fuzz`
- ACL engine: policy evaluation tests

### 14.2 Integration Tests
- Multi-node mesh on localhost (different UDP ports)
- TUN device creation/destruction
- Coordination server API
- Long-poll mechanism
- Key rotation under load

### 14.3 End-to-End Tests
```bash
# Scripted E2E test
1. Start coordination server
2. Start 3 nodes (different terminals/processes)
3. Verify all nodes discover each other
4. ping between all pairs
5. Transfer file between nodes (verify integrity)
6. Kill one node, verify mesh reconverges
7. Enable exit node, verify internet access
8. Apply ACL, verify blocked traffic is dropped
```

### 14.4 Cross-Platform Test Matrix
| Test | Linux | macOS | Windows |
|---|---|---|---|
| TUN create/destroy | CI (GitHub Actions) | Local | Local/VM |
| Point-to-point tunnel | CI | Local | VM |
| Mesh (3+ nodes) | CI | Local + CI | VM |
| Exit node | CI | Local | VM |
| NAT traversal | Manual (cloud VMs) | Manual | Manual |

---

## 15. Performance Targets

| Metric | Target |
|---|---|
| Handshake latency | < 50ms (LAN), < 200ms (WAN) |
| Throughput (direct) | > 500 Mbps on modern hardware |
| Throughput (DERP relay) | Limited by relay bandwidth |
| Memory per peer | < 64KB (keys, buffers, state) |
| Idle memory (10 peers) | < 20MB |
| Binary size | < 15MB |
| Startup to connected | < 3 seconds (coordination server reachable) |
| Key rotation | Zero dropped packets during rotation |
| Reconnection after network change | < 2 seconds |

---

## 16. Future Considerations (Out of Scope, Documented for Later)

- **Mobile clients:** iOS (Network Extension), Android (VpnService)
- **Web admin UI:** React dashboard for coordination server
- **SSO integration:** OIDC/SAML for node authentication
- **Multi-web:** Multiple isolated networks on same coordination server
- **Taildrop equivalent:** Encrypted file transfer between nodes
- **SSH integration:** `karadul ssh <peer>` shortcut
- **MCP server:** Expose mesh status/control to LLM agents
- **Container mode:** Sidecar proxy for Docker/K8s
