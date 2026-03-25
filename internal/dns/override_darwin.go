//go:build darwin

package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

// Override sets the system DNS resolver on macOS using networksetup.
// It saves the original DNS settings so they can be restored.
func Override(listenAddr string) (restore func() error, err error) {
	// Extract just the IP from listenAddr.
	ip := listenAddr
	for i := len(listenAddr) - 1; i >= 0; i-- {
		if listenAddr[i] == ':' {
			ip = listenAddr[:i]
			break
		}
	}

	// Get the active network service.
	service, err := activeNetworkService()
	if err != nil {
		return nil, fmt.Errorf("get network service: %w", err)
	}

	// Save original DNS.
	out, err := exec.Command("networksetup", "-getdnsservers", service).Output()
	if err != nil {
		return nil, fmt.Errorf("get dns servers: %w", err)
	}
	original := strings.TrimSpace(string(out))

	// Set our resolver.
	if err := exec.Command("networksetup", "-setdnsservers", service, ip).Run(); err != nil {
		return nil, fmt.Errorf("set dns servers: %w", err)
	}

	restore = func() error {
		// If original was "There aren't any DNS Servers set", clear it.
		if strings.Contains(original, "aren't any") || original == "" {
			return exec.Command("networksetup", "-setdnsservers", service, "empty").Run()
		}
		servers := strings.Fields(original)
		args := append([]string{"-setdnsservers", service}, servers...)
		return exec.Command("networksetup", args...).Run()
	}
	return restore, nil
}

// activeNetworkService returns the first active network service name.
func activeNetworkService() (string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "*") || strings.Contains(line, "asterisk") {
			continue
		}
		// Skip the header line.
		if strings.Contains(line, "order") {
			continue
		}
		return line, nil
	}
	return "Wi-Fi", nil // fallback
}
