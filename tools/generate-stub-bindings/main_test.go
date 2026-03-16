package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateFromHeader(t *testing.T) {
	dir := t.TempDir()
	headerPath := filepath.Join(dir, "libstatus.h")
	header := strings.Join([]string{
		"extern char* Foo(char* name, int count, bool ok);",
		"extern void Bar(char* state);",
		"extern void SetSignalEventCallback(void* cb);",
		"extern char* NoArgs();",
	}, "\n")
	if err := os.WriteFile(headerPath, []byte(header), 0o600); err != nil {
		t.Fatalf("write header: %v", err)
	}

	fns, err := parseHeader(headerPath)
	if err != nil {
		t.Fatalf("parse header: %v", err)
	}

	stub := genStubExports(fns, "libstatus.h")
	if strings.Contains(stub, "SetSignalEventCallback") {
		t.Fatalf("stub should skip SetSignalEventCallback")
	}
	if !strings.Contains(stub, "Foo") || !strings.Contains(stub, "Bar") || !strings.Contains(stub, "NoArgs") {
		t.Fatalf("stub missing expected functions")
	}
	if !strings.Contains(stub, "snprintf") {
		t.Fatalf("stub should stringify int args")
	}
	if !strings.Contains(stub, "? \"true\" : \"false\"") {
		t.Fatalf("stub should stringify bool args")
	}
	if !strings.Contains(stub, "Free(_out);") {
		t.Fatalf("void wrapper should free output")
	}
	if !strings.Contains(stub, "extern \"C\" char* Foo(") {
		t.Fatalf("non-void wrapper should return char*")
	}
	if !strings.Contains(stub, "statusgo_stub_callv(\"NoArgs\", nullptr, 0);") {
		t.Fatalf("no-args wrapper should pass nullptr for argv")
	}
	if strings.Contains(stub, "argv0[1]") {
		t.Fatalf("stub should not create dummy argv array for no-arg calls")
	}

	dispatch := genServiceDispatcher(fns, "libstatus.h")
	if strings.Contains(dispatch, "SetSignalEventCallback") {
		t.Fatalf("dispatcher should skip SetSignalEventCallback")
	}
	if !strings.Contains(dispatch, "toInt(argv[1])") {
		t.Fatalf("dispatcher should parse int args")
	}
	if !strings.Contains(dispatch, "toBool(argv[2])") {
		t.Fatalf("dispatcher should parse bool args")
	}
	if !strings.Contains(dispatch, "return (char*)strdup(\"\");") {
		t.Fatalf("void dispatch should return empty string")
	}
}
