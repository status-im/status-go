package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type argKind string

const (
	kindCString argKind = "cstring"
	kindInt     argKind = "int"
	kindBool    argKind = "bool"
	kindPointer argKind = "pointer"
	kindUnknown argKind = "unknown"
)

type Arg struct {
	Name string
	Kind argKind
}

type Fn struct {
	Name string
	Ret  string
	Args []Arg
}

func normalizeType(t string) string {
	s := strings.TrimSpace(t)
	s = strings.ReplaceAll(s, "const ", "")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.ReplaceAll(s, " *", "*")
	s = strings.ReplaceAll(s, "* ", "*")
	return strings.TrimSpace(s)
}

func kindForType(t string) argKind {
	switch normalizeType(t) {
	case "char*":
		return kindCString
	case "int":
		return kindInt
	case "bool":
		return kindBool
	case "void*":
		return kindPointer
	default:
		return kindUnknown
	}
}

func parseArgs(raw string) []Arg {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "void" {
		return nil
	}
	parts := strings.Split(raw, ",")
	args := make([]Arg, 0, len(parts))
	nameRe := regexp.MustCompile(`^(.+?)\s+([A-Za-z_][A-Za-z0-9_]*)$`)
	for i, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		name := ""
		typ := p
		if m := nameRe.FindStringSubmatch(p); m != nil {
			typ = m[1]
			name = m[2]
		} else {
			name = fmt.Sprintf("a%d", i)
		}
		args = append(args, Arg{
			Name: name,
			Kind: kindForType(typ),
		})
	}
	return args
}

func parseHeader(path string) ([]Fn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	re := regexp.MustCompile(`^extern\s+(.+?)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\);`)
	var fns []Fn

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "extern ") {
			continue
		}
		if strings.HasPrefix(line, "extern \"C\"") {
			continue
		}
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		retType := normalizeType(m[1])
		name := m[2]
		argsRaw := strings.TrimSpace(m[3])
		if retType != "char*" && retType != "void" {
			fmt.Fprintf(os.Stderr, "skip %s: unsupported return type %q\n", name, retType)
			continue
		}
		fns = append(fns, Fn{
			Name: name,
			Ret:  retType,
			Args: parseArgs(argsRaw),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return fns, nil
}

func sigType(a Arg) string {
	switch a.Kind {
	case kindCString:
		return "const char*"
	case kindInt:
		return "int"
	case kindBool:
		return "bool"
	case kindPointer:
		return "void*"
	default:
		return "const char*"
	}
}

func genWrapper(fn Fn) string {
	if fn.Name == "Free" || fn.Name == "SetSignalEventCallback" {
		return ""
	}
	retIsVoid := fn.Ret == "void"
	retType := "char*"
	if retIsVoid {
		retType = "void"
	}

	var params []string
	var argvExprs []string
	var prelude []string

	for i, a := range fn.Args {
		argName := a.Name
		if argName == "" {
			argName = fmt.Sprintf("a%d", i)
		}
		params = append(params, fmt.Sprintf("%s %s", sigType(a), argName))

		switch a.Kind {
		case kindCString:
			argvExprs = append(argvExprs, argName)
		case kindInt:
			buf := fmt.Sprintf("arg_%d_buf", i)
			prelude = append(prelude, fmt.Sprintf("  char %s[32];", buf))
			prelude = append(prelude, fmt.Sprintf("  snprintf(%s, sizeof(%s), \"%%d\", %s);", buf, buf, argName))
			argvExprs = append(argvExprs, buf)
		case kindBool:
			buf := fmt.Sprintf("arg_%d_buf", i)
			prelude = append(prelude, fmt.Sprintf("  const char* %s = %s ? \"true\" : \"false\";", buf, argName))
			argvExprs = append(argvExprs, buf)
		default:
			argvExprs = append(argvExprs, "\"\"")
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("extern \"C\" %s %s(%s) {", retType, fn.Name, strings.Join(params, ", ")))
	if len(prelude) > 0 {
		lines = append(lines, prelude...)
	}
	if len(argvExprs) == 0 {
		if retIsVoid {
			lines = append(lines, fmt.Sprintf("  char* _out = statusgo_stub_callv(\"%s\", nullptr, 0);", fn.Name))
			lines = append(lines, "  Free(_out);")
			lines = append(lines, "  return;")
		} else {
			lines = append(lines, fmt.Sprintf("  return statusgo_stub_callv(\"%s\", nullptr, 0);", fn.Name))
		}
	} else {
		lines = append(lines, fmt.Sprintf("  const char* argv[] = {%s};", strings.Join(argvExprs, ", ")))
		if retIsVoid {
			lines = append(lines, fmt.Sprintf("  char* _out = statusgo_stub_callv(\"%s\", argv, %d);", fn.Name, len(argvExprs)))
			lines = append(lines, "  Free(_out);")
			lines = append(lines, "  return;")
		} else {
			lines = append(lines, fmt.Sprintf("  return statusgo_stub_callv(\"%s\", argv, %d);", fn.Name, len(argvExprs)))
		}
	}
	lines = append(lines, "}")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func genStubExports(fns []Fn, headerPath string) string {
	lines := []string{
		"// AUTO-GENERATED FILE. DO NOT EDIT.",
		"// Generated from: " + headerPath,
		"//",
		"// This exports status-go C API symbols for the UI-process stub (libstatus_stub.so).",
		"// Each function forwards to statusgo_stub_callv(method, argv, argc) implemented in statusgo_stub.cpp.",
		"",
		"#include <stddef.h>",
		"#include <stdio.h>",
		"",
		"extern \"C\" {",
		"  typedef void (*SignalCallback)(const char*);",
		"  void Free(void* p);",
		"  char* statusgo_stub_callv(const char* method, const char** argv, size_t argc);",
		"}",
		"",
	}

	for _, fn := range fns {
		if fn.Name == "Free" || fn.Name == "SetSignalEventCallback" {
			continue
		}
		lines = append(lines, genWrapper(fn))
	}

	return strings.Join(lines, "\n")
}

func genServiceDispatcher(fns []Fn, headerPath string) string {
	lines := []string{
		"// AUTO-GENERATED FILE. DO NOT EDIT.",
		"// Generated from: " + headerPath,
		"//",
		"// Dispatcher used by libstatus_service.so (service process).",
		"// Maps method name + string argv[] to real libstatus.so exports.",
		"",
		"#include <stddef.h>",
		"#include <string.h>",
		"#include <stdlib.h>",
		"",
		"extern \"C\" {",
		"  typedef void (*SignalCallback)(const char*);",
		"  void Free(void* p);",
		"}",
		"",
		"extern \"C\" {",
	}

	for _, fn := range fns {
		if fn.Name == "Free" || fn.Name == "SetSignalEventCallback" {
			continue
		}
		var params []string
		for i, a := range fn.Args {
			params = append(params, fmt.Sprintf("%s a%d", sigType(a), i))
		}
		retType := "char*"
		if fn.Ret == "void" {
			retType = "void"
		}
		lines = append(lines, fmt.Sprintf("  %s %s(%s);", retType, fn.Name, strings.Join(params, ", ")))
	}
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, "static int toInt(const char* s) { return s ? atoi(s) : 0; }")
	lines = append(lines, "static bool toBool(const char* s) { return s && (strcmp(s,\"true\")==0 || strcmp(s,\"1\")==0); }")
	lines = append(lines, "")
	lines = append(lines, "extern \"C\" char* statusgo_service_dispatch(const char* method, const char** argv, size_t argc) {")
	lines = append(lines, "  if (!method) method = \"\";")
	lines = append(lines, "")

	for _, fn := range fns {
		if fn.Name == "Free" || fn.Name == "SetSignalEventCallback" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  if (strcmp(method, \"%s\") == 0) {", fn.Name))
		if len(fn.Args) > 0 {
			lines = append(lines, fmt.Sprintf("    if (argc < %d) return (char*)strdup(\"{\\\"error\\\":\\\"bad argc\\\"}\");", len(fn.Args)))
		}
		var callArgs []string
		for i, a := range fn.Args {
			switch a.Kind {
			case kindCString:
				callArgs = append(callArgs, fmt.Sprintf("argv[%d] ? argv[%d] : \"\"", i, i))
			case kindInt:
				callArgs = append(callArgs, fmt.Sprintf("toInt(argv[%d])", i))
			case kindBool:
				callArgs = append(callArgs, fmt.Sprintf("toBool(argv[%d])", i))
			default:
				callArgs = append(callArgs, "\"\"")
			}
		}
		if fn.Ret == "void" {
			lines = append(lines, fmt.Sprintf("    %s(%s);", fn.Name, strings.Join(callArgs, ", ")))
			lines = append(lines, "    return (char*)strdup(\"\");")
		} else {
			lines = append(lines, fmt.Sprintf("    return %s(%s);", fn.Name, strings.Join(callArgs, ", ")))
		}
		lines = append(lines, "  }")
		lines = append(lines, "")
	}

	lines = append(lines, "  return (char*)strdup(\"{\\\"error\\\":\\\"unknown method\\\"}\");")
	lines = append(lines, "}")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func main() {
	headerPath := flag.String("header", "", "Path to libstatus.h (defaults to build/bin/libstatus.h)")
	outDir := flag.String("out-dir", "", "Output directory for generated files (defaults to build/bin)")
	flag.Parse()

	resolvedHeader := *headerPath
	if resolvedHeader == "" {
		resolvedHeader = resolveHeaderPath()
	}
	if resolvedHeader == "" {
		fmt.Fprintln(os.Stderr, "failed to resolve header path; pass --header")
		os.Exit(1)
	}

	outDirPath := *outDir
	if outDirPath == "" {
		outDirPath = filepath.Join("build", "bin")
	}
	if err := os.MkdirAll(outDirPath, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	fns, err := parseHeader(resolvedHeader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse header: %v\n", err)
		os.Exit(1)
	}

	stubOut := filepath.Join(outDirPath, "statusgo_stub_exports.cpp")
	serviceOut := filepath.Join(outDirPath, "statusgo_service_dispatch.cpp")

	if err := os.WriteFile(stubOut, []byte(genStubExports(fns, resolvedHeader)), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", stubOut, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", stubOut)

	if err := os.WriteFile(serviceOut, []byte(genServiceDispatcher(fns, resolvedHeader)), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", serviceOut, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", serviceOut)
}

func resolveHeaderPath() string {
	candidates := []string{
		filepath.Join("build", "bin", "libstatus.h"),
	}
	for _, p := range candidates {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
