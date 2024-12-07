package callog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/require"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/status-im/status-go/logutils/requestlog"
)

func TestRemoveSensitiveInfo(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic test",
			input:    `{"username":"user1","password":"secret123","mnemonic":"mnemonic123 xyz"}`,
			expected: fmt.Sprintf(`{"username":"user1","password":"%s","mnemonic":"%s"}`, redactionPlaceholder, redactionPlaceholder),
		},
		{
			name:     "uppercase password field",
			input:    `{"USERNAME":"user1","PASSWORD":"secret123"}`,
			expected: fmt.Sprintf(`{"USERNAME":"user1","PASSWORD":"%s"}`, redactionPlaceholder),
		},
		{
			name:     "password field with spaces",
			input:    `{"username":"user1", "password" : "secret123"}`,
			expected: fmt.Sprintf(`{"username":"user1", "password":"%s"}`, redactionPlaceholder),
		},
		{
			name:     "multiple password fields",
			input:    `{"password":"secret123","data":{"nested_password":"nested_secret"}}`,
			expected: fmt.Sprintf(`{"password":"%s","data":{"nested_password":"%s"}}`, redactionPlaceholder, redactionPlaceholder),
		},
		{
			name:     "no password field",
			input:    `{"username":"user1","email":"user1@example.com"}`,
			expected: `{"username":"user1","email":"user1@example.com"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeSensitiveInfo(tc.input)
			if result != tc.expected {
				t.Errorf("Expected: %s, Got: %s", tc.expected, result)
			}
		})
	}
}

func TestCall(t *testing.T) {
	// Create default logger
	buffer := bytes.NewBuffer(nil)
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buffer),
		zap.DebugLevel,
	))

	// Create a temporary file for request logging
	tempLogFile, err := os.CreateTemp(t.TempDir(), "TestCall*.log")
	require.NoError(t, err)

	// Enable request logging
	requestLogger, err := requestlog.CreateRequestLogger(tempLogFile.Name())
	require.NoError(t, err)
	require.NotNil(t, requestLogger)

	// Test case 1: Normal execution
	testFunc := func(param string) string {
		return "test result: " + param
	}
	testParam := "test input"
	expectedResult := "test result: test input"

	result := CallWithResponse(logger, requestLogger, testFunc, testParam)

	// Check the result
	if result != expectedResult {
		t.Errorf("Expected result %s, got %s", expectedResult, result)
	}

	// Read the log file
	logData, err := os.ReadFile(tempLogFile.Name())
	require.NoError(t, err)
	requestLogOutput := string(logData)

	// Check if the log contains expected information
	expectedLogParts := []string{getShortFunctionName(testFunc), "request", testParam, "response", expectedResult}
	for _, part := range expectedLogParts {
		if !strings.Contains(requestLogOutput, part) {
			t.Errorf("Log output doesn't contain expected part: %s", part)
		}
	}

	// Test case 2: Panic -> recovery -> re-panic
	require.Empty(t, buffer.String())

	e := "test panic"
	panicFunc := func() {
		panic(e)
	}

	require.PanicsWithValue(t, e, func() {
		Call(logger, requestLogger, panicFunc)
	})

	// Check if the panic was logged
	if !strings.Contains(buffer.String(), "panic found in call") {
		t.Errorf("Log output doesn't contain panic information")
	}
	if !strings.Contains(buffer.String(), e) {
		t.Errorf("Log output doesn't contain panic message")
	}
	if !strings.Contains(buffer.String(), "stacktrace") {
		t.Errorf("Log output doesn't contain stacktrace")
	}
}

func initializeApplication(requestJSON string) string {
	return ""
}

func TestGetFunctionName(t *testing.T) {
	fn := getShortFunctionName(initializeApplication)
	require.Equal(t, "initializeApplication", fn)
}

func TestDataField(t *testing.T) {
	entry := zapcore.Entry{}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})

	f := dataField("root", "value")
	require.NotNil(t, f)
	require.Equal(t, "root", f.Key)
	require.Equal(t, zapcore.StringType, f.Type)
	require.Equal(t, "value", f.String)

	// Test JSON object
	f = dataField("root", `{"key1": "value1"}`)
	require.NotNil(t, f)
	require.Equal(t, "root", f.Key)
	require.Equal(t, zapcore.ReflectType, f.Type)

	buf, err := enc.EncodeEntry(entry, []zapcore.Field{f})
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Equal(t, `{"root":{"key1":"value1"}}`+"\n", buf.String())

	// Test JSON array
	f = dataField("root", `["value1", "value2"]`)
	require.NotNil(t, f)
	require.Equal(t, "root", f.Key)
	require.Equal(t, zapcore.ReflectType, f.Type)

	buf, err = enc.EncodeEntry(entry, []zapcore.Field{f})
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Equal(t, `{"root":["value1","value2"]}`+"\n", buf.String())

	// Test non-json content
	f = dataField("root", `{non-json content}`)
	require.NotNil(t, f)
	require.Equal(t, "root", f.Key)
	require.Equal(t, zapcore.StringType, f.Type)
	require.Equal(t, `{non-json content}`, f.String)

	buf, err = enc.EncodeEntry(entry, []zapcore.Field{f})
	require.NoError(t, err)
	require.NotNil(t, buf)
	require.Equal(t, `{"root":"{non-json content}"}`+"\n", buf.String())
}

func TestSignal(t *testing.T) {
	entry := zapcore.Entry{}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})

	// Simulate pairing.AccountData and pairing.Event without importing the package
	type Data struct {
		Name     string `json:"name" fake:"{firstname}"`
		Password string `json:"password"`
	}
	type Event struct {
		Data any `json:"data,omitempty"`
	}

	data := Data{}
	err := gofakeit.Struct(&data)
	require.NoError(t, err)

	event := Event{Data: data}

	f := dataField("event", event)
	require.NotNil(t, f)
	require.Equal(t, "event", f.Key)
	require.Equal(t, zapcore.ReflectType, f.Type)

	buf, err := enc.EncodeEntry(entry, []zapcore.Field{f})
	require.NoError(t, err)
	require.NotNil(t, buf)

	t.Logf("encoded event: %s", buf.String())

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	resultEvent, ok := result["event"]
	require.True(t, ok)
	require.NotNil(t, resultEvent)

	resultEventMap, ok := resultEvent.(map[string]interface{})
	require.True(t, ok)
	require.NotNil(t, resultEventMap)

	resultData, ok := resultEventMap["data"]
	require.True(t, ok)
	require.NotNil(t, resultData)

	resultDataMap, ok := resultData.(map[string]interface{})
	require.True(t, ok)
	require.NotNil(t, resultDataMap)
	require.Equal(t, redactionPlaceholder, resultDataMap["password"])
}
