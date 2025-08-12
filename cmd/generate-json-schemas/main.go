package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/personal"
)

var types = []any{
	protocol.MessengerResponse{},
	[]*ens.UsernameDetail{},
	[]*communities.RequestToJoin{},
	[]personal.SignParams{},
}

var (
	outDir = flag.String("out-dir", ".", "output directory for JSON schema files")
)

func process(t any, path string) error {
	schema := jsonschema.Reflect(t)
	json, err := schema.MarshalJSON()
	if err != nil {
		return errors.Wrap(err, "failed to marshal schema")
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	_, err = file.Write(json)
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}

func main() {
	flag.Parse()

	for _, t := range types {
		typeString := reflect.TypeOf(t)
		filename := fmt.Sprintf("%s.json", typeString)
		filepath := path.Join(*outDir, filename)

		err := process(t, filepath)
		if err != nil {
			log.Printf("Failed to process type '%v' - %v", typeString, err)
			continue
		}

		log.Printf("Generated schema for type '%v': %s", typeString, filepath)
	}
}
