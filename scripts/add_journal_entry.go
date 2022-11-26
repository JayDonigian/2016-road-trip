package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Date                time.Time
	RunningMileageTotal int
	RunningExpenseTotal float64
	DailyExpenseTotal   float64
	BudgetStart         float64
	BudgetEnd           float64

	Mileage    int    `json:"mileage"`
	DateString string `json:"date"`
	Start      struct {
		Emoji string `json:"emoji"`
		Short string `json:"short"`
		Long  string `json:"long"`
	} `json:"start"`
	End struct {
		Emoji string `json:"emoji"`
		Short string `json:"short"`
		Long  string `json:"long"`
	} `json:"end"`
	DailyExpenses []struct {
		Item string  `json:"item"`
		Cost float64 `json:"cost"`
	} `json:"expenses"`
}

func main() {
	err := parseArgs()
	if err != nil {
		log.Fatalf("ERROR: while parsing arguments - %s", err.Error())
	}

	info, err := unmarshalInfo()
	if err != nil {
		log.Fatalf("ERROR: while unmarshaling JSON file - %s", err.Error())
	}

	t, err := time.Parse("01-02", info.DateString)
	if err != nil {
		log.Fatalf("ERROR: while parsing date field - %s", err.Error())
	}
	info.Date = time.Date(2016, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())

	err = checkFiles(info.Date.Format("01-02"))
	if err != nil {
		log.Fatalf("ERROR: while checking files - %s", err.Error())
	}

	err = createFromTemplate(info)
	if err != nil {
		log.Fatalf("ERROR: while copying the template file - %s", err)
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

func unmarshalInfo() (Info, error) {
	jsonFile, err := os.Open("journal/templates/new_entry.json")
	if err != nil {
		return Info{}, nil
	}
	defer jsonFile.Close()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return Info{}, err
	}

	newEntry := Info{}
	err = json.Unmarshal(bytes, &newEntry)
	if err != nil {
		return Info{}, err
	}

	return newEntry, nil
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

func createFromTemplate(entry Info) error {
	previous, err := getInfo(entry.Date.AddDate(0, 0, -1))
	if err != nil {
		return err
	}

	for _, expense := range entry.DailyExpenses {
		entry.DailyExpenseTotal += expense.Cost
	}

	entry.BudgetStart = previous.BudgetEnd + 60
	entry.BudgetEnd = previous.BudgetEnd + 60 - entry.DailyExpenseTotal
	entry.RunningExpenseTotal = entry.DailyExpenseTotal + previous.RunningExpenseTotal
	entry.RunningMileageTotal = entry.Mileage + previous.RunningMileageTotal

	lines, err := applyTemplate(entry)
	if err != nil {
		return err
	}

	return writeTemplate(lines, entry.Date.Format("01-02"))
}

func getInfo(date time.Time) (Info, error) {
	formattedDate := date.Format("01-02")
	path := "journal/" + formattedDate + ".md"
	prevFile, err := os.Stat(path)
	if err != nil {
		return Info{}, err
	}

	if !prevFile.Mode().IsRegular() {
		return Info{}, fmt.Errorf("%s is not a regular file", path)
	}

	source, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer source.Close()

	ei := Info{Date: date}
	var lineCount int

	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// This mess can be cleaned up by adding logic to unmarshal previous entries
		if strings.Contains(line, "* End of day total:") {
			totalString := strings.Replace(line, "* End of day total: **", "", 1)
			totalString = strings.Replace(totalString, "$", "", 1)
			totalString = strings.Replace(totalString, "**", "", 1)
			dailyExpenseTotal, err := strconv.ParseFloat(totalString, 64)
			if err != nil {
				return Info{}, err
			}
			ei.BudgetEnd = dailyExpenseTotal
		}

		if strings.Contains(line, "* **Total Budget Spent:**") {
			totalString := strings.Split(line, " ")[4]
			totalString = strings.Replace(totalString, "$", "", 1)
			runningExpenseTotal, err := strconv.ParseFloat(totalString, 64)
			if err != nil {
				return Info{}, err
			}
			ei.RunningExpenseTotal = runningExpenseTotal
		}

		if strings.Contains(line, "* **Total Distance:**") {
			mileage, _ := strconv.Atoi(strings.Split(line, " ")[3])
			ei.RunningMileageTotal = mileage
		}
	}

	if err := scanner.Err(); err != nil {
		return Info{}, err
	}
	return ei, nil
}

type replacement struct {
	find, replace string
}

func applyTemplate(data Info) ([]string, error) {
	template := "journal/templates/template.md"

	prevDay := data.Date.AddDate(0, 0, -1)
	nextDay := data.Date.AddDate(0, 0, 1)

	var expensesReplacements string
	for _, exp := range data.DailyExpenses {
		expensesReplacements += fmt.Sprintf("  * %s - $%.2f\n", exp.Item, exp.Cost)
	}

	replacements := []replacement{
		{find: "`StartEmoji`", replace: data.Start.Emoji},
		{find: "`EndEmoji`", replace: data.End.Emoji},
		{find: "`Previous`", replace: prevDay.Format("01-02")},
		{find: "`Next`", replace: nextDay.Format("01-02")},
		{find: "mm/dd", replace: data.Date.Format("01/02")},
		{find: "`mm-dd`", replace: data.Date.Format("01-02")},
		{find: "mm-dd", replace: data.Date.Format("01-02")},
		{find: "`Date`", replace: data.Date.Format("Monday, January 02") + ", 2016"},
		{find: "`StartLong`", replace: data.Start.Long},
		{find: "`EndLong`", replace: data.End.Long},
		{find: "`PreviousSpend`", replace: fmt.Sprintf("%.2f", data.BudgetStart-60)},
		{find: "`Mileage`", replace: strconv.Itoa(data.Mileage)},
		{find: "`Expenses`", replace: fmt.Sprintf("%.2f", data.DailyExpenseTotal)},
		{find: "`ExpenseTotal`", replace: fmt.Sprintf("%.2f", data.BudgetEnd)},
		{find: "`TotalSpend`", replace: fmt.Sprintf("%.2f", data.RunningExpenseTotal)},
		{find: "`TotalMileage`", replace: strconv.Itoa(data.RunningMileageTotal)},
		{find: "`EXPENSES`", replace: expensesReplacements},
	}
	if data.Start.Short == data.End.Short {
		replacements = append(replacements, replacement{find: "`Start` to `End`", replace: data.Start.Short})
	}
	replacements = append(replacements, replacement{find: "`Start`", replace: data.Start.Short})
	replacements = append(replacements, replacement{find: "`End`", replace: data.End.Short})

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
