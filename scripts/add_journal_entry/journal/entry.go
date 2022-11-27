package journal

import (
	"bufio"
	"fmt"
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
	return fmt.Sprintf("journal/%s.md", e.Date.Format("01-02"))
}

func (e *Entry) DailyMapFilePath() string {
	return fmt.Sprintf("journal/maps/%s.png", e.Date.Format("01-02"))
}

func (e *Entry) TotalMapFilePath() string {
	return fmt.Sprintf("journal/maps/totals/%s-total.png", e.Date.Format("01-02"))

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

func (e *Entry) ApplyToTemplate(template string) ([]string, error) {
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
