//go:build darwin

package node

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const pfConf = "/etc/pf.karadul.conf"

// EnableExitNode configures macOS pf NAT to act as an exit node.
func EnableExitNode(outIface string) error {
	// Enable IP forwarding via sysctl.
	if err := exec.Command("sysctl", "-w", "net.inet.ip.forwarding=1").Run(); err != nil {
		return fmt.Errorf("sysctl ip.forwarding: %w", err)
	}

	// Write a pf anchor config.
	conf := fmt.Sprintf("nat on %s from !(%s) to any -> (%s)\n", outIface, outIface, outIface)
	if err := os.WriteFile(pfConf, []byte(conf), 0600); err != nil {
		return fmt.Errorf("write pf conf: %w", err)
	}

	// Load the anchor.
	if out, err := exec.Command("pfctl", "-f", pfConf).CombinedOutput(); err != nil {
		return fmt.Errorf("pfctl load: %w: %s", err, out)
	}
	// Enable pf.
	if out, err := exec.Command("pfctl", "-e").CombinedOutput(); err != nil {
		// Ignore "pf already enabled".
		if !strings.Contains(string(out), "already enabled") {
			return fmt.Errorf("pfctl enable: %w: %s", err, out)
		}
	}
	return nil
}

// DisableExitNode removes the pf NAT rules.
func DisableExitNode(outIface string) error {
	_ = os.Remove(pfConf)
	_ = exec.Command("pfctl", "-d").Run()
	return nil
}
