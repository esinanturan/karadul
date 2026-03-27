# Security Policy

## Supported Versions

The following versions of Karadul are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1.0 | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please follow these steps:

### 1. Do Not Open a Public Issue

Please do **not** create a public GitHub issue for security vulnerabilities. This helps protect users while we work on a fix.

### 2. Contact Us Directly

Email security concerns to: **security@karadul.dev** (or the maintainer's email)

Include the following information:
- Description of the vulnerability
- Steps to reproduce (if applicable)
- Potential impact
- Suggested fix (if you have one)

### 3. Response Time

We aim to respond to security reports within:
- **48 hours** — Initial acknowledgment
- **7 days** — Initial assessment and timeline
- **30 days** — Fix deployed (depending on severity)

### 4. Disclosure Policy

We follow responsible disclosure:
1. We work with you to understand and fix the issue
2. We release a patched version
3. We publicly disclose the issue (with credit to the reporter if desired)

## Security Best Practices

When deploying Karadul:

### Coordination Server
- Use TLS in production (`--tls` flag)
- Keep auth keys secure and rotate regularly
- Use strong pre-shared keys
- Monitor access logs

### Node
- Protect private keys (stored in `~/.karadul/`)
- Use firewall rules to restrict access
- Keep software updated

### Network
- Use ACLs to restrict traffic between nodes
- Enable exit nodes only when needed
- Monitor mesh traffic for anomalies

## Known Security Considerations

### Current Limitations

As a beta project, please be aware of:

1. **No built-in rate limiting** (yet) — Use external tools if needed
2. **Basic ACL system** — More granular controls coming in v0.2.0
3. **No audit logging** — Planned for v0.2.0

### Cryptographic Implementation

Karadul uses well-established cryptographic libraries:
- [golang.org/x/crypto](https://golang.org/x/crypto) for X25519, ChaCha20-Poly1305
- [golang.org/x/sys](https://golang.org/x/sys) for system calls

We do not implement our own cryptography.

## Security-Related Configuration

### Recommended Production Settings

```bash
# Server with TLS
karadul server --addr=:443 --tls --cert-file=/path/to/cert.pem --key-file=/path/to/key.pem

# Node with specific listen port
karadul up --server=https://your-server:443 --listen-port=51820
```

### Firewall Rules

```bash
# Linux (iptables)
iptables -A INPUT -p udp --dport 51820 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Windows
karadul firewall setup
```

## Acknowledgments

We thank the following individuals for responsibly disclosing security issues:

- *No disclosures yet — be the first!*

## Security Updates

Subscribe to security announcements:
- Watch this repository (Releases only)
- Join [GitHub Discussions](../../discussions)

---

Last updated: March 2026
