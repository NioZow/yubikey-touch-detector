package notifier

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitForNotifier(t *testing.T, notifiers *sync.Map, name string) chan Event {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if value, ok := notifiers.Load(name); ok {
			return value.(chan Event)
		}
		if time.Now().After(deadline) {
			t.Fatalf("notifier %q was not registered", name)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestFileNotifierWritesOnlyActiveEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history", "touches.log")
	notifiers := &sync.Map{}
	go SetupFileNotifier(notifiers, path, "{{.Type}} active={{.Active}}{{if .Comm}} comm={{.Comm}}{{end}}", false)

	touch := waitForNotifier(t, notifiers, "notifier/file")
	touch <- Event{Message: GPG_OFF, Time: time.Now()}
	touch <- Event{Message: GPG_ON, Time: time.Now(), Process: &Process{Comm: "gpg"}}

	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), "gpg active=true comm=gpg") {
			if strings.Contains(string(data), "active=false") {
				t.Fatalf("inactive event should have been logged: %q", data)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected log line was not written; got %q (err %v)", data, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestFileNotifierWritesAllEventsWhenRequested(t *testing.T) {
	path := filepath.Join(t.TempDir(), "touches.log")
	notifiers := &sync.Map{}
	go SetupFileNotifier(notifiers, path, "{{.Message}}", true)

	touch := waitForNotifier(t, notifiers, "notifier/file")
	touch <- Event{Message: HMAC_OFF, Time: time.Now()}

	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), "MAC_0") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("inactive event was not written; got %q (err %v)", data, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
