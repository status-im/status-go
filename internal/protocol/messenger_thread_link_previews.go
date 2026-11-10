package protocol

import "github.com/status-im/status-go/internal/protocol/common"

// SetThreadLinkPreviewResolver configures local-only metadata resolution before
// the messenger starts. No resolver means thread previews remain redacted.
func (m *Messenger) SetThreadLinkPreviewResolver(resolver func(string) (*common.LinkPreview, error)) {
	m.threadLinkPreviewResolver = resolver
}

func (m *Messenger) resolveThreadLinkPreviews(message *common.Message) {
	appendReference := func(URL string) {
		if !common.IsStatusThreadURL(URL) {
			return
		}
		for _, standard := range message.LinkPreviews {
			if standard.URL == URL {
				return
			}
		}
		message.LinkPreviews = append(message.LinkPreviews, *common.NewUnknownThreadPreview(URL))
	}
	for _, preview := range message.GetUnfurledLinks() {
		appendReference(preview.GetUrl())
	}
	for _, preview := range message.GetUnfurledStatusLinks().GetUnfurledStatusLinks() {
		appendReference(preview.GetUrl())
	}
	statusPreviews := message.StatusLinkPreviews[:0]
	for _, preview := range message.StatusLinkPreviews {
		if common.IsStatusThreadURL(preview.URL) {
			appendReference(preview.URL)
		} else {
			statusPreviews = append(statusPreviews, preview)
		}
	}
	message.StatusLinkPreviews = statusPreviews
	for index, preview := range message.LinkPreviews {
		if !common.IsStatusThreadURL(preview.URL) {
			continue
		}
		resolved := common.NewUnknownThreadPreview(preview.URL)
		if m.threadLinkPreviewResolver != nil {
			local, err := m.threadLinkPreviewResolver(preview.URL)
			if err == nil && local != nil && local.URL == preview.URL && local.ThreadAccessible {
				resolved = local
			}
		}
		message.LinkPreviews[index] = *resolved
	}
}
