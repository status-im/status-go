package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var logPattern = regexp.MustCompile(`(?P<timestamp>[\d-]+T[\d:]+\.\d+Z).*(?P<action>sent-message:.*)\s+(?P<logData>{.*})`)

const (
	filtersNotMatched = "filters did match"
	filtersMatched    = "filters did not match"
	storeNodeMessage  = "received waku2 store message"
)

func receivedMessageCountInfo(scanner *bufio.Scanner) {
	fmt.Printf("Matching\tNot matching\tStore node\tLive\tTotal\n")
	filtersNotMatchedCount := 0
	filtersMatchedCount := 0
	storeNodeReceivedMessageCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains "sent-message"
		if strings.Contains(line, filtersNotMatched) {
			filtersNotMatchedCount++
		} else if strings.Contains(line, filtersMatched) {
			filtersMatchedCount++
		} else if strings.Contains(line, storeNodeMessage) {
			storeNodeReceivedMessageCount++
		}
		// Print the required information
	}
	fmt.Printf("%d\t%d\t%d\t%d\t%d\n", filtersNotMatchedCount, filtersMatchedCount, storeNodeReceivedMessageCount, filtersNotMatchedCount+filtersMatchedCount-storeNodeReceivedMessageCount, filtersNotMatchedCount+filtersMatchedCount)

}

func messagesInfo(scanner *bufio.Scanner) {
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
}

func main() {
	messagesFlag := flag.Bool("messages", false, "Process sent messages in the log file")
	receivedMessageCountFlag := flag.Bool("received-message-count", false, "Count the number of sent messages in the log file")

	// Parse the flags
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Println("Usage: go run script.go -messages -received-message-count <filename>")
		os.Exit(1)
	}

	filename := flag.Arg(0)
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if !*messagesFlag && !*receivedMessageCountFlag {
		*messagesFlag = true
	}

	scanner := bufio.NewScanner(file)

	if *messagesFlag {
		messagesInfo(scanner)
	} else if *receivedMessageCountFlag {
		receivedMessageCountInfo(scanner)
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
