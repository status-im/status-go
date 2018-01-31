package notification

import "github.com/NaySoftware/go-fcm"

// Payload data of message.
type Payload struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Sound string `json:"sound,omitempty"`
	Badge string `json:"badge,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Color string `json:"color,omitempty"`
	// this are leftovers from original fcm.NotificationPayload
	ClickAction      string `json:"click_action,omitempty"`
	BodyLocKey       string `json:"body_loc_key,omitempty"`
	BodyLocArgs      string `json:"body_loc_args,omitempty"`
	TitleLocKey      string `json:"title_loc_key,omitempty"`
	TitleLocArgs     string `json:"title_loc_args,omitempty"`
	AndroidChannelID string `json:"android_channel_id,omitempty"`
}

// ToFCMNotificationPayload turns Payload into fcm.NotificationPayload
func (p *Payload) ToFCMNotificationPayload() *fcm.NotificationPayload {
	return &fcm.NotificationPayload{
		Title:            p.Title,
		Body:             p.Body,
		Icon:             p.Icon,
		Sound:            p.Sound,
		Badge:            p.Badge,
		Tag:              p.Tag,
		Color:            p.Color,
		ClickAction:      p.ClickAction,
		BodyLocKey:       p.BodyLocKey,
		BodyLocArgs:      p.BodyLocArgs,
		TitleLocKey:      p.BodyLocKey,
		TitleLocArgs:     p.TitleLocArgs,
		AndroidChannelID: p.AndroidChannelID,
	}
}
