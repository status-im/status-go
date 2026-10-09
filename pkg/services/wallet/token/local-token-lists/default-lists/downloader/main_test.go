package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	defaulttokenlists "github.com/status-im/status-go/pkg/services/wallet/token/local-token-lists/default-lists"
)

func TestDownloadTokensSuccess(t *testing.T) {
	body := []byte(`{"name":"Status"}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	output := filepath.Join(t.TempDir(), "status.go")
	err := downloadTokens(server.Client(), "status", defaulttokenlists.TokensSource{
		SourceURL:  server.URL,
		OutputFile: output,
	})
	require.NoError(t, err)

	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(written), `StatusTokenList.ID = "status"`)
	require.Contains(t, string(written), `StatusTokenList.SourceURL = "`+server.URL+`"`)
	require.Contains(t, string(written), "//go:embed status.json\n")
	require.Contains(t, string(written), "StatusTokenList.JsonData = statusTokenListJSON")
	embedded, err := os.ReadFile(filepath.Join(filepath.Dir(output), "status.json"))
	require.NoError(t, err)
	require.Equal(t, body, embedded)

	entries, err := os.ReadDir(filepath.Dir(output))
	require.NoError(t, err)
	require.Len(t, entries, 2)

	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestDownloadTokensPreservesExistingFileMode(t *testing.T) {
	output := filepath.Join(t.TempDir(), "status.go")
	require.NoError(t, os.WriteFile(output, []byte("previous list"), 0o600))

	err := downloadTokens(okClient(), "status", defaulttokenlists.TokensSource{
		SourceURL:  "http://example.test/tokens",
		OutputFile: output,
	})
	require.NoError(t, err)

	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(written), `StatusTokenList.ID = "status"`)
}

func TestDownloadTokensValidatesAgainstSchema(t *testing.T) {
	const schema = `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`
	body := []byte(`{"name":"Status"}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/schema.json") {
			_, _ = w.Write([]byte(schema))
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	output := filepath.Join(t.TempDir(), "uniswap.go")
	err := downloadTokens(server.Client(), "uniswap", defaulttokenlists.TokensSource{
		SourceURL:  server.URL + "/list.json",
		Schema:     server.URL + "/schema.json",
		OutputFile: output,
	})
	require.NoError(t, err)
	_, err = os.Stat(output)
	require.NoError(t, err)
}

func TestDownloadTokensErrors(t *testing.T) {
	okBody := []byte(`{"name":"Status"}`)

	schemaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"type":"object","required":["name"]}`))
	}))
	t.Cleanup(schemaServer.Close)

	tests := []struct {
		name    string
		client  *http.Client
		source  defaulttokenlists.TokensSource
		setup   func(t *testing.T)
		wantErr string
	}{
		{
			name: "fetch",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			})},
			source:  defaulttokenlists.TokensSource{SourceURL: "http://example.test/tokens"},
			wantErr: "failed to fetch tokens",
		},
		{
			name: "read",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(errReader{}),
					Header:     make(http.Header),
				}, nil
			})},
			source:  defaulttokenlists.TokensSource{SourceURL: "http://example.test/tokens"},
			wantErr: "failed to read tokens",
		},
		{
			name: "status",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Status:     "502 Bad Gateway",
					Body:       io.NopCloser(bytes.NewReader(nil)),
					Header:     make(http.Header),
				}, nil
			})},
			source:  defaulttokenlists.TokensSource{SourceURL: "http://example.test/tokens"},
			wantErr: "unexpected status 502 Bad Gateway",
		},
		{
			name: "json",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, "200 OK", []byte("not-json")), nil
			})},
			source:  defaulttokenlists.TokensSource{SourceURL: "http://example.test/tokens"},
			wantErr: "failed to unmarshal body",
		},
		{
			name: "schema",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, "200 OK", []byte(`{"symbol":"WETH"}`)), nil
			})},
			source: defaulttokenlists.TokensSource{
				SourceURL: "http://example.test/tokens",
				Schema:    schemaServer.URL,
			},
			wantErr: "failed to validate token list against schema",
		},
		{
			name: "create",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, "200 OK", okBody), nil
			})},
			source: defaulttokenlists.TokensSource{
				SourceURL:  "http://example.test/tokens",
				OutputFile: filepath.Join(t.TempDir(), "missing", "out.go"),
			},
			wantErr: "failed to create json file",
		},
		{
			name: "write",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, "200 OK", okBody), nil
			})},
			source: defaulttokenlists.TokensSource{
				SourceURL:  "http://example.test/tokens",
				OutputFile: filepath.Join(t.TempDir(), "out.go"),
			},
			setup: func(t *testing.T) {
				t.Helper()
				stubTempOutput(t, func(dir, _ string) (string, io.WriteCloser, error) {
					return filepath.Join(dir, "out.go.tmp"), failWriteCloser{}, nil
				})
			},
			wantErr: "failed to write file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			err := downloadTokens(tt.client, "status", tt.source)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestDownloadTokensWriteFailureKeepsExistingFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.go")
	const previous = "package defaulttokenlists\n// previous list\n"
	require.NoError(t, os.WriteFile(output, []byte(previous), 0o600))

	leftover := filepath.Join(dir, ".out.go.leftover")
	require.NoError(t, os.WriteFile(leftover, []byte("temp"), 0o600))
	stubTempOutput(t, func(string, string) (string, io.WriteCloser, error) {
		return leftover, failWriteCloser{}, nil
	})

	err := downloadTokens(okClient(), "status", defaulttokenlists.TokensSource{
		SourceURL:  "http://example.test/tokens",
		OutputFile: output,
	})
	require.ErrorContains(t, err, "failed to write file")

	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, previous, string(written))
	_, err = os.Stat(leftover)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDownloadTokensCloseFailureDoesNotReplaceFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.go")
	const previous = "package defaulttokenlists\n// previous list\n"
	require.NoError(t, os.WriteFile(output, []byte(previous), 0o600))

	leftover := filepath.Join(dir, ".out.go.leftover")
	require.NoError(t, os.WriteFile(leftover, []byte("temp"), 0o600))
	stubTempOutput(t, func(string, string) (string, io.WriteCloser, error) {
		return leftover, failCloseWriter{}, nil
	})

	err := downloadTokens(okClient(), "status", defaulttokenlists.TokensSource{
		SourceURL:  "http://example.test/tokens",
		OutputFile: output,
	})
	require.ErrorContains(t, err, "failed to write file")

	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, previous, string(written))
	_, err = os.Stat(leftover)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDownloadTokensRenameFailureKeepsExistingFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.go")
	require.NoError(t, os.Mkdir(output, 0o750))

	err := downloadTokens(okClient(), "status", defaulttokenlists.TokensSource{
		SourceURL:  "http://example.test/tokens",
		OutputFile: output,
	})
	require.ErrorContains(t, err, "failed to replace go file")

	info, err := os.Stat(output)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestMainExitsWhenDownloadFailsAndStillWritesTheRest(t *testing.T) {
	body := []byte(`{"name":"Status"}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "broken") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	output := filepath.Join(t.TempDir(), "status.go")
	restore := stubMainDeps(t, map[string]defaulttokenlists.TokensSource{
		"broken": {SourceURL: server.URL + "/broken"},
		"status": {SourceURL: server.URL + "/ok", OutputFile: output},
	})
	defer restore()

	var exitCode int
	var exited bool
	osExit = func(code int) {
		exited = true
		exitCode = code
	}

	stderr := captureStderr(t, func() { main() })

	require.True(t, exited)
	require.Equal(t, 1, exitCode)
	require.Contains(t, stderr, "ERR: [broken]")
	written, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(written), `StatusTokenList.ID = "status"`)
}

func TestMainDoesNotExitWhenEveryDownloadSucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name":"Status"}`))
	}))
	t.Cleanup(server.Close)

	restore := stubMainDeps(t, map[string]defaulttokenlists.TokensSource{
		"status": {
			SourceURL:  server.URL,
			OutputFile: filepath.Join(t.TempDir(), "status.go"),
		},
	})
	defer restore()

	osExit = func(int) { t.Fatal("os.Exit called") }
	main()
}

// Regenerating every list from its embedded data must reproduce the committed .go and .json files.
func TestWriteTokenListReproducesCommittedFiles(t *testing.T) {
	lists := []defaulttokenlists.DownloadedTokenList{
		defaulttokenlists.StatusTokenList, defaulttokenlists.UniswapTokenList,
		defaulttokenlists.CoingeckoEthereumTokenList, defaulttokenlists.CoingeckoOptimismTokenList,
		defaulttokenlists.CoingeckoArbitrumTokenList, defaulttokenlists.CoingeckoBaseTokenList,
		defaulttokenlists.CoingeckoBscTokenList, defaulttokenlists.CoingeckoLineaTokenList,
	}
	require.Len(t, lists, len(defaulttokenlists.TokensSources))

	dir := t.TempDir()
	for _, l := range lists {
		source, ok := defaulttokenlists.TokensSources[l.ID]
		require.True(t, ok, l.ID)
		base := filepath.Base(source.OutputFile)
		require.NoError(t, writeTokenList(l.ID, l.SourceURL, filepath.Join(dir, base), l.JsonData, l.Fetched))

		for _, name := range []string{base, base[:len(base)-len(".go")] + ".json"} {
			want, err := os.ReadFile(filepath.Join("..", name))
			require.NoError(t, err)
			got, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			require.Equal(t, string(want), string(got), name)
		}
	}

	require.NoError(t, writeManifest(dir))
	want, err := os.ReadFile(filepath.Join("..", "SHA256SUMS"))
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
}

func TestWriteTokenListLeavesNoOrphanJSON(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "list.go")
	require.NoError(t, os.Mkdir(outputFile, 0o700)) // the .go cannot be written over a directory

	require.Error(t, writeTokenList("list", "https://example.com/list.json", outputFile, []byte(`{}`), time.Unix(0, 0)))
	_, err := os.Stat(filepath.Join(dir, "list.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestWriteManifests(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"a":1}`), 0o600))
	require.NoError(t, writeManifests(map[string]defaulttokenlists.TokensSource{"a": {OutputFile: filepath.Join(dir, "a.go")}}))
	require.ErrorContains(t, writeManifests(map[string]defaulttokenlists.TokensSource{
		"missing": {OutputFile: filepath.Join(dir, "missing", "b.go")},
	}), "failed to write")
	got, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	require.NoError(t, err)
	require.Equal(t, "015abd7f5cc57a2dd94b7590f04ad8084273905ee33ec5cebeae62276a97f862  a.json\n", string(got))
}

func TestWriteTokenListFailsWithoutWritingWhenJSONCannotBeWritten(t *testing.T) {
	dir := t.TempDir()
	outputFile := filepath.Join(dir, "missing", "list.go")
	require.Error(t, writeTokenList("list", "https://example.com/list.json", outputFile, []byte(`{}`), time.Unix(0, 0)))
	_, err := os.Stat(outputFile)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func stubMainDeps(t *testing.T, sources map[string]defaulttokenlists.TokensSource) func() {
	t.Helper()
	originalSources := tokenSources
	originalExit := osExit
	tokenSources = sources
	return func() {
		tokenSources = originalSources
		osExit = originalExit
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stderr
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = writer
	defer func() { os.Stderr = original }()

	fn()
	require.NoError(t, writer.Close())
	captured, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(captured)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type failWriteCloser struct{}

func (failWriteCloser) Write([]byte) (int, error) { return 0, errors.New("disk full") }
func (failWriteCloser) Close() error              { return nil }

type failCloseWriter struct{}

func (failCloseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (failCloseWriter) Close() error                { return errors.New("close failed") }

func okClient() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, "200 OK", []byte(`{"name":"Status"}`)), nil
	})}
}

func stubTempOutput(t *testing.T, fn func(dir, pattern string) (string, io.WriteCloser, error)) {
	t.Helper()
	original := createTempOutput
	t.Cleanup(func() { createTempOutput = original })
	createTempOutput = fn
}

func jsonResponse(code int, status string, body []byte) *http.Response {
	return &http.Response{
		StatusCode: code,
		Status:     status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}
