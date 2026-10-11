package notifier

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventJSONIncludesProcessAndDerivedFields(t *testing.T) {
	event := Event{
		Message: GPG_ON,
		Process: &Process{PID: 4242, Comm: "gpg", Cmdline: []string{"gpg", "--sign"}},
		Device:  "/dev/hidraw0",
	}
	encoded := event.JSONString()

	var decoded struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Active  bool   `json:"active"`
		Process struct {
			PID     int      `json:"pid"`
			Comm    string   `json:"comm"`
			Cmdline []string `json:"cmdline"`
		} `json:"process"`
		Device string `json:"device"`
	}
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("invalid JSON %q: %v", encoded, err)
	}
	if decoded.Message != "GPG_1" || decoded.Type != "gpg" || !decoded.Active {
		t.Fatalf("unexpected event fields: %+v", decoded)
	}
	if decoded.Process.PID != 4242 || decoded.Process.Comm != "gpg" {
		t.Fatalf("unexpected process fields: %+v", decoded.Process)
	}
	if decoded.Device != "/dev/hidraw0" {
		t.Fatalf("unexpected device: %q", decoded.Device)
	}
}

func TestBuildTemplateDataFlattensProcess(t *testing.T) {
	event := Event{
		Message: U2F_ON,
		Process: &Process{PID: 7, UID: 1000, User: "user", Comm: "ssh", Cmdline: []string{"ssh", "host"}},
	}
	data := buildTemplateData(event, "u2f")
	if data.Type != "u2f" || data.Reason != "u2f" || !data.Active {
		t.Fatalf("unexpected data: %+v", data)
	}
	if data.PID != 7 || data.User != "user" || data.Comm != "ssh" || data.Cmdline != "ssh host" {
		t.Fatalf("process not flattened: %+v", data)
	}
	if data.Reasons != "u2f" {
		t.Fatalf("unexpected reasons: %q", data.Reasons)
	}
}

func TestDefaultLogFormatRendersWithoutProcess(t *testing.T) {
	renderer, err := newEventTemplate("test", DefaultLogFormat)
	if err != nil {
		t.Fatal(err)
	}
	line, err := renderer.render(buildTemplateData(Event{Message: HMAC_OFF}, ""))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, "hmac") || !strings.Contains(line, "active=false") {
		t.Fatalf("unexpected line: %q", line)
	}
	if strings.Contains(line, "pid=") {
		t.Fatalf("line should not mention a pid without a process: %q", line)
	}
}
