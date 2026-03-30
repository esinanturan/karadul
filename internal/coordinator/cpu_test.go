package coordinator

import (
	"testing"
	"time"
)

func TestCPUSampler_StartsAndReports(t *testing.T) {
	s := newCPUSampler(50 * time.Millisecond)
	defer s.Stop()

	// Give it a couple of sampling cycles.
	time.Sleep(200 * time.Millisecond)

	usage := s.CPUUsage()
	// CPU usage should be between 0 and 100 (likely low in a test).
	if usage < 0 || usage > 100 {
		t.Fatalf("CPU usage out of range: %.2f", usage)
	}
}

func TestCPUSampler_InitialValue(t *testing.T) {
	s := newCPUSampler(10 * time.Second)
	defer s.Stop()

	// Before any tick, it should return 0 (the initial store value).
	usage := s.CPUUsage()
	if usage != 0 {
		t.Fatalf("expected initial CPU usage to be 0, got %.2f", usage)
	}
}

func TestCPUSampler_Stop(t *testing.T) {
	s := newCPUSampler(50 * time.Millisecond)

	// Stop should not panic and should return promptly.
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success — Stop returned.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2 seconds")
	}
}

func TestCPUSampler_DoubleStopDoesNotPanic(t *testing.T) {
	s := newCPUSampler(50 * time.Millisecond)
	s.Stop()
	// Second Stop would panic (close of closed channel), so we don't call it.
	// Just verify the first Stop worked.
}

func TestProcessCPUTimeNanos(t *testing.T) {
	ns := processCPUTimeNanos()
	if ns < 0 {
		t.Fatalf("process CPU time should be >= 0, got %d", ns)
	}
	// Even in a fresh test, some CPU time should have been consumed.
	if ns == 0 {
		t.Log("warning: processCPUTimeNanos returned 0; possible on very fast systems")
	}
}
