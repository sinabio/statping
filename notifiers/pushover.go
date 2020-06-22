package notifiers

import (
	"fmt"
	"github.com/statping/statping/types/failures"
	"github.com/statping/statping/types/notifications"
	"github.com/statping/statping/types/notifier"
	"github.com/statping/statping/types/services"
	"github.com/statping/statping/utils"
	"net/url"
	"strings"
	"time"
)

const (
	pushoverUrl = "https://api.pushover.net/1/messages.json"
)

var _ notifier.Notifier = (*pushover)(nil)

type pushover struct {
	*notifications.Notification
}

func (t *pushover) Select() *notifications.Notification {
	return t.Notification
}

var Pushover = &pushover{&notifications.Notification{
	Method:      "pushover",
	Title:       "Pushover",
	Description: "Use Pushover to receive push notifications. You will need to create a <a href=\"https://pushover.net/apps/build\">New Application</a> on Pushover before using this notifier.",
	Author:      "Hunter Long",
	AuthorUrl:   "https://github.com/hunterlong",
	Icon:        "fa dot-circle",
	Delay:       time.Duration(10 * time.Second),
	Limits:      60,
	SuccessData: `Your service '{{.Service.Name}}' is currently online!`,
	FailureData: `Your service '{{.Service.Name}}' is currently offline!`,
	DataType:    "text",
	Form: []notifications.NotificationForm{{
		Type:        "text",
		Title:       "User Token",
		Placeholder: "Insert your Pushover User Token",
		DbField:     "api_key",
		Required:    true,
	}, {
		Type:        "text",
		Title:       "Application API Key",
		Placeholder: "Create an Application and insert the API Key here",
		DbField:     "api_secret",
		Required:    true,
	}, {
		Type:        "list",
		Title:       "Priority",
		Placeholder: "Set the notification priority level",
		DbField:     "Var1",
		Required:    true,
		ListOptions: []string{"Lowest", "Low", "Normal", "High", "Emergency"},
	}, {
		Type:        "list",
		Title:       "Notification Sound",
		Placeholder: "Choose a sound for this Pushover notification",
		DbField:     "Var2",
		Required:    true,
		ListOptions: []string{"none", "pushover", "bike", "bugle", "cashregister", "classical", "cosmic", "falling", "gamelan", "incoming", "intermissioon", "magic", "mechanical", "painobar", "siren", "spacealarm", "tugboat", "alien", "climb", "persistent", "echo", "updown"},
	},
	}},
}

// Send will send a HTTP Post to the Pushover API. It accepts type: string
func (t *pushover) sendMessage(message string) (string, error) {
	v := url.Values{}
	v.Set("token", t.ApiSecret)
	v.Set("user", t.ApiKey)
	v.Set("message", message)
	rb := strings.NewReader(v.Encode())

	content, _, err := utils.HttpRequest(pushoverUrl, "POST", "application/x-www-form-urlencoded", nil, rb, time.Duration(10*time.Second), true, nil)
	if err != nil {
		return "", err
	}
	return string(content), err
}

// OnFailure will trigger failing service
func (t *pushover) OnFailure(s *services.Service, f *failures.Failure) (string, error) {
	message := ReplaceVars(t.FailureData, s, f)
	out, err := t.sendMessage(message)
	return out, err
}

// OnSuccess will trigger successful service
func (t *pushover) OnSuccess(s *services.Service) (string, error) {
	message := ReplaceVars(t.SuccessData, s, nil)
	out, err := t.sendMessage(message)
	return out, err
}

// OnTest will test the Pushover SMS messaging
func (t *pushover) OnTest() (string, error) {
	example := services.Example(true)
	msg := fmt.Sprintf("Testing the Pushover Notifier, Your service '%s' is currently offline! Error: %s", example.Name, exampleFailure.Issue)
	content, err := t.sendMessage(msg)
	return content, err
}

// OnSave will trigger when this notifier is saved
func (t *pushover) OnSave() (string, error) {
	return "", nil
}
