package callog

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/sentry"
)

const redactionPlaceholder = "***"

var sensitiveKeys = []string{
	"password",
	"newPassword",
	"mnemonic",
	"openseaAPIKey",
	"poktToken",
	"alchemyArbitrumMainnetToken",
	"raribleTestnetAPIKey",
	"alchemyOptimismMainnetToken",
	"statusProxyBlockchainUser",
	"alchemyEthereumSepoliaToken",
	"alchemyArbitrumSepoliaToken",
	"infuraToken",
	"raribleMainnetAPIKey",
	"alchemyEthereumMainnetToken",
	"alchemyOptimismSepoliaToken",
	"verifyENSURL",
	"verifyTransactionURL",
	"gifs/api-key",
}

var sensitiveRegexString = fmt.Sprintf(`(?i)(".*?(%s).*?")\s*:\s*("[^"]*")`, strings.Join(sensitiveKeys, "|"))

var sensitiveRegex = regexp.MustCompile(sensitiveRegexString)

func getFunctionName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func getShortFunctionName(fn any) string {
	fullName := getFunctionName(fn)
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}

// Call executes the given function and logs request details if logging is enabled
//
// Parameters:
//   - fn: The function to be executed
//   - params: A variadic list of parameters to be passed to the function
//
// Returns:
//   - The result of the function execution (if any)
//
// Functionality:
// 1. Sets up panic recovery to log and re-panic
// 2. Records start time if request logging is enabled
// 3. Uses reflection to Call the given function
// 4. If request logging is enabled, logs method name, parameters, response, and execution duration
// 5. Removes sensitive information before logging
func Call(logger, requestLogger *zap.Logger, fn any, params ...any) any {
	defer Recover(logger)

	startTime := requestStartTime(requestLogger != nil)
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	if fnType.Kind() != reflect.Func {
		panic("fn must be a function")
	}

	args := make([]reflect.Value, len(params))
	for i, param := range params {
		args[i] = reflect.ValueOf(param)
	}

	results := fnValue.Call(args)

	var resp any

	if len(results) > 0 {
		resp = results[0].Interface()
	}

	if requestLogger != nil {
		methodName := getShortFunctionName(fn)
		LogCall(requestLogger, methodName, params, resp, startTime)
	}

	return resp
}

func CallWithResponse(logger, requestLogger *zap.Logger, fn any, params ...any) string {
	resp := Call(logger, requestLogger, fn, params...)
	if resp == nil {
		return ""
	}
	return resp.(string)
}

func removeSensitiveInfo(jsonStr string) string {
	// see related test for the usage of this function
	return sensitiveRegex.ReplaceAllStringFunc(jsonStr, func(match string) string {
		parts := sensitiveRegex.FindStringSubmatch(match)
		return fmt.Sprintf(`%s:"%s"`, parts[1], redactionPlaceholder)
	})
}

func requestStartTime(enabled bool) time.Time {
	if !enabled {
		return time.Time{}
	}
	return time.Now()
}

func Recover(logger *zap.Logger) {
	err := recover()
	if err == nil {
		return
	}

	logger.Error("panic found in call",
		zap.Any("error", err),
		zap.Stack("stacktrace"))

	sentry.RecoverError(err)

	panic(err)
}

func LogCall(logger *zap.Logger, method string, params any, resp any, startTime time.Time) {
	if logger == nil {
		return
	}
	duration := time.Since(startTime)
	logger.Debug("call",
		zap.String("method", method),
		zap.Duration("duration", duration),
		dataField("request", params),
		dataField("response", resp),
	)
}

func LogSignal(logger *zap.Logger, eventType string, event interface{}) {
	if logger == nil {
		return
	}
	logger.Debug("signal",
		zap.String("type", eventType),
		dataField("event", event),
	)
}

func dataField(name string, data any) zap.Field {
	dataString := removeSensitiveInfo(marshalData(data))
	var paramsParsed any
	if json.Unmarshal([]byte(dataString), &paramsParsed) == nil {
		return zap.Any(name, paramsParsed)
	}
	return zap.String(name, dataString)
}

func marshalData(data any) string {
	switch d := data.(type) {
	case string:
		return d
	default:
		bytes, err := json.Marshal(d)
		if err != nil {
			return "<failed to marshal value>"
		}
		return string(bytes)
	}
}
