//go:build freebsd || openbsd || netbsd

package firewall

import "fmt"

// Setup configures firewall rules on BSD systems using pf.
// This is a placeholder for future implementation.
func Setup(exePath string) error {
	return fmt.Errorf("firewall auto-configuration on BSD is not yet implemented; please configure pf manually")
}

// Remove deletes the firewall rules.
func Remove() error {
	return fmt.Errorf("firewall auto-configuration on BSD is not yet implemented")
}

// Check returns true if the firewall rules are configured.
func Check() bool {
	// On BSD, we assume the admin has configured the firewall
	return true
}

// AllowPort adds a firewall rule for a specific port.
func AllowPort(port int, protocol string) error {
	return fmt.Errorf("firewall management on BSD must be done manually via pf")
}

// RemovePort removes a port-specific firewall rule.
func RemovePort(port int, protocol string) error {
	return fmt.Errorf("firewall management on BSD must be done manually via pf")
}
