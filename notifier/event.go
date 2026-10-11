package notifier

import (
	"encoding/json"
	"strings"
	"time"
)

// Process describes the process that has been attributed to a touch request.
//
// Attribution is best-effort: for operations passing through a unix socket
// (GPG agent, SSH agent) the peer credentials provide an exact PID, while for
// raw HID access (U2F/HMAC) the process holding the device open is discovered
// by scanning /proc. When the requesting process cannot be determined (for
// example a root-owned process when running unprivileged), Process is nil.
type Process struct {
	PID     int      `json:"pid"`
	PPID    int      `json:"ppid,omitempty"`
	UID     uint32   `json:"uid,omitempty"`
	GID     uint32   `json:"gid,omitempty"`
	User    string   `json:"user,omitempty"`
	Comm    string   `json:"comm,omitempty"`
	Exe     string   `json:"exe,omitempty"`
	Cmdline []string `json:"cmdline,omitempty"`
	Cwd     string   `json:"cwd,omitempty"`
	// Source records how the process was attributed, e.g. "peercred" or
	// "proc-scan".
	Source string `json:"source,omitempty"`
}

// CmdlineString returns the process command line joined by spaces.
func (process *Process) CmdlineString() string {
	if process == nil {
		return ""
	}
	return strings.Join(process.Cmdline, " ")
}

// Event is a single touch state transition, optionally carrying the process
// that triggered it.
type Event struct {
	// Message is the legacy 5 byte token, e.g. "GPG_1".
	Message Message `json:"message"`
	// Time is when the transition was observed.
	Time time.Time `json:"time"`
	// Process is the process attributed to the transition, if known.
	Process *Process `json:"process,omitempty"`
	// Device is the hidraw device involved, when applicable.
	Device string `json:"device,omitempty"`
	// Detail carries any extra source specific information, e.g. a keygrip.
	Detail string `json:"detail,omitempty"`
}

// NewEvent builds an Event for the given message, stamping it with the
// current time.
func NewEvent(message Message) Event {
	return Event{Message: message, Time: time.Now()}
}

// NewEventWithProcess builds an Event carrying a process attribution.
func NewEventWithProcess(message Message, process *Process) Event {
	return Event{Message: message, Time: time.Now(), Process: process}
}

// Type returns the coarse event type: "gpg", "u2f", "hmac" or "" if unknown.
func (event Event) Type() string {
	return event.Message.Reason()
}

// Active reports whether the event represents a touch starting (as opposed to
// ending).
func (event Event) Active() bool {
	return event.Message.Active()
}

type eventJSON struct {
	Message string   `json:"message"`
	Type    string   `json:"type,omitempty"`
	Reason  string   `json:"reason,omitempty"`
	Active  bool     `json:"active"`
	Time    string   `json:"time"`
	Unix    int64    `json:"unix"`
	Reasons string   `json:"reasons,omitempty"`
	Device  string   `json:"device,omitempty"`
	Detail  string   `json:"detail,omitempty"`
	Process *Process `json:"process,omitempty"`
}

func (event Event) view(reasons string) eventJSON {
	timestamp := event.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	return eventJSON{
		Message: string(event.Message),
		Type:    event.Message.Reason(),
		Reason:  event.Message.Reason(),
		Active:  event.Message.Active(),
		Time:    timestamp.Format(time.RFC3339Nano),
		Unix:    timestamp.Unix(),
		Reasons: reasons,
		Device:  event.Device,
		Detail:  event.Detail,
		Process: event.Process,
	}
}

// JSONString returns the event serialized as a single line JSON object.
func (event Event) JSONString() string {
	return event.JSONStringWithReasons("")
}

// JSONStringWithReasons returns the event serialized as JSON, additionally
// embedding the currently active touch reasons.
func (event Event) JSONStringWithReasons(reasons string) string {
	encoded, err := json.Marshal(event.view(reasons))
	if err != nil {
		return ""
	}
	return string(encoded)
}

// MarshalJSON implements json.Marshaler so that encoding an Event anywhere
// produces the same rich schema used on the socket and in the logs.
func (event Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(event.view(""))
}
