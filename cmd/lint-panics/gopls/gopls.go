package gopls

import (
	"os/exec"

	"github.com/pkg/errors"

	"context"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"time"

	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

type Connection struct {
	logger *zap.Logger
	server protocol.Server
	cmd    *exec.Cmd
	conn   jsonrpc2.Conn
}

func NewGoplsClient(ctx context.Context, logger *zap.Logger, rootDir string) *Connection {
	var err error

	logger.Debug("initializing gopls client")

	gopls := &Connection{
		logger: logger,
	}

	client := NewDummyClient(logger)

	// Create a JSON-RPC connection using stdin and stdout
	gopls.cmd = exec.Command("gopls", "serve")

	stdin, err := gopls.cmd.StdinPipe()
	if err != nil {
		logger.Error("Failed to get stdin pipe", zap.Error(err))
		panic(err)
	}

	stdout, err := gopls.cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to get stdout pipe", zap.Error(err))
		panic(err)
	}

	err = gopls.cmd.Start()
	if err != nil {
		logger.Error("Failed to start gopls", zap.Error(err))
		panic(err)
	}

	stream := jsonrpc2.NewStream(&CombinedReadWriteCloser{
		stdin:  stdin,
		stdout: stdout,
	})

	ctx, gopls.conn, gopls.server = protocol.NewClient(ctx, client, stream, logger)

	initParams := protocol.InitializeParams{
		RootURI: uri.From("file", "", rootDir, "", ""),
		InitializationOptions: map[string]interface{}{
			"symbolMatcher": "FastFuzzy",
		},
	}

	_, err = gopls.server.Initialize(ctx, &initParams)
	if err != nil {
		logger.Error("Error during initialize", zap.Error(err))
		panic(err)
	}

	// Step 2: Send 'initialized' notification
	err = gopls.server.Initialized(ctx, &protocol.InitializedParams{})
	if err != nil {
		logger.Error("Error during initialized", zap.Error(err))
		panic(err)
	}

	return gopls
}

func (gopls *Connection) Definition(ctx context.Context, filePath string, lineNumber int, charPosition int) (string, int, error) {
	// NOTE: gopls uses 0-based line and column numbers
	defFile, defLine, err := gopls.definition(ctx, filePath, lineNumber-1, charPosition-1)
	return defFile, defLine + 1, err
}

func (gopls *Connection) definition(ctx context.Context, filePath string, lineNumber int, charPosition int) (string, int, error) {
	// Define the file URI and position where the function/method is invoked
	fileURI := protocol.DocumentURI("file://" + filePath) // Replace with actual file URI
	line := lineNumber                                    // Line number where the function is called
	character := charPosition                             // Character (column) where the function is called

	// Send the definition request
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: fileURI,
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	// Create context with a timeout to avoid hanging
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	locations, err := gopls.server.Definition(ctx, params)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to fetch definition")
	}

	if len(locations) == 0 {
		return "", 0, errors.New("no definition found")
	}

	location := locations[0]
	return location.URI.Filename(), int(location.Range.Start.Line), nil
}

func (gopls *Connection) DidOpen(ctx context.Context, path string, content string, logger *zap.Logger) {
	err := gopls.server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        protocol.DocumentURI(path),
			LanguageID: "go",
			Version:    1,
			Text:       content,
		},
	})
	if err != nil {
		logger.Error("failed to call DidOpen", zap.Error(err))
	}
}

func (gopls *Connection) DidClose(ctx context.Context, path string, lgoger *zap.Logger) {
	err := gopls.server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(path),
		},
	})
	if err != nil {
		lgoger.Error("failed to call DidClose", zap.Error(err))
	}
}
