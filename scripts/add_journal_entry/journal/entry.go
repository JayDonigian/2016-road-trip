package journal

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

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

func (e *Entry) EntryFilePath() string {
	return fmt.Sprintf("journal/%s.png", e.Date.Format("01-02"))
}

func (e *Entry) DailyMapFilePath() string {
	return fmt.Sprintf("journal/maps/%s.png", e.Date.Format("01-02"))
}

func (e *Entry) TotalMapFilePath() string {
	return fmt.Sprintf("journal/maps/totals/%s.png", e.Date.Format("01-02"))

}

func (e *Entry) HasEntryFile() bool {
	_, err := os.Stat(e.EntryFilePath())
	if err != nil {
		return false
	}
	return true
}

func (e *Entry) HasDailyMapFile() bool {
	_, err := os.Stat(e.DailyMapFilePath())
	if err != nil {
		return false
	}
	return true
}

func (e *Entry) HasTotalMapFile() bool {
	_, err := os.Stat(e.TotalMapFilePath())
	if err != nil {
		return false
	}
	return true
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
	return fmt.Sprintf("### %s - %s  [%s](journal/%s.md) %s", date, e.Start.Emoji, e.Title(), date, e.End.Emoji)
}

func (e *Entry) Title() string {
	if e.Start.Short == e.End.Short {
		return e.Start.Short
	}
	return fmt.Sprintf("%s to %s", e.Start.Short, e.End.Short)
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
