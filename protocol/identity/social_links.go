package identity

import (
	"encoding/json"
	"sort"

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

func (s *SocialLinks) TransformToProtobuf() []*protobuf.SocialLink {
	res := []*protobuf.SocialLink{}
	for _, link := range *s {
		res = append(res, &protobuf.SocialLink{Text: link.Text, Url: link.URL})
	}
	return res
}

func (s SocialLinks) Equals(rhs SocialLinks) bool {
	if len(s) != len(rhs) {
		return false
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Text < s[j].Text })
	sort.Slice(rhs, func(i, j int) bool { return rhs[i].Text < rhs[j].Text })
	for i := range s {
		if s[i] != rhs[i] {
			return false
		}
	}

	return true
}

func (s SocialLinks) EqualsProtobuf(rhs []*protobuf.SocialLink) bool {
	if len(s) != len(rhs) {
		return false
	}
	sort.Slice(s, func(i, j int) bool { return s[i].Text < s[j].Text })
	sort.Slice(rhs, func(i, j int) bool { return rhs[i].Text < rhs[j].Text })
	for i := range s {
		if s[i].Text != rhs[i].Text || s[i].URL != rhs[i].Url {
			return false
		}
	}

	return true
}

func (s *SocialLinks) Serialize() ([]byte, error) {
	return json.Marshal(s)
}
