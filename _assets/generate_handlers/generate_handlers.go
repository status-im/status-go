//go:generate go run generate_handlers.go

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// EnumType defines the type of the protobuf enum
type EnumType struct {
	Name   string
	Values []string
}

// MethodInfo holds information about a method
type MethodInfo struct {
	ProtobufName string
	MethodName   string
	EnumValue    string
	ProcessRaw   bool
	SyncMessage  bool
}

func main() {
	inputFile := "../../protocol/protobuf/application_metadata_message.proto"
	outputFile := "../../protocol/messenger_handlers.go"
	templateFile := "./generate_handlers_template.txt"
	enumName := "Type"

	// Load the protobuf file
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	templateFileContent, err := os.ReadFile(templateFile)
	if err != nil {
		fmt.Println("Failed to read template:", err)
		os.Exit(1)
	}

	// Extract enum values
	enum := extractEnum(content, enumName)

	// Prepare method information
	var methodInfos []MethodInfo
	for _, value := range enum.Values {
		protobufName := toCamelCase(value)
		if protobufName == "Unknown" || strings.HasPrefix(value, "DEPRECATED") {
			continue
		}
		methodName := "handle" + protobufName + "Protobuf"

		info := MethodInfo{MethodName: methodName, ProtobufName: protobufName, EnumValue: value}

		if strings.HasPrefix(value, "SYNC_") {
			info.SyncMessage = true
		}

		if protobufName == "PushNotificationRegistration" {
			info.ProcessRaw = true
		}

		methodInfos = append(methodInfos, info)
	}

	// Generate code
	templateCode := string(templateFileContent)

	tmpl, err := template.New("handlers").Parse(templateCode)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	output, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer output.Close()

	err = tmpl.Execute(output, methodInfos)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("Generated handlers in %s for %s enum.\n", outputFile, enumName)
}

func extractEnum(content []byte, enumName string) EnumType {
	enumPattern := fmt.Sprintf(`enum\s+%s\s*{([^}]+)}`, enumName)
	re := regexp.MustCompile(enumPattern)
	match := re.FindStringSubmatch(string(content))

	if len(match) != 2 {
		fmt.Println("Enum not found")
		os.Exit(1)
	}

	valuesPattern := `(?m)^\s*([A-Z_0-9]+)\s*=\s*\d+;`
	re = regexp.MustCompile(valuesPattern)
	valueMatches := re.FindAllStringSubmatch(match[1], -1)

	values := make([]string, len(valueMatches))
	for i, match := range valueMatches {
		values[i] = strings.TrimSpace(match[1])
	}

	return EnumType{Name: enumName, Values: values}
}

func toCamelCase(s string) string {
	words := strings.Split(strings.ToLower(s), "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	return strings.Join(words, "")
}
