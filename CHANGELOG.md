# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Windows Support** - Full Wintun integration for Windows TUN devices
  - `internal/tunnel/tun_windows.go` - Windows TUN implementation using Wintun driver
  - `internal/tunnel/wintun_dll_windows.go` - Wintun DLL loading and management
  - `internal/tunnel/wintun_dll_other.go` - Non-Windows stubs
  - `karadul wintun-check` command to verify Wintun driver installation
- **Cross-Platform Firewall Management**
  - `internal/firewall/firewall_windows.go` - Windows Firewall netsh integration
  - `internal/firewall/firewall_linux.go` - Linux firewall stubs
  - `internal/firewall/firewall_darwin.go` - macOS firewall stubs
  - `karadul firewall` command with `setup`, `remove`, `check`, `allow-port` subcommands
- **GitHub Actions Workflows**
  - `release.yml` - Automated binary releases for 10+ platforms
  - `container.yml` - Docker image builds and GHCR publishing
- **Docker Support**
  - `Dockerfile` - Multi-stage build for minimal runtime image
  - `docker-compose.yml` - Example Docker Compose configuration
- **Homebrew Formula** - macOS/Linux Homebrew tap support
  - `contrib/homebrew/karadul.rb.template` - Formula template
  - `contrib/homebrew/update-formula.sh` - Formula update script

### Changed
- Updated CI workflow to test all supported platforms (Linux, macOS, Windows, FreeBSD, OpenBSD)
- Updated README with new installation methods (Homebrew, Docker, Windows binary)
- Expanded comparison table to include Windows support

### Platform Support
- ✅ Linux (amd64, arm64, armv7)
- ✅ macOS (amd64, arm64)
- ✅ Windows (amd64, arm64, x86) - NEW
- ✅ FreeBSD (amd64) - NEW
- ✅ OpenBSD (amd64) - NEW

### Core Packages
- `internal/crypto` - Noise protocol implementation, X25519, ChaCha20-Poly1305
- `internal/tunnel` - TUN device management (Linux, macOS)
- `internal/nat` - STUN client and hole punching
- `internal/coordinator` - Coordination server and state management
- `internal/mesh` - Peer management and topology
- `internal/relay` - DERP relay server
- `internal/auth` - Authentication keys and validation
- `internal/dns` - MagicDNS resolver

## [0.1.0] - TBD

### Added
- Initial release
- Basic mesh networking
- Coordination server
- NAT traversal

[Unreleased]: https://github.com/ersinkoc/karadul/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ersinkoc/karadul/releases/tag/v0.1.0
