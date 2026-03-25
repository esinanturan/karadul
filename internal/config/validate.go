package config

import (
	"fmt"
	"net"
	"strings"
)

// ValidateNodeConfig checks that cfg is internally consistent.
func ValidateNodeConfig(cfg *NodeConfig) error {
	if cfg.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}
	if !strings.HasPrefix(cfg.ServerURL, "http://") && !strings.HasPrefix(cfg.ServerURL, "https://") {
		return fmt.Errorf("server_url must start with http:// or https://")
	}
	if cfg.ListenPort < 0 || cfg.ListenPort > 65535 {
		return fmt.Errorf("listen_port must be 0–65535")
	}
	for _, cidr := range cfg.AdvertiseRoutes {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid advertise_route %q: %w", cidr, err)
		}
	}
	if cfg.DNS.Upstream != "" {
		if _, _, err := net.SplitHostPort(cfg.DNS.Upstream); err != nil {
			return fmt.Errorf("invalid dns.upstream %q: %w", cfg.DNS.Upstream, err)
		}
	}
	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return err
	}
	return nil
}

// ValidateServerConfig checks that cfg is internally consistent.
func ValidateServerConfig(cfg *ServerConfig) error {
	if cfg.Addr == "" {
		return fmt.Errorf("addr is required")
	}
	if _, _, err := net.SplitHostPort(cfg.Addr); err != nil {
		return fmt.Errorf("invalid addr %q: %w", cfg.Addr, err)
	}
	if _, _, err := net.ParseCIDR(cfg.Subnet); err != nil {
		return fmt.Errorf("invalid subnet %q: %w", cfg.Subnet, err)
	}
	if cfg.ApprovalMode != "auto" && cfg.ApprovalMode != "manual" {
		return fmt.Errorf("approval_mode must be 'auto' or 'manual'")
	}
	if cfg.TLS.Enabled {
		if !cfg.TLS.SelfSigned && (cfg.TLS.CertFile == "" || cfg.TLS.KeyFile == "") {
			return fmt.Errorf("tls.cert_file and tls.key_file are required when tls is enabled without self_signed")
		}
	}
	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return err
	}
	return nil
}

func validateLogLevel(level string) error {
	switch level {
	case "", "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("log_level must be one of: debug, info, warn, error")
	}
}
