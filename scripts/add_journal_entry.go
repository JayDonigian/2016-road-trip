package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("ERROR: This script requires an argument specifying the date of the journal entry in the format: mm-dd\n.")
	}

	entryDate := os.Args[1]
	_, err := time.Parse("01-02", entryDate)
	if err != nil {
		log.Fatalf("ERROR: %s is not a valid date. Use the format: mm-dd\n.", entryDate)
	}

	err = os.Chdir("journal")
	if err != nil {
		log.Fatalf("ERROR: unable to find the journal directory\n.")
	}

	// check if files exist for date, warn of missing photos, exit script if journal entry already exists
	mapsDir := "maps/"
	totalsDir := mapsDir + "totals/"

	mapFileName := mapsDir + entryDate + ".png"
	_, err = os.Stat(mapFileName)
	if err != nil {
		log.Printf("WARNING: %s does not exist\n", mapFileName)
	}

	totalsFileName := totalsDir + entryDate + "-total.png"
	_, err = os.Stat(totalsFileName)
	if err != nil {
		log.Printf("WARNING: %s does not exist\n", totalsFileName)
	}

	journalFile := entryDate + ".md"
	_, err = os.Stat(journalFile)
	if err == nil {
		log.Fatalf("ERROR: %s already exists\n", journalFile)
	}

	// create a new entry from the template
	err = copyTemplate(journalFile, entryDate)
	if err != nil {
		log.Fatalf("ERROR: an error was encountered while trying to copy the template file: %s", err)
	}
}

func copyTemplate(dst, date string) error {
	src := "template.md"

	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	var lines []string
	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "mm/dd") {
			slashDate := strings.Replace(date, "-", "/", -1)
			line = strings.Replace(line, "mm/dd", slashDate, -1)
		}
		if strings.Contains(line, "`mm-dd`") {
			line = strings.Replace(line, "`mm-dd`", date, -1)
		}
		if strings.Contains(line, "mm-dd") {
			line = strings.Replace(line, "mm-dd", date, -1)
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	writer := bufio.NewWriter(destination)
	defer writer.Flush()

	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}

	return nil
}
