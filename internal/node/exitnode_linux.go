//go:build linux

package node

import (
	"fmt"
	"os"
	"os/exec"
)

// EnableExitNodeLinux configures Linux IP masquerading to act as an exit node.
func EnableExitNode(outIface string) error {
	// Enable IP forwarding.
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1\n"), 0644); err != nil {
		return fmt.Errorf("enable ip_forward: %w", err)
	}

	// Add MASQUERADE rule.
	cmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-o", outIface, "-j", "MASQUERADE")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables MASQUERADE: %w: %s", err, out)
	}

	// Allow forwarded traffic.
	cmd2 := exec.Command("iptables", "-A", "FORWARD", "-i", outIface, "-j", "ACCEPT")
	if out, err := cmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables FORWARD: %w: %s", err, out)
	}

	return nil
}

// DisableExitNode removes the IP masquerading rules added by EnableExitNode.
func DisableExitNode(outIface string) error {
	_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-o", outIface, "-j", "MASQUERADE").Run()
	_ = exec.Command("iptables", "-D", "FORWARD", "-i", outIface, "-j", "ACCEPT").Run()
	return nil
}
