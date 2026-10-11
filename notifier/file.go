package notifier

import (
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

// SetupFileNotifier appends every touch event to a file using the provided
// template. When all is false only events starting a touch (Active) are
// written, which is the common case for a touch history log.
func SetupFileNotifier(notifiers *sync.Map, path string, format string, all bool) {
	renderer, err := newEventTemplate("log format", format)
	if err != nil {
		log.Error("Cannot initialize file notifier: ", err)
		return
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Error("Cannot create directory for touch log: ", err)
			return
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Error("Cannot open touch log file: ", err)
		return
	}
	defer file.Close()

	touch := make(chan Event, 10)
	notifiers.Store("notifier/file", touch)

	state := newTouchState()
	for value := range touch {
		state.update(value.Message)
		if !all && !value.Active() {
			continue
		}
		line, err := renderer.render(buildTemplateData(value, state.reasons()))
		if err != nil {
			log.Error("Cannot render touch log line: ", err)
			continue
		}
		if _, err := file.WriteString(line + "\n"); err != nil {
			log.Error("Cannot write touch log line: ", err)
		}
	}
}
