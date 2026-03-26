//go:build freebsd || openbsd || netbsd

package dns

import "fmt"

// Override is not implemented on BSD systems.
func Override(listenAddr string) (restore func() error, err error) {
	return nil, fmt.Errorf("DNS override on BSD is not yet implemented")
}

