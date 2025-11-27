package linkpreview

import (
	"github.com/status-im/status-go/protocol/common"
)

const UnfurledLinksPerMessageLimit = 5

type URLUnfurlPermission int

const (
	URLUnfurlingAllowed URLUnfurlPermission = iota
	URLUnfurlingAskUser
	URLUnfurlingForbiddenBySettings
	URLUnfurlingNotSupported
)

type URLUnfurlingMetadata struct {
	URL               string              `json:"url"`
	Permission        URLUnfurlPermission `json:"permission"`
	IsStatusSharedURL bool                `json:"isStatusSharedURL"`
}

type URLsUnfurlPlan struct {
	URLs []URLUnfurlingMetadata `json:"urls"`
}

type UnfurlURLsResponse struct {
	LinkPreviews       []*common.LinkPreview       `json:"linkPreviews,omitempty"`
	StatusLinkPreviews []*common.StatusLinkPreview `json:"statusLinkPreviews,omitempty"`
}
