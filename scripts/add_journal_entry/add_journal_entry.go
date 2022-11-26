package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	err := parseArgs()
	if err != nil {
		log.Fatalf("ERROR: while parsing arguments - %s", err.Error())
	}

	info, err := unmarshal("journal/journal.json")
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

type replacement struct {
	find, replace string
}

// TODO: This has nested for-loops, optimize it if you can.
func apply(templatePath string, replacements []replacement) ([]string, error) {
	sourceFileStat, err := os.Stat(templatePath)
	if err != nil {
		return nil, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", templatePath)
	}

	source, err := os.Open(templatePath)
	if err != nil {
		return nil, err
	}
	defer source.Close()

	var lines []string
	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		line = findAndReplace(line, replacements)
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func findAndReplace(s string, reps []replacement) string {
	for _, rep := range reps {
		if strings.Contains(s, rep.find) {
			s = strings.Replace(s, rep.find, rep.replace, -1)
		}
	}
	return s
}
