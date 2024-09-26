//go:generate go run main.go

/*
This script generates a Go file with a list of supported endpoints based on the public functions in `mobile/status.go`.
The output has 3 sections:
- Endpoints with a response of type `string`
- Endpoints with no arguments and a response of type `string`
- Unsupported endpoints: those have non-standard signatures
Deprecated functions are ignored.
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

const (
	inputFilePath    = "../../../../mobile/status.go"
	templateFilePath = "./endpoints_template.txt"
	outputFilePath   = "../endpoints.go"
)

var (
	// Regular expressions extracted as global variables
	publicFunc                   = regexp.MustCompile(`func\s+([A-Z]\w+)\(.*\).*\{`)
	publicFuncWithArgsPattern    = regexp.MustCompile(`^func\s+([A-Z]\w*)\((\w|\s)+\)\s+string\s+\{$`)
	publicFuncWithoutArgsPattern = regexp.MustCompile(`^func\s+([A-Z]\w*)\(\)\s+string\s+\{$`)
	funcNamePattern              = regexp.MustCompile(`^func\s+([A-Z]\w*)\(`)
)

type TemplateData struct {
	PackageName          string
	FunctionsWithResp    []string
	FunctionsNoArgs      []string
	UnsupportedEndpoints []string
	DeprecatedEndpoints  []string
}

func main() {
	// Open the Go source file
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Printf("Failed to open file: %s\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var publicFunctionsWithArgs []string
	var publicFunctionsWithoutArgs []string
	var unsupportedFunctions []string
	var deprecatedFucntions []string
	var isDeprecated bool
	var packageName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect package name
		if strings.HasPrefix(line, "package ") {
			packageName = strings.TrimPrefix(line, "package ")
		}

		// Check for deprecation comment
		if isDeprecatedComment(line) {
			isDeprecated = true
			continue
		}

		if !publicFunc.MatchString(line) {
			continue
		}

		functionName := extractFunctionName(line)

		if isDeprecated {
			isDeprecated = false
			deprecatedFucntions = append(deprecatedFucntions, functionName)
		}

		switch {
		case isPublicFunctionWithArgs(line):
			publicFunctionsWithArgs = append(publicFunctionsWithArgs, functionName)
		case isPublicFunctionWithoutArgs(line):
			publicFunctionsWithoutArgs = append(publicFunctionsWithoutArgs, functionName)
		default:
			unsupportedFunctions = append(unsupportedFunctions, functionName)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return
	}

	// Prepare the template data
	data := TemplateData{
		PackageName:          packageName,
		FunctionsWithResp:    publicFunctionsWithArgs,
		FunctionsNoArgs:      publicFunctionsWithoutArgs,
		UnsupportedEndpoints: unsupportedFunctions,
		DeprecatedEndpoints:  deprecatedFucntions,
	}

	// Load and parse the template
	tmpl, err := template.ParseFiles(templateFilePath)
	if err != nil {
		fmt.Printf("Failed to parse template file: %s\n", err)
		return
	}

	// Create the output file
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Failed to create output file: %s\n", err)
		return
	}
	defer outputFile.Close()

	// Execute the template and write the result to the output file
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		fmt.Printf("Failed to execute template: %s\n", err)
		return
	}

	fmt.Println("Generated endpoints file:", outputFilePath)
}

// Function to check if a line contains a public function with a response of string
func isPublicFunctionWithArgs(line string) bool {
	return publicFuncWithArgsPattern.MatchString(line)
}

// Function to check if a line contains a public function with not arguments and a response of string
func isPublicFunctionWithoutArgs(line string) bool {
	return publicFuncWithoutArgsPattern.MatchString(line)
}

// Function to extract the public function name from a line
func extractFunctionName(line string) string {
	matches := funcNamePattern.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Function to check if a comment indicates a deprecated function
func isDeprecatedComment(line string) bool {
	return strings.Contains(line, "// Deprecated:")
}
