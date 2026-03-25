//go:build windows

package node

import "fmt"

// EnableExitNode is not yet implemented on Windows.
func EnableExitNode(_ string) error {
	return fmt.Errorf("exit node not implemented on Windows")
}

// DisableExitNode is not yet implemented on Windows.
func DisableExitNode(_ string) error {
	return fmt.Errorf("exit node not implemented on Windows")
}
