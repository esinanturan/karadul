//go:build windows

package tunnel

import (
	"fmt"
	"net"
)

// windowsTUN is a stub for Windows. Full Wintun support is a future milestone.
type windowsTUN struct {
	name string
}

// CreateTUN creates a TUN device on Windows (stub — not yet implemented).
func CreateTUN(name string) (Device, error) {
	return nil, fmt.Errorf("windows TUN not implemented: install Wintun (https://www.wintun.net) and rebuild with wintun support")
}

type stubTUN struct{}

func (s *stubTUN) Name() string                          { return "karadul0" }
func (s *stubTUN) Read(_ []byte) (int, error)            { return 0, fmt.Errorf("not implemented") }
func (s *stubTUN) Write(_ []byte) (int, error)           { return 0, fmt.Errorf("not implemented") }
func (s *stubTUN) MTU() int                              { return 1420 }
func (s *stubTUN) SetMTU(_ int) error                    { return fmt.Errorf("not implemented") }
func (s *stubTUN) SetAddr(_ net.IP, _ int) error         { return fmt.Errorf("not implemented") }
func (s *stubTUN) AddRoute(_ *net.IPNet) error           { return fmt.Errorf("not implemented") }
func (s *stubTUN) Close() error                          { return nil }
