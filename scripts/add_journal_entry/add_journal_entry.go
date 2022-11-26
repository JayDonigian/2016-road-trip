package main

import (
	"errors"
	"flag"
	"github.com/jaydonigian/2016roadtrip/scripts/add_journal_entry/journal"
	"log"
	"strings"
)

func main() {
	err := parseArgs()
	if err != nil {
		log.Fatalf("ERROR: while parsing arguments - %s", err.Error())
	}

	info, err := journal.Unmarshal("journal/journal.json")
	if err != nil {
		log.Fatalf("ERROR: while unmarshaling JSON file - %s", err.Error())
	}

	err = info.AddYearToDates(2016)
	if err != nil {
		log.Fatalf("ERROR: while parsing date field - %s", err.Error())
	}

	template := "journal/templates/template.md"
	for _, entry := range info.MissingEntries() {
		err = entry.NewFromTemplate(template)
		if err != nil {
			log.Fatalf("ERROR: while creating from template file - %s", err)
		}
	}
}

func parseArgs() error {
	filepathArg := flag.String("filepath", "", "filepath is a required argument that should point to a JSON file")

	flag.Parse()

	if *filepathArg == "" {
		return errors.New("filepath is a required argument")
	}

	if !strings.HasSuffix(*filepathArg, ".json") {
		return errors.New("the provided file should be a JSON file")
	}

	return nil
}
