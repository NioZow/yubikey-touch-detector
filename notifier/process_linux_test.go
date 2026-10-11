//go:build linux

package notifier

import (
	"os"
	"testing"
)

func TestNewProcessReadsCurrentProcess(t *testing.T) {
	process := NewProcess(os.Getpid())
	if process == nil {
		t.Fatal("NewProcess returned nil for the current process")
	}
	if process.PID != os.Getpid() {
		t.Fatalf("PID = %d, want %d", process.PID, os.Getpid())
	}
	if process.Comm == "" || process.Exe == "" {
		t.Fatalf("expected comm and exe to be populated: %+v", process)
	}
}
