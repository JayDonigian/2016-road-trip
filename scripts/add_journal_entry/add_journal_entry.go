package main

import (
	"github.com/jaydonigian/2016roadtrip/scripts/add_journal_entry/journal"
	"log"
)

func main() {
	j, err := journal.New("journal/journal.json")
	if err != nil {
		log.Fatalf("ERROR: while creating journal - %s", err.Error())
	}

	for _, e := range j.MissingEntries() {
		err = j.Write(e)
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
