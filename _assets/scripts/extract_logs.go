package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var logPattern = regexp.MustCompile(`(?P<timestamp>[\d-]+T[\d:]+\.\d+Z).*(?P<action>sent-message:.*)\s+(?P<logData>{.*})`)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run script.go <filename>")
		os.Exit(1)
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	fmt.Printf("Timestamp\tMessageType\tContentType\tMessageID\tHashes\tRecipients\n")
	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains "sent-message"
		if strings.Contains(line, "sent-message") {
			match := logPattern.FindStringSubmatch(line)

			// Ensure the match is not nil and has expected groups
			if match != nil && len(match) > 3 {
				logTime, _ := time.Parse(time.RFC3339Nano, matchMap("timestamp", match))
				logData := matchMap("logData", match)

				var data map[string]interface{}
				if err := json.Unmarshal([]byte(logData), &data); err == nil {
					recipients := arrayToString(data["recipient"])
					messageID := fmt.Sprintf("%v", data["messageID"])
					messageType := fmt.Sprintf("%v", data["messageType"])
					contentType := fmt.Sprintf("%v", data["contentType"])
					hashes := arrayToString(data["hashes"])

					// Print the required information
					fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\t\n",
						logTime.Format(time.RFC3339Nano), messageType, contentType, messageID, hashes, recipients)
				}
			} else {
				fmt.Printf("Warning: Line does not match expected format: %s\n", line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %s\n", err)
	}
}

// Helper function to convert an array to a string
func arrayToString(arr interface{}) string {
	if arr != nil {
		switch v := arr.(type) {
		case []interface{}:
			var result []string
			for _, item := range v {
				result = append(result, fmt.Sprintf("%v", item))
			}
			return strings.Join(result, ", ")
		}
	}
	return ""
}

// Helper function to get the value of a named capture group from regex match
func matchMap(key string, matches []string) string {
	for i, name := range logPattern.SubexpNames() {
		if name == key {
			return matches[i]
		}
	}
	return ""
}
