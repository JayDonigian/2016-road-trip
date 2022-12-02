package journal

import (
	"fmt"
	"time"
)

type Location struct {
	Emoji string `json:"emoji"`
	Short string `json:"short"`
	Long  string `json:"long"`
}

type Expense struct {
	Item string  `json:"item"`
	Cost float64 `json:"cost"`
}

type Entry struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`

	Mileage int `json:"mileage"`

	BudgetStart   float64 `json:"budget_start"`
	DailyExpenses float64 `json:"daily_expenses"`
	BudgetEnd     float64 `json:"budget_end"`

	Start    Location  `json:"start"`
	End      Location  `json:"end"`
	Expenses []Expense `json:"expenses"`
}

func (e *Entry) addHistory(j *Journal) {
	var prevMileage int
	var prevEnd, prevExpense float64

	p, err := j.previousEntry(e)
	if err != nil {
		e.BudgetStart = 60
	} else {
		prevMileage = j.MileageTotal
		prevEnd = p.BudgetEnd
		prevExpense = j.ExpenseTotal
	}

	e.BudgetStart = prevEnd + 60.00
	e.BudgetEnd = prevEnd + 60.00 - e.DailyExpenses
	j.MileageTotal = e.Mileage + prevMileage
	j.ExpenseTotal = e.DailyExpenses + prevExpense
}

func (e *Entry) Index() string {
	date := e.Date.Format("01-02")
	return fmt.Sprintf("### %s - %s  [%s](https://jay-d.me/2016RT-%s) %s", date, e.Start.Emoji, e.Title(), e.Name, e.End.Emoji)
}

func (e *Entry) PrevName() string {
	return e.Date.AddDate(0, 0, -1).Format("01-02")
}

func (e *Entry) NextName() string {
	return e.Date.AddDate(0, 0, 1).Format("01-02")
}

func (e *Entry) Title() string {
	if e.Start.Short == e.End.Short {
		return fmt.Sprintf("%s", e.Start.Short)
	}
	return fmt.Sprintf("%s to %s", e.Start.Short, e.End.Short)
}

func (e *Entry) RelativePathToFile(ft fileType) string {
	return fmt.Sprintf(ft.formatPathRelativeToEntry(), e.Name)
}

func (e *Entry) TitleWithEmoji() string {
	if e.Start.Short == e.End.Short {
		return fmt.Sprintf("%s  %s %s", e.Start.Emoji, e.Start.Short, e.Start.Emoji)
	}
	return fmt.Sprintf("%s  %s to %s %s", e.Start.Emoji, e.Start.Short, e.End.Short, e.End.Emoji)
}

func (e *Entry) TitleSection() []string {
	return []string{fmt.Sprintf("# %s\n", e.TitleWithEmoji())}
}

func (e *Entry) PrevNextLinks() []string {
	format := "#### [<< Previous Post](https://jay-d.me/2016RT-%s) | [Index](../../README.md) | [Next Post >>](https://jay-d.me/2016RT-%s)\n"
	return []string{fmt.Sprintf(format, e.PrevName(), e.NextName())}
}

func (e *Entry) TripInfo() []string {
	return []string{
		"## Today's Trip\n",
		fmt.Sprintf("**Date:** %s\n", e.Date.Format("Monday, January 02, 2006")),
		fmt.Sprintf("**Starting Point:** %s\n", e.Start.Long),
		fmt.Sprintf("**Destination:** %s\n", e.End.Long),
		fmt.Sprintf("**Distance:** %d miles\n", e.Mileage),
		fmt.Sprintf("**Photos:** [%s Photos](https://jay-d.me/2016RT-%s-photos)\n", e.Date.Format("01/02"), e.Name),
		fmt.Sprintf("![map from %s](%s \"day map\")\n", e.Title(), e.RelativePathToFile(dayMap)),
	}
}

func (e *Entry) EmojiStory() []string {
	return []string{"##  `EmojiStory`\n"}
}

func (e *Entry) JournalEntry() []string {
	return []string{
		"## Journal Entry\n",
		"* `Journal Entry`\n",
	}
}

func (e *Entry) Budget() []string {
	lines := []string{
		"## The Budget\n",
		fmt.Sprintf("* $%.2f from previous day", e.BudgetStart-60),
		"* $60.00 daily addition",
		fmt.Sprintf("* $%.2f expenses", e.DailyExpenses),
	}
	for _, ex := range e.Expenses {
		lines = append(lines, fmt.Sprintf("  * $%.2f\t%s", ex.Cost, ex.Item))
	}
	lines = append(lines, fmt.Sprintf("* End of day total: **$%.2f**\n", e.BudgetEnd))
	return lines
}

func (e *Entry) Write() []string {
	sections := [][]string{
		e.TitleSection(),
		e.PrevNextLinks(),
		e.TripInfo(),
		e.EmojiStory(),
		e.JournalEntry(),
		e.Budget(),
	}

	var lines []string
	for _, s := range sections {
		for _, l := range s {
			lines = append(lines, l)
		}
	}

	return lines
}
