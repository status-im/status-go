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

func getFunctionName(fn any) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func getShortFunctionName(fn any) string {
	fullName := getFunctionName(fn)
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}

// logAndCall logs request call details and executes the fn function if logging is enabled
func logAndCall(fn any, params ...any) any {
	defer func() {
		if r := recover(); r != nil {
			// we're not sure if request logging is enabled here, so we log it use default logger
			log.Error("panic found in logAndCall", "error", r, "stacktrace", string(debug.Stack()))
			panic(r)
		}
	}()

	var startTime time.Time
	var duration time.Duration

	methodName := getShortFunctionName(fn)

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
		duration = time.Since(startTime)
		paramsString := removeSensitiveInfo(fmt.Sprintf("%+v", params))
		respString := removeSensitiveInfo(fmt.Sprintf("%+v", resp))
		requestlog.GetRequestLogger().Debug(methodName, "params", paramsString, "resp", respString, "duration", duration)
	}

	return resp
}

func logAndCallString(fn any, params ...any) string {
	resp := logAndCall(fn, params...)
	if resp == nil {
		return ""
	}
	return resp.(string)
}

func removeSensitiveInfo(jsonStr string) string {
	// see related test for the usage of this function
	re := regexp.MustCompile(`(?i)(".*?(password|mnemonic|openseaAPIKey|poktToken|alchemyArbitrumMainnetToken|raribleTestnetAPIKey|alchemyOptimismMainnetToken|statusProxyBlockchainUser|alchemyEthereumSepoliaToken|alchemyArbitrumSepoliaToken|infuraToken|raribleMainnetAPIKey|alchemyEthereumMainnetToken).*?")\s*:\s*("[^"]*")`)
	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		parts := re.FindStringSubmatch(match)
		return fmt.Sprintf(`%s:"***"`, parts[1])
	})
}
