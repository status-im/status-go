package callog

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

var sensitiveKeys = []string{
	"password",
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
func Call(logger *zap.Logger, fn any, params ...any) any {
	defer func() {
		if r := recover(); r != nil {
			logutils.ZapLogger().Error("panic found in call", zap.Any("error", r), zap.Stack("stacktrace"))
			panic(r)
		}
	}()

	var startTime time.Time

	requestLoggingEnabled := logger != nil
	if requestLoggingEnabled {
		startTime = time.Now()
	}

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

	if requestLoggingEnabled {
		duration := time.Since(startTime)
		methodName := getShortFunctionName(fn)
		paramsString := removeSensitiveInfo(fmt.Sprintf("%+v", params))
		respString := removeSensitiveInfo(fmt.Sprintf("%+v", resp))

		logger.Debug("call",
			zap.String("method", methodName),
			zap.String("params", paramsString),
			zap.String("resp", respString),
			zap.Duration("duration", duration),
		)
	}

	return resp
}

func CallWithResponse(logger *zap.Logger, fn any, params ...any) string {
	resp := Call(logger, fn, params...)
	if resp == nil {
		return ""
	}
	return resp.(string)
}

func removeSensitiveInfo(jsonStr string) string {
	// see related test for the usage of this function
	return sensitiveRegex.ReplaceAllStringFunc(jsonStr, func(match string) string {
		parts := sensitiveRegex.FindStringSubmatch(match)
		return fmt.Sprintf(`%s:"***"`, parts[1])
	})
}
