package notifier

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// SetupDebugNotifier configures a notifier to log all touch events
func SetupDebugNotifier(notifiers *sync.Map) {
	touch := make(chan Event, 10)
	notifiers.Store("notifier/debug", touch)

	state := newTouchState()
	for value := range touch {
		state.update(value.Message)
		log.Debug("[notifiers/debug] ", value.JSONStringWithReasons(state.reasons()))
	}
}
