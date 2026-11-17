package params

// TODO(status-im/status-app#23177): remove with the test proxy fallback.

const (
	// ProdProxyStageName is the stage of the Status proxies release builds use.
	ProdProxyStageName = "prod"
	// TestProxyStageName is the stage of the Status proxies that back the prod ones up.
	TestProxyStageName = "test"
)

// FallbackProxyStages returns the stages whose Status proxies are asked, in
// this order and after everything else, when the proxy of stageName fails.
// Only prod has any, and a proxy at a custom URL has none.
//
// A fallback proxy goes last in a provider list, is used wherever the main
// proxy is, and is reached with puzzle auth, never with credentials.
func FallbackProxyStages(stageName, customURL string) []string {
	if customURL != "" || stageName != ProdProxyStageName {
		return nil
	}
	return []string{TestProxyStageName}
}
