package main

import (
	"os"
	"bufio"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"fmt"
)

type definitionGetter func(filePath string, lineNumber int, charPosition int) (string, int, error)

// checkFileForGoroutines scans a Go file for any `go` statements (goroutines)
func checkFileForGoroutines(filePath string, definition definitionGetter, logger *zap.Logger) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Error opening file", zap.String("file", filePath), zap.Error(err))
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNumber int
	// Regex for non-anonymous function/method calls: `go functionName()`
	regex := regexp.MustCompile(`go\s+(\.|\w)+\(\)$`)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text() // Do not trim spaces here

		lineLogger := logger.With(
			zap.String("file", filePath),
			zap.Int("line", lineNumber))

		// Detect anonymous goroutines
		if strings.Contains(line, "go func") {
			lineLogger.Debug("Found anonymous goroutine", zap.String("lineContent", line))
			checkFirstLineInFunctionBody(filePath, lineNumber, logger)
			continue
		}

		// Detect non-anonymous goroutines using regex
		if !regex.MatchString(line) {
			continue
		}

		// Find the position of the first occurrence of "()"
		cursorPos := strings.Index(line, "()")
		if cursorPos == -1 {
			lineLogger.Error("failed to find function call")
			continue
		}

		lineLogger.Debug("Found non-anonymous goroutine call",
			zap.Int("cursor", cursorPos),
			zap.String("lineContent", line),
		)

		cursorPos -= 2

		// Calculate the cursor position by adjusting for tabs (counting tabs as 4 characters)
		//tabs := strings.Count(line[:cursorPos], "\t")
		//adjustedCursorPos := cursorPos - (tabs * 4) // Subtract 3 for each tab since a tab counts as 4 chars

		defFilePath, defLineNumber, err := definition(filePath, lineNumber-1, cursorPos)
		if err != nil {
			lineLogger.Error("failed to find function", zap.Error(err))
			continue
		}

		checkFirstLineInFunctionBody(defFilePath, defLineNumber, logger)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("failed to read file", zap.Error(err))
	}
}

// checkFirstLineInFunctionBody checks the first line inside a function body for `defer gocommon.utils.LogOnPanic()`
func checkFirstLineInFunctionBody(filePath string, startLine int, givenLogger *zap.Logger) {
	logger := givenLogger.With(zap.String("file", filePath), zap.Int("startLine", startLine))
	logger.Debug("checking function body")

	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Error opening file", zap.Error(err))
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
		url := fmt.Sprintf("%s:%d", filePath, startLine)

		if strings.Contains(line, "LogOnPanic()") {
			givenLogger.Info("found defer gocommon.LogOnPanic() in function", zap.String("url", url))
		} else {
			givenLogger.Warn("missing defer gocommon.LogOnPanic() in function", zap.String("url", url))
		}

		return
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading file", zap.Error(err))
	}
}
