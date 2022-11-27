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

	j, err := journal.New("journal/journal.json")
	if err != nil {
		log.Fatalf("ERROR: while creating journal - %s", err.Error())
	}

	var lines []string
	for _, e := range j.MissingEntries() {
		lines, err = e.ApplyToTemplate("journal/templates/template.md")
		if err != nil {
			log.Fatalf("ERROR: while creating from template file - %s", err)
		}

		err = j.Write(e, lines)
		if err != nil {
			log.Fatalf("ERROR: while creating from template file - %s", err)
		}

		err = j.WriteIndex(e)
		if err != nil {
			log.Fatalf("ERROR: while creating from template file - %s", err)
		}
	}

	err = j.Save()
	if err != nil {
		log.Printf("WARNING: while saving journal - %s", err.Error())
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
