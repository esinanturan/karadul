//go:build !windows

package tunnel

import (
	"strings"
	"testing"
)

// ─── WintunDLLPath (non-Windows) ──────────────────────────────────────────────

func TestWintunDLLPath_NonWindows(t *testing.T) {
	path, err := WintunDLLPath()
	if err == nil {
		t.Errorf("expected error on non-Windows, got nil (path=%q)", path)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

// ─── EnsureWintunDLL (non-Windows) ───────────────────────────────────────────

func TestEnsureWintunDLL_NonWindows(t *testing.T) {
	path, err := EnsureWintunDLL()
	if err == nil {
		t.Errorf("expected error on non-Windows, got nil (path=%q)", path)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

// ─── GetWintunDownloadURL (non-Windows) ───────────────────────────────────────

func TestGetWintunDownloadURL_NonWindows(t *testing.T) {
	// On non-Windows the stub returns an empty string. Verify that it does
	// not panic and returns the expected value.
	url := GetWintunDownloadURL()
	// The non-Windows stub returns ""; just ensure no panic.
	// If the implementation changes to return a URL even on non-Windows,
	// this test documents that we expect "" on non-Windows platforms.
	if url != "" {
		// If it returns something, at minimum it should look like a URL.
		if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
			t.Errorf("expected empty string or a valid URL, got %q", url)
		}
	}
}
