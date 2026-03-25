//go:build windows

package dns

import "fmt"

// Override is not yet implemented on Windows.
func Override(_ string) (func() error, error) {
	return nil, fmt.Errorf("DNS override not implemented on Windows")
}
