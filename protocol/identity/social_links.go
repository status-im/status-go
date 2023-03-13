package identity

import (
	"encoding/json"
	"reflect"

	"github.com/status-im/status-go/protocol/protobuf"
)

// static links which need to be decorated by the UI clients
const (
	TwitterID      = "__twitter"
	PersonalSiteID = "__personal_site"
	GithubID       = "__github"
	YoutubeID      = "__youtube"
	DiscordID      = "__discord"
	TelegramID     = "__telegram"
)

type SocialLink struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type SocialLinks []SocialLink

func NewSocialLinks(links []*protobuf.SocialLink) *SocialLinks {
	res := SocialLinks{}
	for _, link := range links {
		res = append(res, SocialLink{Text: link.Text, URL: link.Url})
	}
	return &res
}

func (s *SocialLinks) ToProtobuf() []*protobuf.SocialLink {
	res := []*protobuf.SocialLink{}
	for _, link := range *s {
		res = append(res, &protobuf.SocialLink{Text: link.Text, Url: link.URL})
	}
	return res
}

func (s SocialLinks) Equals(rhs SocialLinks) bool {
	return reflect.DeepEqual(s, rhs)
}

func (s *SocialLinks) Serialize() ([]byte, error) {
	return json.Marshal(s)
}
