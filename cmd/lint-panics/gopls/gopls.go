package gopls

import (
	"fmt"
	"os/exec"
	"github.com/pkg/errors"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"net"
	"context"

	"time"
	"io"

	"go.uber.org/zap"
)

const (
	goplsRemote  = true
	goplsAddress = "http://localhost:6060"
)

var requestID = 1

type Connection struct {
	logger  *zap.Logger
	tcpConn net.Conn
	server  protocol.Server
	cmd     *exec.Cmd
	client  protocol.Client
	stream  jsonrpc2.Stream
	conn    jsonrpc2.Conn
	stdin   io.WriteCloser
	stdout  io.ReadCloser
}

// CombinedReadWriteCloser combines stdin and stdout into one interface.
type CombinedReadWriteCloser struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// Write writes data to stdin.
func (c *CombinedReadWriteCloser) Write(p []byte) (n int, err error) {
	return c.stdin.Write(p)
}

// Read reads data from stdout.
func (c *CombinedReadWriteCloser) Read(p []byte) (n int, err error) {
	return c.stdout.Read(p)
}

// Close closes both stdin and stdout.
func (c *CombinedReadWriteCloser) Close() error {
	err1 := c.stdin.Close()
	err2 := c.stdout.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func NewGoplsClient(ctx context.Context, logger *zap.Logger) *Connection {
	var err error

	logger.Debug("initializing gopls client")

	gopls := &Connection{
		logger: logger,
		client: NewDummyClient(logger),
	}

	//// Create a JSON-RPC connection using stdin and stdout
	//gopls.cmd = exec.Command("gopls", "serve", "-rpc.trace", "-debug", "localhost:6061")
	////gopls.Stdout = os.Stdout
	////stdout := os.Stdout
	//gopls.cmd.Stderr = os.Stderr
	//
	//gopls.stdin, err = gopls.cmd.StdinPipe()
	//if err != nil {
	//	logger.Error("Failed to get stdin pipe", "error", err)
	//	panic(err)
	//}
	//
	//gopls.stdout, err = gopls.cmd.StdoutPipe()
	//if err != nil {
	//	logger.Error("Failed to get stdout pipe", "error", err)
	//	panic(err)
	//}
	//
	//err = gopls.cmd.Start()
	//if err != nil {
	//	logger.Error("Failed to start gopls", "error", err)
	//	panic(err)
	//}
	//
	//std := &CombinedReadWriteCloser{
	//	stdin:  gopls.stdin,
	//	stdout: gopls.stdout,
	//}
	//gopls.stream = jsonrpc2.NewStream(std)

	// Dial to the gopls server (running on 127.0.0.1:6060)
	conn, err := net.Dial("tcp", "127.0.0.1:6060")
	if err != nil {
		panic(err)
	}
	gopls.stream = jsonrpc2.NewStream(conn)

	ctx, gopls.conn, gopls.server = protocol.NewClient(ctx, gopls.client, gopls.stream, logger)

	initParams := protocol.InitializeParams{
		//ProcessID: 1,
		//ClientInfo: &protocol.ClientInfo{
		//	Name:    "lint-panics",
		//	Version: "0.0.1",
		//},
		RootURI: "file://Users/igorsirotin/Repositories/Status/status-go/",

		InitializationOptions: map[string]interface{}{
			"symbolMatcher": "FastFuzzy",
		},
	}

	fmt.Println("Sending initialize request...")
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

	//
	//// Send the definition request
	//params := &protocol.DefinitionParams{
	//	TextDocumentPositionParams: protocol.TextDocumentPositionParams{
	//		TextDocument: protocol.TextDocumentIdentifier{
	//			URI: "file://Users/igorsirotin/Repositories/Status/status-go/telemetry/client.go",
	//		},
	//		Position: protocol.Position{
	//			Line:      uint32(215),
	//			Character: uint32(9),
	//		},
	//	},
	//}
	//
	//locations, err := gopls.server.Definition(ctx, params)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Printf("Definition result: %+v\n", locations)
	return gopls
}

func (gopls *Connection) Definition(filePath string, lineNumber int, charPosition int) (string, int, error) {
	// NOTE: gopls uses 0-based line and column numbers
	defFile, defLine, err := gopls.definitionTCP(filePath, lineNumber-1, charPosition-1)
	return defFile, defLine + 1, err
}

func (gopls *Connection) definitionTCP(filePath string, lineNumber int, charPosition int) (string, int, error) {
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

	//call, err := jsonrpc2.NewCall(
	//	jsonrpc2.NewNumberID(1),
	//	"textDocument/definition",
	//	params,
	//)

	//if err != nil {
	//	return "", 0, errors.Wrap(err, "failed to create call")
	//}

	// Create context with a timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locations, err := gopls.server.Definition(ctx, params)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to fetch definition")
	}

	//requestJSON, err := call.MarshalJSON()
	//if err != nil {
	//	return "", 0, errors.Wrap(err, "failed to marshal request")
	//}
	//
	//// Send the request to the gopls server running on HTTP
	//resp, err := http.Post(goplsAddress, "application/json", bytes.NewBuffer(requestJSON))
	//if err != nil {
	//	return "", 0, errors.Wrap(err, "failed to send request to gopls server")
	//}
	//defer resp.Body.Close()
	//
	//body, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	return "", 0, errors.Wrap(err, "failed to read response body")
	//}

	//// Decode the response
	//var jsonResp jsonrpc2.Response
	//err = jsonResp.UnmarshalJSON(body)
	//if err != nil {
	//	return "", 0, errors.Wrap(err, "failed to decode gopls response")
	//}
	//
	//// Check if there was an error in the response
	//if jsonResp.Err() != nil {
	//	log.Error("Error from gopls: %v", jsonResp.Err().Error())
	//	return "", 0, errors.New(jsonResp.Err().Error())
	//}
	//
	//// Print the result (the location of the definition)
	//var locations []protocol.Location
	//if err := json.Unmarshal(jsonResp.Result(), &locations); err != nil {
	//	log.Error("Failed to unmarshal result: %v", err)
	//	return "", 0, errors.Wrap(err, "failed to unmarshal result")
	//}

	if len(locations) == 0 {
		return "", 0, errors.New("no definition found")
	}

	//for index, loc := range locations {
	//
	//	q := protocol.HoverParams{
	//		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
	//			TextDocument: protocol.TextDocumentIdentifier{URI: loc.URI},
	//			Position:     loc.Range.Start,
	//		},
	//	}
	//	hover, err := gopls.server.Hover(ctx, &q)
	//	if err != nil {
	//		return "", 0, err
	//	}
	//	var description string
	//	if hover != nil {
	//		description = strings.TrimSpace(hover.Contents.Value)
	//	}
	//
	//	gopls.logger.Info("definition found",
	//		zap.Int("index", index),
	//		zap.String("description", description),
	//		zap.Any("location", loc),
	//	)
	//}

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
