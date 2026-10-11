package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/maximbaz/yubikey-touch-detector/notifier"
)

// Override with -ldflags "-X main.version=xxx" when compiling not from a git-archive tarball
var version = "$Format:%(describe)$"

func main() {
	truthyValues := map[string]bool{"true": true, "yes": true, "1": true}

	envVerbose := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_VERBOSE"))]
	envNotify := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_NOTIFY"))]
	envLibnotify := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_LIBNOTIFY"))]
	envStdout := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_STDOUT"))]
	envNosocket := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_NOSOCKET"))]
	envDbus := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_DBUS"))]
	envLogAll := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_LOG_ALL"))]
	envSocketJSON := truthyValues[strings.ToLower(os.Getenv("YUBIKEY_TOUCH_DETECTOR_SOCKET_JSON"))]
	notificationTitle := notifier.DefaultNotificationTitle
	if value, found := os.LookupEnv("YUBIKEY_TOUCH_DETECTOR_NOTIFY_TITLE"); found {
		notificationTitle = value
	}
	notificationBody := notifier.DefaultNotificationBody
	if value, found := os.LookupEnv("YUBIKEY_TOUCH_DETECTOR_NOTIFY_BODY"); found {
		notificationBody = value
	}
	stdoutFormat := "{{.JSON}}"
	if value, found := os.LookupEnv("YUBIKEY_TOUCH_DETECTOR_STDOUT_FORMAT"); found {
		stdoutFormat = value
	}
	logFile := os.Getenv("YUBIKEY_TOUCH_DETECTOR_LOG_FILE")
	logFormat := notifier.DefaultLogFormat
	if value, found := os.LookupEnv("YUBIKEY_TOUCH_DETECTOR_LOG_FORMAT"); found {
		logFormat = value
	}

	var version bool
	var verbose bool
	var desktopNotify bool
	var stdout bool
	var nosocket bool
	var dbus bool
	var logAll bool
	var socketJSON bool

	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&verbose, "v", envVerbose, "enable debug logging")
	flag.BoolVar(&desktopNotify, "notify", envNotify || envLibnotify, "show desktop notifications")
	flag.BoolVar(&desktopNotify, "libnotify", envNotify || envLibnotify, "deprecated alias for --notify")
	flag.StringVar(&notificationTitle, "notify-title", notificationTitle, "desktop notification title template")
	flag.StringVar(&notificationBody, "notify-body", notificationBody, "desktop notification body template")
	flag.BoolVar(&stdout, "stdout", envStdout, "print touch events to stdout")
	flag.StringVar(&stdoutFormat, "stdout-format", stdoutFormat, "stdout event template (see docs for available fields)")
	flag.BoolVar(&nosocket, "no-socket", envNosocket, "disable unix socket notifier")
	flag.BoolVar(&dbus, "dbus", envDbus, "enable dbus server for IPC")
	flag.BoolVar(&socketJSON, "socket-json", envSocketJSON, "emit rich JSON events on the unix socket instead of legacy 5 byte tokens")
	flag.StringVar(&logFile, "log-file", logFile, "append touch events to this file (empty disables logging)")
	flag.StringVar(&logFormat, "log-format", logFormat, "log file event template (see docs for available fields)")
	flag.BoolVar(&logAll, "log-all", envLogAll, "log both touch start and end events (default: only touch start)")
	flag.Parse()

	if version {
		fmt.Println("YubiKey touch detector version:", appVersion())
		os.Exit(0)
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	libnotifyFlagUsed := false
	flag.Visit(func(value *flag.Flag) {
		libnotifyFlagUsed = libnotifyFlagUsed || value.Name == "libnotify"
	})
	if libnotifyFlagUsed || envLibnotify {
		log.Warn("--libnotify and YUBIKEY_TOUCH_DETECTOR_LIBNOTIFY are deprecated; use --notify or YUBIKEY_TOUCH_DETECTOR_NOTIFY")
	}
	log.Debug("Starting YubiKey touch detector")

	exits := &sync.Map{}
	go setupExitSignalWatch(exits)

	notifiers := &sync.Map{}

	if verbose {
		go notifier.SetupDebugNotifier(notifiers)
	}
	if !nosocket {
		go notifier.SetupUnixSocketNotifier(notifiers, exits, socketJSON)
	}
	if desktopNotify {
		go notifier.SetupDesktopNotifier(notifiers, notificationTitle, notificationBody)
	}
	if stdout {
		go notifier.SetupStdoutNotifier(notifiers, stdoutFormat)
	}
	if logFile != "" {
		go notifier.SetupFileNotifier(notifiers, logFile, logFormat, logAll)
	}
	if dbus {
		go notifier.SetupDbusNotifier(notifiers)
	}

	initDetectors(notifiers, exits)

	wait := make(chan bool)
	<-wait
}

func setupExitSignalWatch(exits *sync.Map) {
	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt, syscall.SIGTERM)

	<-exitSignal
	println()

	exits.Range(func(_, v interface{}) bool {
		exit := v.(chan bool)
		exit <- true // Notify exit watcher
		<-exit       // Wait for confirmation
		return true
	})

	log.Debug("Stopping YubiKey touch detector")
	os.Exit(0)
}

func appVersion() string {
	if strings.HasPrefix(version, "$") {
		return "unknown"
	}
	return version
}
