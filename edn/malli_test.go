package edn

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	common "github.com/status-im/status-go/protocol/common"
)

func TestConvertStructToMalli(t *testing.T) {
	malliRepresentation := ConvertStructToMalli(reflect.TypeOf(common.Message{}))
	bytes, err := json.MarshalIndent(malliRepresentation, "", "  ")
	require.NoError(t, err)
	fmt.Println(string(bytes))
}
