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
	var output = prelude

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
			output += handleFile(file)
		}
	}

	fmt.Println(output)
}

func handleFunction(name string, funcDecl *ast.FuncDecl) string {
	params := funcDecl.Type.Params.List
	results := funcDecl.Type.Results

	// add export tag
	output := fmt.Sprintf("// export %s\n", name)
	// add initial func declaration
	output += fmt.Sprintf("func %s (", name)

	// iterate over parameters and correctly add the C type
	paramCount := 0
	for _, p := range params {
		for _, paramIdentity := range p.Names {
			if paramCount != 0 {
				output += ", "
			}
			paramCount++
			output += paramIdentity.Name

			typeString := fmt.Sprint(paramIdentity.Obj.Decl.(*ast.Field).Type)
			switch typeString {
			case stringType:
				output += " *C.char"
			case intType, boolType:
				output += " C.int"
			case unsafePointerType:
				output += " unsafe.Pointer"
			default:
				// ignore if the type is any different
				return ""
			}
		}
	}

	output += ")"

	// check if it has a return value, convert to CString if so and return
	if results != nil {
		output += " *C.char {\nreturn C.CString("
	} else {
		output += " {\n"

	}

	// call the mobile equivalent function
	output += fmt.Sprintf("mobile.%s(", name)

	// iterate through the parameters, convert to go types and close
	// the function call
	paramCount = 0
	for _, p := range params {
		for _, paramIdentity := range p.Names {
			if paramCount != 0 {
				output += ", "
			}
			paramCount++
			typeString := fmt.Sprint(paramIdentity.Obj.Decl.(*ast.Field).Type)
			switch typeString {
			case stringType:
				output += fmt.Sprintf("C.GoString(%s)", paramIdentity.Name)
			case intType:
				output += fmt.Sprintf("int(%s)", paramIdentity.Name)
			case unsafePointerType:
				output += paramIdentity.Name
			case boolType:
				output += paramIdentity.Name
				// convert int to bool
				output += " == 1"
			default:
				// ignore otherwise
				return ""
			}
		}
	}

	// close function call
	output += ")"

	// close conversion to CString
	if results != nil {
		output += ")\n"
	}

	// close function declaration
	output += "}\n"
	return output
}

func handleFile(parsedAST *ast.File) string {
	output := ""
	for name, obj := range parsedAST.Scope.Objects {
		// Ignore non-functions or non exported fields
		if obj.Kind != ast.Fun || !unicode.IsUpper(rune(name[0])) {
			continue
		}
		output += handleFunction(name, obj.Decl.(*ast.FuncDecl))
	}
	return output
}
