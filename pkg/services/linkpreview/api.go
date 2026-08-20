package linkpreview

type PublicAPI struct {
	service *Service
}

// GetTextURLsToUnfurl parses text and returns a deduplicated and (somewhat) normalized
// slice of URLs. The returned URLs can be used as cache keys by clients.
// For each URL there's a corresponding metadata which should be used as to plan the unfurling.
func (api *PublicAPI) GetTextURLsToUnfurl(text string) *URLsUnfurlPlan {
	return api.service.GetTextURLsToUnfurl(text)
}

// UnfurlURLs uses a best-effort approach to unfurl each URL. Failed URLs will
// be removed from the response.
//
// This endpoint expects the client to send URLs normalized by GetTextURLs.
func (api *PublicAPI) UnfurlURLs(urls []string) (UnfurlURLsResponse, error) {
	return api.service.UnfurlURLs(urls)
}
