//go:build linux

package detector

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAttributeDeviceFindsOpenHolder(t *testing.T) {
	device := filepath.Join(t.TempDir(), "fake-device")
	if err := os.WriteFile(device, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	child := exec.Command("sh", "-c", `exec 3<"$1"; sleep 10`, "sh", device)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		child.Process.Kill()
		child.Wait()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if process := attributeDevice(device, false); process != nil {
			if process.PID == os.Getpid() {
				t.Fatalf("attributed the detector process itself")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("did not attribute the process holding the device open")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestAttributeDeviceReturnsNilForUnusedDevice(t *testing.T) {
	device := filepath.Join(t.TempDir(), "unused-device")
	if err := os.WriteFile(device, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if process := attributeDevice(device, false); process != nil {
		t.Fatalf("unexpected attribution: %+v", process)
	}
}
