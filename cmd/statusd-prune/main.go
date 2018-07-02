package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/mailserver"
)

var (
	dbPath         = flag.String("db", "", "Path to wnode database folder")
	lowerTimestamp = flag.Int("lower", 0, "Removes messages sent starting from this timestamp")
	upperTimestamp = flag.Int("upper", 0, "Removes messages sent up to this timestamp")
)

func missingFlag(f string) {
	log.Printf("flag -%s is required", f)
	flag.Usage()
	os.Exit(1)
}

func validateRange(lower, upper int) error {
	if upper <= lower {
		return fmt.Errorf("upper value must be greater than lower value")
	}

	if lower < 0 || upper < 0 {
		return fmt.Errorf("upper and lower values must be greater than zero")
	}

	return nil
}

func init() {
	flag.Parse()

	if *dbPath == "" {
		missingFlag("db")
	}

	if *upperTimestamp == 0 {
		missingFlag("upper")
	}
}

func main() {
	db, err := db.Open(*dbPath, nil)
	if err != nil {
		log.Fatal(err)
	}

	c := mailserver.NewCleanerWithDB(db)

	if err = validateRange(*lowerTimestamp, *upperTimestamp); err != nil {
		log.Fatal(err)
	}

	lower := uint32(*lowerTimestamp)
	upper := uint32(*upperTimestamp)

	n, err := c.Prune(lower, upper)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("removed %d messages.\n", n)
}
