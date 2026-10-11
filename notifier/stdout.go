package notifier

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// SetupStdoutNotifier configures a notifier to print touch events to stdout
// using the provided template (see DefaultLogFormat).
func SetupStdoutNotifier(notifiers *sync.Map, format string) {
	renderer, err := newEventTemplate("stdout format", format)
	if err != nil {
		log.Error("Cannot initialize stdout notifier: ", err)
		return
	}

	touch := make(chan Event, 10)
	notifiers.Store("notifier/stdout", touch)

	state := newTouchState()
	for value := range touch {
		state.update(value.Message)
		line, err := renderer.render(buildTemplateData(value, state.reasons()))
		if err != nil {
			log.Error("Cannot render stdout event: ", err)
			continue
		}
		fmt.Println(line)
	}
}
