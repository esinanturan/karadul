---
name: Beta Test Report
about: Report issues or feedback for beta releases
title: '[BETA] '
labels: ['beta', 'needs-triage']
assignees: ''

---

## Beta Version
<!-- Which beta version are you testing? -->
- Version: <!-- e.g., v0.1.0-beta.1 -->

## Platform
<!-- Platform you are testing on -->
- [ ] Linux
- [ ] macOS
- [ ] Windows
- [ ] FreeBSD
- [ ] OpenBSD

**System Information:**
- OS Version: <!-- e.g., Windows 11 23H2, macOS 14.3, Ubuntu 22.04 -->
- Architecture: <!-- e.g., amd64, arm64 -->

## Features Tested
<!-- Which features did you test? -->
- [ ] Installation (binary download)
- [ ] Windows: Wintun driver installation
- [ ] Windows: Firewall setup
- [ ] Starting coordination server
- [ ] Joining mesh as a node
- [ ] Peer discovery
- [ ] NAT traversal
- [ ] DERP relay fallback
- [ ] Exit node
- [ ] MagicDNS
- [ ] Docker container

## Results

### ✅ Working
<!-- Features that worked without issues -->

### ❌ Issues
<!-- Issues encountered -->

**Error Message:**
```
Paste error message here
```

**Steps to Reproduce:**
1.
2.
3.

**Expected Behavior:**
<!-- What did you expect to happen? -->

**Actual Behavior:**
<!-- What actually happened? -->

## Logs
<!-- If applicable, add log output -->
```bash
# Linux/macOS
karadul up --log-level=debug 2>&1 | tee karadul.log

# Windows PowerShell
karadul up --log-level=debug 2>&1 | Tee-Object karadul.log
```

## Additional Information
<!-- Screenshots, additional notes -->

---

**Note:** Beta releases are for testing purposes only. Wait for stable releases for production use.
