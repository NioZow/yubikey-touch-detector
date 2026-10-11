package notifier

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// DefaultLogFormat is the template used when logging touch events to stdout or
// to a file, unless overridden.
const DefaultLogFormat = `{{.Timestamp}} {{.Type}} active={{.Active}}{{if .Comm}} pid={{.PID}} user={{.User}} comm={{.Comm}} cmdline="{{.Cmdline}}"{{end}}{{if .Device}} device={{.Device}}{{end}}`

// TemplateData is the data made available to every template (desktop
// notifications, stdout, log file, ...). The process fields are also exposed
// flattened for convenience; they are empty when no process was attributed.
type TemplateData struct {
	// Event level fields.
	Message   string
	Type      string
	Reason    string
	Active    bool
	Time      time.Time
	Timestamp string
	Unix      int64
	Device    string
	Detail    string
	Reasons   string
	JSON      string

	// Process is the full attributed process, or nil.
	Process *Process

	// Flattened process fields (zero when no process was attributed).
	PID     int
	PPID    int
	UID     uint32
	GID     uint32
	User    string
	Comm    string
	Exe     string
	Cmdline string
	Cwd     string
}

func buildTemplateData(event Event, reasons string) TemplateData {
	message := event.Message
	timestamp := event.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	data := TemplateData{
		Message:   string(message),
		Type:      message.Reason(),
		Reason:    message.Reason(),
		Active:    message.Active(),
		Time:      timestamp,
		Timestamp: timestamp.Format(time.RFC3339),
		Unix:      timestamp.Unix(),
		Device:    event.Device,
		Detail:    event.Detail,
		Reasons:   reasons,
		JSON:      event.JSONStringWithReasons(reasons),
		Process:   event.Process,
	}
	if process := event.Process; process != nil {
		data.PID = process.PID
		data.PPID = process.PPID
		data.UID = process.UID
		data.GID = process.GID
		data.User = process.User
		data.Comm = process.Comm
		data.Exe = process.Exe
		data.Cmdline = process.CmdlineString()
		data.Cwd = process.Cwd
	}
	return data
}

var templateFunctions = template.FuncMap{
	"join": strings.Join,
	"quote": func(value string) string {
		return strconv.Quote(value)
	},
}

// eventTemplate renders a single template against TemplateData.
type eventTemplate struct {
	template *template.Template
}

func newEventTemplate(name, text string) (*eventTemplate, error) {
	parsed, err := template.New(name).Funcs(templateFunctions).Parse(text)
	if err != nil {
		return nil, err
	}
	return &eventTemplate{template: parsed}, nil
}

func (renderer *eventTemplate) render(data TemplateData) (string, error) {
	var buffer bytes.Buffer
	if err := renderer.template.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}
	return buffer.String(), nil
}
