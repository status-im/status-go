package statusgo

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/logutils/requestlog"
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

// call executes the given function and logs request details if logging is enabled
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
// 3. Uses reflection to call the given function
// 4. If request logging is enabled, logs method name, parameters, response, and execution duration
// 5. Removes sensitive information before logging
func call(fn any, params ...any) any {
	defer func() {
		if r := recover(); r != nil {
			// we're not sure if request logging is enabled here, so we log it use default logger
			log.Error("panic found in call", "error", r, "stacktrace", string(debug.Stack()))
			panic(r)
		}
	}()

	var startTime time.Time

	if requestlog.IsRequestLoggingEnabled() {
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

	if requestlog.IsRequestLoggingEnabled() {
		duration := time.Since(startTime)
		methodName := getShortFunctionName(fn)
		paramsString := removeSensitiveInfo(fmt.Sprintf("%+v", params))
		respString := removeSensitiveInfo(fmt.Sprintf("%+v", resp))
		requestlog.GetRequestLogger().Debug(methodName, "params", paramsString, "resp", respString, "duration", duration)
	}

	return resp
}

func callWithResponse(fn any, params ...any) string {
	resp := call(fn, params...)
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
