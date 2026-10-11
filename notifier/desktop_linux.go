package notifier

import "sync"

func SetupDesktopNotifier(notifiers *sync.Map, title, body, icon string) {
	setupLibnotifyNotifier(notifiers, title, body, icon)
}
