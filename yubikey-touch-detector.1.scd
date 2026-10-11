yubikey-touch-detector(1)

# NAME

yubikey-touch-detector - detect when a YubiKey is waiting for a touch

# SYNOPSIS

*yubikey-touch-detector* [options...]

# OPTIONS

*-notify*
	Show desktop notifications.

*-notify-title* _title_
	Set the desktop notification title template. Defaults to "YubiKey is waiting for a touch".

*-notify-body* _body_
	Set the desktop notification body template. Defaults to "Touch your YubiKey to continue ({{.Reasons}}).{{if .Comm}}\nRequested by {{.Comm}} ({{.Cmdline}}){{end}}".

*-notify-icon* _name_
	Set the freedesktop icon theme name used for the desktop notification. Defaults to "yubikey-touch-detector" (for example, use "pgp-keys" for a key icon).

*-no-socket*
	Disable unix socket notifier.

*-socket-json*
	Emit rich JSON events on the unix socket instead of the legacy fixed-length 5 byte tokens. See *SOCKET EVENTS*.

*-stdout*
	Print touch events to stdout.

*-stdout-format* _template_
	Template used by *-stdout*. Defaults to _{{.JSON}}_ (one JSON object per line). See *TEMPLATE FIELDS*.

*-log-file* _path_
	Append touch events to _path_. The parent directory is created if needed. Empty (the default) disables file logging.

*-log-format* _template_
	Template used by *-log-file*. Defaults to a readable line containing the timestamp, type, and process information. See *TEMPLATE FIELDS*.

*-log-all*
	Log both the start and the end of touch requests. By default only events that start a touch are written, which is usually what a touch history wants.

*-dbus*
	Support notifications via dbus SessionBus.

*-v*
	Enable debug logging.

*-version*
	Print version information and exit.

# DESCRIPTION

*yubikey-touch-detector* detects when a YubiKey is waiting for a touch
configuration. It is designed to be integrated with other UI components to
display a visible indicator.

In addition to the touch reason, the detector attempts to identify the process
that triggered the request. For operations that pass through a unix socket
(the GPG agent and the SSH agent) the peer credentials provide an exact PID.
For raw HID access (U2F and HMAC) the process holding the device open is
discovered by scanning _/proc_, which only sees the current user's processes.
When the requesting process cannot be determined, the process fields are left
empty.

# TEMPLATE FIELDS

The notification, stdout and log templates use Go template syntax. The
following fields are available:

_Event fields:_
	_.Message_ (legacy token, e.g. _GPG_1_), _.Type_ ( _gpg_, _u2f_ or _hmac_), _.Reason_ (alias of _.Type_), _.Active_ (boolean), _.Time_, _.Timestamp_ (RFC3339), _.Unix_, _.Device_, _.Detail_, _.Reasons_ (sorted list of currently active reasons) and _.JSON_ (the full event as JSON).

_Process fields:_
	_.Process_ (the full process object) and the flattened shortcuts _.PID_, _.PPID_, _.UID_, _.GID_, _.User_, _.Comm_, _.Exe_, _.Cmdline_ (command line joined by spaces) and _.Cwd_. These are empty when no process could be attributed.

The _.Process_ object exposes the raw fields as well, e.g. _{{.Process.Cmdline}}_ is a list and _{{.Process.CmdlineString}}_ joins it.

Example notification body:

	Requested by {{.Comm}} ({{.Cmdline}})

Example log line template:

	{{.Timestamp}} {{.Type}} pid={{.PID}} {{.Cmdline}}

# ENVIRONMENT

_YUBIKEY_TOUCH_DETECTOR_VERBOSE_
	Equivalent to specifying *-v*.

_YUBIKEY_TOUCH_DETECTOR_NOTIFY_
	Equivalent to specifying *-notify*.

_YUBIKEY_TOUCH_DETECTOR_NOTIFY_TITLE_
	Equivalent to specifying *-notify-title*.

_YUBIKEY_TOUCH_DETECTOR_NOTIFY_BODY_
	Equivalent to specifying *-notify-body*.

_YUBIKEY_TOUCH_DETECTOR_NOTIFY_ICON_
	Equivalent to specifying *-notify-icon*.

_YUBIKEY_TOUCH_DETECTOR_STDOUT_
	Equivalent to specifying *-stdout*.

_YUBIKEY_TOUCH_DETECTOR_STDOUT_FORMAT_
	Equivalent to specifying *-stdout-format*.

_YUBIKEY_TOUCH_DETECTOR_NOSOCKET_
	Equivalent to specifying *-no-socket*.

_YUBIKEY_TOUCH_DETECTOR_SOCKET_JSON_
	Equivalent to specifying *-socket-json*.

_YUBIKEY_TOUCH_DETECTOR_LOG_FILE_
	Equivalent to specifying *-log-file*.

_YUBIKEY_TOUCH_DETECTOR_LOG_FORMAT_
	Equivalent to specifying *-log-format*.

_YUBIKEY_TOUCH_DETECTOR_LOG_ALL_
	Equivalent to specifying *-log-all*.

_YUBIKEY_TOUCH_DETECTOR_DBUS_
	Equivalent to specifying *-dbus*.

# FILES

_$XDG_RUNTIME_DIR/yubikey-touch-detector.socket_
	The socket exposing events on Linux.

_$TMPDIR/yubikey-touch-detector.socket_
	The socket exposing events on macOS.

# EVENTS

By default the following fixed-length 5 byte tokens are emitted over the
socket:

_GPG_1_
	When a GPG operation starts waiting for a touch.

_GPG_0_
	When a GPG operation stops waiting for a touch.

_U2F_1_
	When a U2F operation starts waiting for a touch.

_U2F_0_
	When a U2F operation stops waiting for a touch.

_MAC_1_
	When a HMAC operation starts waiting for a touch.

_MAC_0_
	When a HMAC operation stops waiting for a touch.

# SOCKET EVENTS

With *-socket-json* each event is emitted as a single JSON object followed by a
newline:

```
{"message":"GPG_1","type":"gpg","reason":"gpg","active":true,"time":"2026-01-02T15:04:05.123456789Z","unix":1767366245,"reasons":"gpg","process":{"pid":1234,"ppid":1,"uid":1000,"user":"user","comm":"gpg","exe":"/usr/bin/gpg","cmdline":["gpg","--sign"],"cwd":"/home/user","source":"peercred"}}
```

Unknown or empty fields are omitted.

The *-dbus* interface additionally exposes a _LastEvent_ property containing the
same JSON document for the most recent event.

# SEE ALSO

ykman, pam_u2f(8)

# LICENCE

yubikey-touch-detector is distributed under the ISC licence.
