package gopls

import (
	"context"
	"go.lsp.dev/protocol"

	"go.uber.org/zap"
)

type DummyClient struct {
	logger *zap.Logger
}

func NewDummyClient(logger *zap.Logger) *DummyClient {
	return &DummyClient{
		logger: nil, //logger.Named("client"),
	}
}

func (d *DummyClient) Progress(ctx context.Context, params *protocol.ProgressParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: Progress", zap.Any("params", params))
	}
	return
}
func (d *DummyClient) WorkDoneProgressCreate(ctx context.Context, params *protocol.WorkDoneProgressCreateParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: WorkDoneProgressCreate")
	}
	return nil
}

func (d *DummyClient) LogMessage(ctx context.Context, params *protocol.LogMessageParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: LogMessage", zap.Any("message", params))
	}
	return nil
}

func (d *DummyClient) PublishDiagnostics(ctx context.Context, params *protocol.PublishDiagnosticsParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: PublishDiagnostics")
	}
	return nil
}

func (d *DummyClient) ShowMessage(ctx context.Context, params *protocol.ShowMessageParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: ShowMessage", zap.Any("message", params))
	}
	return nil
}

func (d *DummyClient) ShowMessageRequest(ctx context.Context, params *protocol.ShowMessageRequestParams) (result *protocol.MessageActionItem, err error) {
	if d.logger != nil {
		d.logger.Debug("client: ShowMessageRequest", zap.Any("message", params))
	}
	return nil, nil
}

func (d *DummyClient) Telemetry(ctx context.Context, params interface{}) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: Telemetry")
	}
	return nil
}

func (d *DummyClient) RegisterCapability(ctx context.Context, params *protocol.RegistrationParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: RegisterCapability")
	}
	return nil
}

func (d *DummyClient) UnregisterCapability(ctx context.Context, params *protocol.UnregistrationParams) (err error) {
	if d.logger != nil {
		d.logger.Debug("client: UnregisterCapability")
	}
	return nil
}

func (d *DummyClient) ApplyEdit(ctx context.Context, params *protocol.ApplyWorkspaceEditParams) (result bool, err error) {
	if d.logger != nil {
		d.logger.Debug("client: ApplyEdit")
	}
	return false, nil
}

func (d *DummyClient) Configuration(ctx context.Context, params *protocol.ConfigurationParams) (result []interface{}, err error) {
	if d.logger != nil {
		d.logger.Debug("client: Configuration")
	}
	return nil, nil
}

func (d *DummyClient) WorkspaceFolders(ctx context.Context) (result []protocol.WorkspaceFolder, err error) {
	if d.logger != nil {
		d.logger.Debug("client: WorkspaceFolders")
	}
	return nil, nil
}
