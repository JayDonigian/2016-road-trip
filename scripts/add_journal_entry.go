package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	info, err := parseArgs()
	if err != nil {
		log.Fatalf("ERROR: while parsing arguments - %s", err.Error())
	}

	err = checkFiles(info.date.Format("01-02"))
	if err != nil {
		log.Fatalf("ERROR: while checking files - %s", err.Error())
	}

	err = createFromTemplate(info)
	if err != nil {
		log.Fatalf("ERROR: while copying the template file - %s", err)
	}
}

type entryInfo struct {
	date                                      time.Time
	startShort, startLong                     string
	endShort, endLong                         string
	mileage, previousMileage                  int
	budget, previousSpend, previousTotalSpend string
}

func parseArgs() (entryInfo, error) {
	dateArg := flag.String("date", "", "date is a required argument in the form -date=mm-dd")
	startShortArg := flag.String("start", "", "start is a short description (1-3 words) of the starting location")
	endShortArg := flag.String("end", "", "end is a short description (1-3 words) of the ending location")
	startLongArg := flag.String("start-long", "", "start is a longer description (name, state, country) of the starting location")
	endLongArg := flag.String("end-long", "", "end is a longer description (name, state, country) of the ending location")
	milesArg := flag.Int("miles", 0, "miles is the number of miles driven on the day of the journal entry")

	flag.Parse()

	stringArgs := []*string{dateArg, startShortArg, endShortArg, startLongArg, endLongArg}
	for _, arg := range stringArgs {
		if *arg == "" {
			return entryInfo{}, errors.New("missing a required argument")
		}
	}

	if *milesArg == 0 {
		return entryInfo{}, errors.New("missing a required argument")
	}

	t, err := time.Parse("01-02", *dateArg)
	if err != nil {
		return entryInfo{}, err
	}
	entryDate := time.Date(2016, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())

	return entryInfo{
		date:       entryDate,
		startShort: *startShortArg,
		startLong:  *startLongArg,
		endShort:   *endShortArg,
		endLong:    *endLongArg,
		mileage:    *milesArg,
	}, nil
}

func checkFiles(formattedDate string) error {
	if _, err := os.Stat("journal/maps/" + formattedDate + ".png"); errors.Is(err, os.ErrNotExist) {
		log.Printf("WARNING: map for %s does not exist\n", formattedDate)
	}

	if _, err := os.Stat("journal/maps/totals/" + formattedDate + "-total.png"); errors.Is(err, os.ErrNotExist) {
		log.Printf("WARNING: total map for %s does not exist\n", formattedDate)
	}

	if _, err := os.Stat("journal/" + formattedDate + ".md"); err == nil {
		return errors.New(fmt.Sprintf("journal entry for %s already exists\n", formattedDate))
	}

	return nil
}

func createFromTemplate(entry entryInfo) error {
	previousEntry, err := getInfo(entry.date.AddDate(0, 0, -1))
	if err != nil {
		return err
	}

	entry.previousMileage = previousEntry.mileage
	entry.previousSpend = previousEntry.budget
	entry.previousTotalSpend = previousEntry.previousTotalSpend

	lines, err := applyTemplate(entry)
	if err != nil {
		return err
	}

	return writeTemplate(lines, entry.date.Format("01-02"))
}

func getInfo(date time.Time) (entryInfo, error) {
	formattedDate := date.Format("01-02")
	path := "journal/" + formattedDate + ".md"
	prevFile, err := os.Stat(path)
	if err != nil {
		return entryInfo{}, err
	}

	if !prevFile.Mode().IsRegular() {
		return entryInfo{}, fmt.Errorf("%s is not a regular file", path)
	}

	source, err := os.Open(path)
	if err != nil {
		return entryInfo{}, err
	}
	defer source.Close()

	ei := entryInfo{date: date}
	//var lines []string
	var lineCount int

	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if strings.Contains(line, "* End of day total:") {
			b := strings.Replace(line, "* End of day total: **", "", 1)
			ei.budget = strings.Replace(b, "**", "", 1)
		}

		if strings.Contains(line, "* **Total Distance:**") {
			mileage, _ := strconv.Atoi(strings.Split(line, " ")[3])
			ei.mileage = mileage
		}

		if strings.Contains(line, "* **Total Budget Spent:**") {
			ei.previousTotalSpend = strings.Split(line, " ")[4]
		}
	}

	if err := scanner.Err(); err != nil {
		return entryInfo{}, err
	}
	return ei, nil
}

type replacement struct {
	find, replace string
}

func applyTemplate(data entryInfo) ([]string, error) {
	template := "journal/template.md"

	replacements := []replacement{
		{find: "mm/dd", replace: data.date.Format("01/02")},
		{find: "`mm-dd`", replace: data.date.Format("01-02")},
		{find: "mm-dd", replace: data.date.Format("01-02")},
		{find: "`Date`", replace: data.date.Format("Monday, January 02") + ", 2016"},
		{find: "`StartLong`", replace: data.startLong},
		{find: "`EndLong`", replace: data.endLong},
		{find: "`PreviousSpend`", replace: data.previousSpend},
		{find: "`Mileage`", replace: strconv.Itoa(data.mileage)},
		{find: "`TotalMileage`", replace: strconv.Itoa(data.previousMileage + data.mileage)},
		{find: "TotalSpend", replace: data.previousTotalSpend + " + Expenses"},
	}
	if data.startShort == data.endShort {
		replacements = append(replacements, replacement{find: "`Start` to `End`", replace: data.startShort})
	}
	replacements = append(replacements, replacement{find: "`Start`", replace: data.startShort})
	replacements = append(replacements, replacement{find: "`End`", replace: data.endShort})

	lines, err := apply(template, replacements)
	if err != nil {
		return nil, err
	}

	return lines, nil
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

func writeTemplate(lines []string, formattedDate string) error {
	destination, err := os.Create("journal/" + formattedDate + ".md")
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
