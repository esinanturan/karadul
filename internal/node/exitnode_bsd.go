//go:build freebsd || openbsd || netbsd

package node

import "fmt"

// EnableExitNode is not implemented on BSD systems.
func EnableExitNode(vip string) error {
	return fmt.Errorf("exit node on BSD is not yet implemented")
}

// DisableExitNode is not implemented on BSD systems.
func DisableExitNode() error {
	return fmt.Errorf("exit node on BSD is not yet implemented")
}
