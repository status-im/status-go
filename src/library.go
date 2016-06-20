package main

// #ifdef __cplusplus
// extern "C" {
// #endif
//
// extern int runCreateAccount(const char*);
//
// #ifdef __cplusplus
// }
// #endif
import "C"
import (
	"fmt"
	"os"
	"strings"
)

//export doRunCreateAccount
func doRunCreateAccount(args *C.char) C.int {
	// This is equivalent to geth.main, just modified to handle the function arg passing
	if err := app.Run(strings.Split("statusgo "+C.GoString(args), " ")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}
	return 0
}
