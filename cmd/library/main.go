package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"unicode"
)

func isCodeFile(info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "test.go")
}

func main() {
	var output strings.Builder
	output.WriteString(prelude)

	fset := token.NewFileSet()

	// Parse the whole `mobile/` directory, excluding test files
	parsedAST, err := parser.ParseDir(fset, "mobile/", isCodeFile, parser.AllErrors)
	if err != nil {
		fmt.Printf("Error parsing directory: %+v\n", err)
		os.Exit(1)
	}

	for _, a := range parsedAST {
		for _, file := range a.Files {
			// handle each file and append the output
			output.WriteString(handleFile(file))
		}
	}

	// To free memory allocated to strings
	output.WriteString("//export Free\n")
	output.WriteString("func Free (param unsafe.Pointer){\n")
	output.WriteString("C.free(param);\n")
	output.WriteString("}\n")

	fmt.Println(output.String())
}

func handleFunction(name string, funcDecl *ast.FuncDecl) string {
	params := funcDecl.Type.Params.List
	results := funcDecl.Type.Results

	// add export tag
	var output strings.Builder
	output.WriteString(fmt.Sprintf("//export %s\n", name))
	// add initial func declaration
	output.WriteString(fmt.Sprintf("func %s (", name))

	// iterate over parameters and correctly add the C type
	paramCount := 0
	for _, p := range params {
		for _, paramIdentity := range p.Names {
			if paramCount != 0 {
				output.WriteString(", ")
			}
			paramCount++
			output.WriteString(paramIdentity.Name)

			typeString := fmt.Sprint(paramIdentity.Obj.Decl.(*ast.Field).Type)
			// We match against the stringified type,
			// could not find a better way to match this
			switch typeString {
			case stringType:
				output.WriteString(" *C.char")
			case intType, boolType:
				output.WriteString(" C.int")
			case unsafePointerType:
				output.WriteString(" unsafe.Pointer")
			default:
				// ignore if the type is any different
				return ""
			}
		}
	}

	output.WriteString(")")

	// check if it has a return value, convert to CString if so and return
	if results != nil {
		output.WriteString(" *C.char {\nreturn C.CString(")
	} else {
		output.WriteString(" {\n")

	}

	// call the mobile equivalent function
	output.WriteString(fmt.Sprintf("mobile.%s(", name))

	// iterate through the parameters, convert to go types and close
	// the function call
	paramCount = 0
	for _, p := range params {
		for _, paramIdentity := range p.Names {
			if paramCount != 0 {
				output.WriteString(", ")
			}
			paramCount++
			typeString := fmt.Sprint(paramIdentity.Obj.Decl.(*ast.Field).Type)
			switch typeString {
			case stringType:
				output.WriteString(fmt.Sprintf("C.GoString(%s)", paramIdentity.Name))
			case intType:
				output.WriteString(fmt.Sprintf("int(%s)", paramIdentity.Name))
			case unsafePointerType:
				output.WriteString(paramIdentity.Name)
			case boolType:
				output.WriteString(paramIdentity.Name)
				// convert int to bool
				output.WriteString(" == 1")
			default:
				// ignore otherwise
				return ""
			}
		}
	}

	// close function call
	output.WriteString(")")

	// close conversion to CString
	if results != nil {
		output.WriteString(")\n")
	}

	// close function declaration
	output.WriteString("}\n")
	return output.String()
}

func handleFile(parsedAST *ast.File) string {
	var output strings.Builder
	for name, obj := range parsedAST.Scope.Objects {
		// Ignore non-functions or non exported fields
		if obj.Kind != ast.Fun || !unicode.IsUpper(rune(name[0])) {
			continue
		}
		output.WriteString(handleFunction(name, obj.Decl.(*ast.FuncDecl)))
	}

	return output.String()
}
