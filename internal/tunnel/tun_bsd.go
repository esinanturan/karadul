//go:build freebsd || openbsd || netbsd

package tunnel

import (
	"fmt"
)

// CreateTUN is not implemented on BSD systems.
func CreateTUN(name string) (Device, error) {
	return nil, fmt.Errorf("TUN devices on BSD are not yet implemented; please use Linux or macOS")
}
