package journal

import (
	"fmt"
	"os"
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

	Mileage             int `json:"mileage"`
	RunningMileageTotal int `json:"running_mileage_total"`

	BudgetStart         float64 `json:"budget_start"`
	DailyExpenseTotal   float64 `json:"daily_expense_total"`
	BudgetEnd           float64 `json:"budget_end"`
	RunningExpenseTotal float64 `json:"running_expense_total"`

	Start         Location  `json:"start"`
	End           Location  `json:"end"`
	DailyExpenses []Expense `json:"expenses"`
}

func (e *Entry) EntryFilePath() string {
	return fmt.Sprintf("journal/entries/%s.md", e.Date.Format("01-02"))
}

func (e *Entry) DailyMapFilePath() string {
	return fmt.Sprintf("journal/maps/day/%s.png", e.Date.Format("01-02"))
}

func (e *Entry) TotalMapFilePath() string {
	return fmt.Sprintf("journal/maps/total/%s-total.png", e.Date.Format("01-02"))

}

func (e *Entry) RelativeDailyMapFilePath() string {
	return fmt.Sprintf("../maps/day/%s.png", e.Date.Format("01-02"))
}

func (e *Entry) RelativeTotalMapFilePath() string {
	return fmt.Sprintf("../maps/total/%s-total.png", e.Date.Format("01-02"))

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

func (e *Entry) addHistory(p *Entry) {
	var prevMileage int
	var prevEnd, prevExpense float64

	if p == nil {
		e.BudgetStart = 60
	} else {
		prevMileage = p.RunningMileageTotal
		prevEnd = p.BudgetEnd
		prevExpense = p.RunningExpenseTotal
	}

	e.BudgetStart = prevEnd + 60.00
	e.BudgetEnd = prevEnd + 60.00 - e.DailyExpenseTotal
	e.RunningMileageTotal = e.Mileage + prevMileage
	e.RunningExpenseTotal = e.DailyExpenseTotal + prevExpense
}

func (e *Entry) Index() string {
	date := e.Date.Format("01-02")
	return fmt.Sprintf("### %s - %s  [%s](%s) %s", date, e.Start.Emoji, e.Title(), e.EntryFilePath(), e.End.Emoji)
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
	format := "#### [<< Previous Post](%s.md) | [Index](../../README.md) | [Next Post >>](%s.md)\n"
	return []string{fmt.Sprintf(format, e.PrevName(), e.NextName())}
}

func (e *Entry) TripInfo() []string {
	return []string{
		"## Today's Trip\n",
		fmt.Sprintf("**Date:** %s\n", e.Date.Format("Monday, January 02, 2006")),
		fmt.Sprintf("**Starting Point:** %s\n", e.Start.Long),
		fmt.Sprintf("**Destination:** %s\n", e.End.Long),
		fmt.Sprintf("**Distance:** %d miles\n", e.Mileage),
		fmt.Sprintf("**Photos:** [%s Photos](https://jay-d.me/2016RT-%s)\n", e.Date.Format("01/02"), e.Name),
		fmt.Sprintf("![map from %s](%s \"day map\")\n", e.Title(), e.RelativeDailyMapFilePath()),
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
		fmt.Sprintf("* $%.2f expenses", e.DailyExpenseTotal),
	}
	for _, ex := range e.DailyExpenses {
		lines = append(lines, fmt.Sprintf("  * $%.2f\t%s", ex.Cost, ex.Item))
	}
	lines = append(lines, fmt.Sprintf("* End of day total: **$%.2f**\n", e.BudgetEnd))
	return lines
}

func (e *Entry) TotalTripStats() []string {
	return []string{
		"## Trip Statistics\n",
		fmt.Sprintf("* **Total Distance:** %d miles", e.RunningMileageTotal),
		fmt.Sprintf("* **Total Budget Spent:** $%.2f", e.RunningExpenseTotal),
		"* **U.S. States**",
		"  * New Hampshire",
		"  * Maine",
		"* **Canadian Provinces**",
		"  * Nova Scotia",
		"* **National Parks**",
		"  * Acadia\n",
		fmt.Sprintf("![total trip from Fremont to %s](%s \"total trip map\")\n", e.End.Short, e.RelativeTotalMapFilePath()),
	}
}

func (e *Entry) Write() []string {
	sections := [][]string{
		e.TitleSection(),
		e.PrevNextLinks(),
		e.TripInfo(),
		e.EmojiStory(),
		e.JournalEntry(),
		e.Budget(),
		e.TotalTripStats(),
		e.PrevNextLinks(),
	}

	var lines []string
	for _, s := range sections {
		for _, l := range s {
			lines = append(lines, l)
		}
	}

	return lines
}
