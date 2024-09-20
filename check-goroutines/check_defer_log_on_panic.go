package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <directory>")
		return
	}

	// Initialize logger with colors
	handler := log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, handler))

	dir := os.Args[1]
	log.Info("Starting analysis...", "directory", dir)

	// Step 1: Scan all files and look for `go` calls
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("Error walking the path", "path", dir, "error", err)
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		log.Info("Scanning Go file", "file", path)
		checkFileForGoroutines(path)
		return nil
	})

	if err != nil {
		log.Error("Error during file walk", "error", err)
	}

	log.Info("Analysis complete")
}

// checkFileForGoroutines scans a Go file for any `go` statements (goroutines)
func checkFileForGoroutines(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Error("Error opening file", "file", filePath, "error", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNumber int
	// Regex for non-anonymous function/method calls: `go functionName()`
	regex := regexp.MustCompile(`go\s+(\.|\w)+\(\)$`)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text() // Do not trim spaces here

		// Detect anonymous goroutines
		if strings.Contains(line, "go func") {
			log.Info("Found anonymous goroutine", "file", filePath, "line", lineNumber, "lineContent", line)
			checkAnonymousGoroutine(filePath, lineNumber)
			continue
		}

		// Detect non-anonymous goroutines using regex
		if regex.MatchString(line) {
			log.Info("Found non-anonymous goroutine", "file", filePath, "line", lineNumber, "lineContent", line)

			// Find the position of the first occurrence of "()"
			cursorPos := strings.Index(line, "()")
			if cursorPos == -1 {
				log.Error("Failed to find function call", "file", filePath, "line", lineNumber)
				continue
			}

			// Calculate the cursor position by adjusting for tabs (counting tabs as 4 characters)
			//tabs := strings.Count(line[:cursorPos], "\t")
			//adjustedCursorPos := cursorPos - (tabs * 4) // Subtract 3 for each tab since a tab counts as 4 chars

			checkNamedFunction(filePath, lineNumber, cursorPos)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Error reading file", "file", filePath, "error", err)
	}
}

// checkAnonymousGoroutine checks if an anonymous goroutine has `defer utils.LogOnPanic()`
func checkAnonymousGoroutine(filePath string, lineNumber int) {
	//log.Debug("Checking anonymous goroutine", "file", filePath, "line", lineNumber)

	// Open the file again and scan from the `go func` line onwards
	file, err := os.Open(filePath)
	if err != nil {
		log.Error("Error opening file", "file", filePath, "error", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentLine int
	for scanner.Scan() {
		currentLine++
		if currentLine <= lineNumber {
			continue
		}

		line := scanner.Text()
		// First line of the function body
		if strings.Contains(line, "defer utils.LogOnPanic()") {
			log.Info("Found defer utils.LogOnPanic() in anonymous function", "file", filePath, "line", lineNumber)
		} else {
			log.Warn("Missing defer utils.LogOnPanic() in anonymous function", "file", filePath, "line", lineNumber)
		}
		return
	}

	if err := scanner.Err(); err != nil {
		log.Error("Error reading file", "file", filePath, "error", err)
	}
}

// extractFunctionNameAndCursorPosition extracts the function or method name and calculates the cursor position
func extractFunctionNameAndCursorPosition(line string, matches []string) (string, int) {
	funcName := matches[1]

	// Calculate the cursor position (count tabs as one character)
	regex := regexp.MustCompile(`\bgo\s+`)
	cursorPos := regex.FindStringIndex(line)[1] // Position after `go ` keyword

	return funcName, cursorPos
}

// checkNamedFunction uses `gopls` to find the definition of a named function/method and checks its first line
func checkNamedFunction(filePath string, lineNumber, charPos int) {
	log.Debug("Checking named function", "file", filePath, "line", lineNumber, "char", charPos)

	// Use `gopls` to find the definition of the function/method
	cmd := exec.Command("gopls", "definition", fmt.Sprintf("%s:%d:%d", filePath, lineNumber, charPos))
	output, err := cmd.Output()
	if err != nil {
		log.Error("Error running gopls definition", "file", filePath, "line", lineNumber, "error", err)
		return
	}

	definitionOutput := string(output)
	// Parse the definition output to find the file and line number of the function definition
	parseAndCheckFunctionDefinition(definitionOutput)
}

// parseAndCheckFunctionDefinition parses the output of `gopls definition` and checks the function body
func parseAndCheckFunctionDefinition(definitionOutput string) {
	// The output of `gopls definition` will contain the file path and position of the function definition
	// Example output might be:
	// /path/to/file.go:23:5
	log.Debug("Parsed definition", "definition", definitionOutput)

	// Extract file path and line number from the output
	parts := strings.Split(definitionOutput, ":")
	if len(parts) < 2 {
		log.Error("Failed to parse gopls definition output", "output", definitionOutput)
		return
	}
	defFilePath := parts[0]
	lineNumber := atoi(parts[1])

	// Open the file and check the first statement inside the function body
	checkFirstLineInFunctionBody(defFilePath, lineNumber)
}

// checkFirstLineInFunctionBody checks the first line inside a function body for `defer utils.LogOnPanic()`
func checkFirstLineInFunctionBody(filePath string, startLine int) {
	log.Debug("Checking function body", "file", filePath, "startLine", startLine)

	file, err := os.Open(filePath)
	if err != nil {
		log.Error("Error opening file", "file", filePath, "error", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentLine int
	for scanner.Scan() {
		currentLine++
		if currentLine <= startLine {
			continue
		}

		line := scanner.Text()

		if strings.Contains(line, "defer utils.LogOnPanic()") {
			log.Info("Found defer utils.LogOnPanic() in function", "file", filePath, "line", startLine)
		} else {
			log.Warn("Missing defer utils.LogOnPanic() in function", "file", filePath, "line", startLine)
		}

		return
	}

	if err := scanner.Err(); err != nil {
		log.Error("Error reading file", "file", filePath, "error", err)
	}
}

// atoi is a helper to safely convert a string to an int
func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
