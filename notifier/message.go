package notifier

import (
	"sort"
	"strings"
)

type Message string

// All messages have a fixed length of 5 chars to simplify code on the receiving side
const (
	GPG_ON   Message = "GPG_1"
	GPG_OFF  Message = "GPG_0"
	U2F_ON   Message = "U2F_1"
	U2F_OFF  Message = "U2F_0"
	HMAC_ON  Message = "MAC_1"
	HMAC_OFF Message = "MAC_0"
)

// Reason returns the coarse event type ("gpg", "u2f" or "hmac") for the
// message, or an empty string when the message is unknown.
func (message Message) Reason() string {
	switch message {
	case GPG_ON, GPG_OFF:
		return "gpg"
	case U2F_ON, U2F_OFF:
		return "u2f"
	case HMAC_ON, HMAC_OFF:
		return "hmac"
	default:
		return ""
	}
}

// Active reports whether the message marks the start (as opposed to the end)
// of a touch request.
func (message Message) Active() bool {
	switch message {
	case GPG_ON, U2F_ON, HMAC_ON:
		return true
	default:
		return false
	}
}

type touchState struct {
	active map[string]bool
}

func newTouchState() *touchState {
	return &touchState{active: make(map[string]bool)}
}

func (state *touchState) update(message Message) (bool, bool, bool) {
	wasActive := len(state.active) > 0
	reason := message.Reason()
	if reason == "" {
		return wasActive, wasActive, false
	}
	active := message.Active()
	wasReasonActive := state.active[reason]
	if active {
		state.active[reason] = true
	} else {
		delete(state.active, reason)
	}
	return wasActive, len(state.active) > 0, wasReasonActive != active
}

func (state *touchState) reasons() string {
	reasons := make([]string, 0, len(state.active))
	for reason := range state.active {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	return strings.Join(reasons, ", ")
}
