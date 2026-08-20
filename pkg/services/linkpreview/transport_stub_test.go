package linkpreview

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

// StubMatcher should either return an http.Response or nil in case the request
// doesn't match.
type StubMatcher func(req *http.Request) *http.Response

type StubTransport struct {
	// fallbackToDefaultTransport when true will make the transport use
	// http.DefaultTransport in case no matcher is found.
	fallbackToDefaultTransport bool
	// disabledStubs when true, will skip all matchers and use
	// http.DefaultTransport.
	//
	// Useful while testing to toggle between the original and stubbed responses.
	disabledStubs bool
	// matchers are http.RoundTripper functions.
	matchers []StubMatcher
}

// RoundTrip returns a stubbed response if any matcher returns a non-nil
// http.Response. If no matcher is found and fallbackToDefaultTransport is true,
// then it executes the HTTP request using the default http transport.
//
// If StubTransport#disabledStubs is true, the default http transport is used.
func (t *StubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.disabledStubs {
		return http.DefaultTransport.RoundTrip(req)
	}

	for _, matcher := range t.matchers {
		res := matcher(req)
		if res != nil {
			return res, nil
		}
	}

	if t.fallbackToDefaultTransport {
		return http.DefaultTransport.RoundTrip(req)
	}

	return nil, fmt.Errorf("no HTTP matcher found")
}

func (t *StubTransport) AddURLMatcherRoundTrip(urlRegexp string, roundTrip func(r *http.Request) *http.Response) {
	matcher := func(req *http.Request) *http.Response {
		rx, err := regexp.Compile(regexp.QuoteMeta(urlRegexp))
		if err != nil {
			return nil
		}
		if !rx.MatchString(req.URL.String()) {
			return nil
		}
		return roundTrip(req)
	}
	t.matchers = append(t.matchers, matcher)
}

// Add a matcher based on a URL regexp. If a given request URL matches the
// regexp, then responseBody will be returned with a hardcoded 200 status code.
// If headers is non-nil, use it as the value of http.Response.Header.
func (t *StubTransport) AddURLMatcher(urlRegexp string, responseBody []byte, headers map[string]string) {
	matcher := func(req *http.Request) *http.Response {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		}

		if headers != nil {
			res.Header = http.Header{}
			for k, v := range headers {
				res.Header.Set(k, v)
			}
		}

		return res
	}
	t.AddURLMatcherRoundTrip(urlRegexp, matcher)
}
