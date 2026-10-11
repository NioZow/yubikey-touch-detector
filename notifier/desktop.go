package notifier

import (
	"bytes"
	"fmt"
	"text/template"
)

const DefaultNotificationTitle = "YubiKey is waiting for a touch"
const DefaultNotificationBody = "Touch your YubiKey to continue ({{.Reasons}}).{{if .Comm}}\nRequested by {{.Comm}} ({{.Cmdline}}){{end}}"

type notificationTemplates struct {
	title *template.Template
	body  *template.Template
}

func newNotificationTemplates(title, body string) (*notificationTemplates, error) {
	titleTemplate, err := template.New("notification title").Funcs(templateFunctions).Parse(title)
	if err != nil {
		return nil, fmt.Errorf("invalid notification title template: %w", err)
	}
	bodyTemplate, err := template.New("notification body").Funcs(templateFunctions).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("invalid notification body template: %w", err)
	}
	return &notificationTemplates{title: titleTemplate, body: bodyTemplate}, nil
}

func (templates *notificationTemplates) render(data TemplateData) (string, string, error) {
	var title bytes.Buffer
	if err := templates.title.Execute(&title, data); err != nil {
		return "", "", fmt.Errorf("rendering notification title: %w", err)
	}
	var body bytes.Buffer
	if err := templates.body.Execute(&body, data); err != nil {
		return "", "", fmt.Errorf("rendering notification body: %w", err)
	}
	return title.String(), body.String(), nil
}
