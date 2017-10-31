package account

import (
	"testing"
	"github.com/status-im/status-go/geth/common"
	"github.com/golang/mock/gomock"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

func TestManager_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := common.NewMockNodeManager(ctrl)
	nodeMock.EXPECT().WhisperService().Return(&whisper.Whisper{},nil)
	m:=NewManager(nodeMock)
	if err:=m.Logout(); err!=nil {
		t.FailNow()
	}
}