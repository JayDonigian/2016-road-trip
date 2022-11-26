package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Location struct {
	Emoji string `json:"emoji"`
	Short string `json:"short"`
	Long  string `json:"long"`
}

type Entry struct {
	Date                time.Time
	RunningMileageTotal int
	RunningExpenseTotal float64
	DailyExpenseTotal   float64
	BudgetStart         float64
	BudgetEnd           float64

	Mileage       int      `json:"mileage"`
	DateString    string   `json:"date"`
	Start         Location `json:"start"`
	End           Location `json:"end"`
	DailyExpenses []struct {
		Item string  `json:"item"`
		Cost float64 `json:"cost"`
	} `json:"expenses"`
}

func (e *Entry) NewFromTemplate(template string) error {
	err := e.InfoFromPrevious()
	if err != nil {
		return err
	}

	lines, err := e.Apply(template)
	if err != nil {
		return err
	}

	err = e.Write(lines)
	if err != nil {
		return err
	}

	err = e.WriteIndex()
	if err != nil {
		return err
	}

	return nil
}

func (e *Entry) InfoFromPrevious() error {
	previous, err := e.PreviousEntry()
	if err != nil {
		return err
	}

	for _, expense := range e.DailyExpenses {
		e.DailyExpenseTotal += expense.Cost
	}

	e.BudgetStart = previous.BudgetEnd + 60
	e.BudgetEnd = previous.BudgetEnd + 60 - e.DailyExpenseTotal
	e.RunningExpenseTotal = e.DailyExpenseTotal + previous.RunningExpenseTotal
	e.RunningMileageTotal = e.Mileage + previous.RunningMileageTotal

	return nil
}

func (e *Entry) PreviousEntry() (Entry, error) {
	path := fmt.Sprintf("journal/%s.md", e.Date.AddDate(0, 0, -1).Format("01-02"))

	prevFile, err := os.Stat(path)
	if err != nil {
		return Entry{}, err
	}

	if !prevFile.Mode().IsRegular() {
		return Entry{}, fmt.Errorf("%s is not a regular file", path)
	}

	source, err := os.Open(path)
	if err != nil {
		return Entry{}, err
	}
	defer func() { _ = source.Close() }()

	ei := Entry{Date: e.Date}
	var lineCount int

	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// This mess can be cleaned up by adding logic to Unmarshal previous entries
		if strings.Contains(line, "* End of day total:") {
			totalString := strings.Replace(line, "* End of day total: **$", "", 1)
			totalString = strings.Replace(totalString, "**", "", 2)
			dailyExpenseTotal, err := strconv.ParseFloat(totalString, 64)
			if err != nil {
				return Entry{}, err
			}
			ei.BudgetEnd = dailyExpenseTotal
		}

		if strings.Contains(line, "* **Total Budget Spent:**") {
			totalString := strings.Split(line, " ")[4]
			totalString = strings.Replace(totalString, "$", "", 1)
			runningExpenseTotal, err := strconv.ParseFloat(totalString, 64)
			if err != nil {
				return Entry{}, err
			}
			ei.RunningExpenseTotal = runningExpenseTotal
		}

		if strings.Contains(line, "* **Total Distance:**") {
			mileage, _ := strconv.Atoi(strings.Split(line, " ")[3])
			ei.RunningMileageTotal = mileage
		}
	}

	if err := scanner.Err(); err != nil {
		return Entry{}, err
	}
	return ei, nil
}

func (e *Entry) Apply(template string) ([]string, error) {
	prevDay := e.Date.AddDate(0, 0, -1)
	nextDay := e.Date.AddDate(0, 0, 1)

	var expensesReplacements string
	for _, exp := range e.DailyExpenses {
		expensesReplacements += fmt.Sprintf("  * $%.2f - %s\n", exp.Cost, exp.Item)
	}

	replacements := []replacement{
		{find: "`StartEmoji`", replace: e.Start.Emoji},
		{find: "`EndEmoji`", replace: e.End.Emoji},
		{find: "`Previous`", replace: prevDay.Format("01-02")},
		{find: "`Next`", replace: nextDay.Format("01-02")},
		{find: "mm/dd", replace: e.Date.Format("01/02")},
		{find: "`mm-dd`", replace: e.Date.Format("01-02")},
		{find: "mm-dd", replace: e.Date.Format("01-02")},
		{find: "`Date`", replace: e.Date.Format("Monday, January 02") + ", 2016"},
		{find: "`StartLong`", replace: e.Start.Long},
		{find: "`EndLong`", replace: e.End.Long},
		{find: "`PreviousSpend`", replace: fmt.Sprintf("%.2f", e.BudgetStart-60)},
		{find: "`Mileage`", replace: strconv.Itoa(e.Mileage)},
		{find: "`Expenses`", replace: fmt.Sprintf("%.2f", e.DailyExpenseTotal)},
		{find: "`ExpenseTotal`", replace: fmt.Sprintf("%.2f", e.BudgetEnd)},
		{find: "`TotalSpend`", replace: fmt.Sprintf("%.2f", e.RunningExpenseTotal)},
		{find: "`TotalMileage`", replace: strconv.Itoa(e.RunningMileageTotal)},
		{find: "`EXPENSES`", replace: expensesReplacements},
	}
	if e.Start.Short == e.End.Short {
		replacements = append(replacements, replacement{find: "`Start` to `End`", replace: e.Start.Short})
	}
	replacements = append(replacements, replacement{find: "`Start`", replace: e.Start.Short})
	replacements = append(replacements, replacement{find: "`End`", replace: e.End.Short})

	lines, err := apply(template, replacements)
	if err != nil {
		return nil, err
	}

	return lines, nil
}

func (e *Entry) Write(lines []string) error {
	destination, err := os.Create("journal/" + e.Date.Format("01-02") + ".md")
	if err != nil {
		return err
	}
	defer func() { _ = destination.Close() }()

	writer := bufio.NewWriter(destination)
	defer func() { _ = writer.Flush() }()

	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}

	return nil
}

func (e *Entry) WriteIndex() error {
	f, err := os.OpenFile("README.md", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer func() { _ = f.Close() }()

	_, err = f.WriteString(fmt.Sprintf("%s\n", e.Index()))
	if err != nil {
		return err
	}

	return nil
}

func (e *Entry) Index() string {
	date := e.Date.Format("01-02")
	return fmt.Sprintf("### %s - %s  [%s](journal/%s.md) %s", date, e.Start.Emoji, e.Short(), date, e.End.Emoji)
}

func (e *Entry) Short() string {
	if e.Start.Short == e.End.Short {
		return e.Start.Short
	}
	return fmt.Sprintf("%s to %s", e.Start.Short, e.End.Short)
}

type Journal struct {
	Entries []Entry `json:"entries"`
}

func (j *Journal) AddYearToDates(y int) error {
	for i, entry := range j.Entries {
		t, err := time.Parse("01-02", entry.DateString)
		if err != nil {
			return err
		}
		j.Entries[i].Date = time.Date(y, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	}
	return nil
}

func (j *Journal) MissingEntries() []Entry {
	var missing []Entry
	for _, entry := range j.Entries {
		formattedDate := entry.Date.Format("01-02")
		if _, err := os.Stat("journal/maps/" + formattedDate + ".png"); errors.Is(err, os.ErrNotExist) {
			log.Printf("WARNING: map for %s does not exist\n", formattedDate)
		}

		if _, err := os.Stat("journal/maps/totals/" + formattedDate + "-total.png"); errors.Is(err, os.ErrNotExist) {
			log.Printf("WARNING: total map for %s does not exist\n", formattedDate)
		}

		if _, err := os.Stat("journal/" + formattedDate + ".md"); err != nil {
			missing = append(missing, entry)
		}
	}
	return missing
}

func Unmarshal(jsonPath string) (Journal, error) {
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return Journal{}, err
	}
	defer func() { _ = jsonFile.Close() }()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return Journal{}, err
	}

	entryInfo := Journal{}
	err = json.Unmarshal(bytes, &entryInfo)
	if err != nil {
		return Journal{}, err
	}

	return entryInfo, nil
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
	defer func() { _ = source.Close() }()

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
