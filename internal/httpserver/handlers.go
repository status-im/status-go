package httpserver

import "net/http"

// HandlerPatternMap maps a URL pattern to the handler that serves it.
type HandlerPatternMap map[string]http.HandlerFunc
