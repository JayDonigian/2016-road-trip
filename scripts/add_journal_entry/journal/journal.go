package journal

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"
)

type Journal struct {
	Entries []Entry `json:"entries"`
}

func Unmarshal(jsonPath string) (*Journal, error) {
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = jsonFile.Close() }()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	j := &Journal{}
	err = json.Unmarshal(bytes, &j)
	if err != nil {
		return nil, err
	}

	var previous Entry
	var t time.Time
	for i, e := range j.Entries {
		t, err = time.Parse("01-02", e.DateString)
		if err != nil {
			return nil, err
		}
		j.Entries[i].Date = time.Date(2016, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())

		previous, err = e.PreviousEntry()
		if err != nil {
			return nil, err
		}

		for _, expense := range e.DailyExpenses {
			e.DailyExpenseTotal += expense.Cost
		}

		e.BudgetStart = previous.BudgetEnd + 60
		e.BudgetEnd = previous.BudgetEnd + 60 - e.DailyExpenseTotal
		e.RunningExpenseTotal = e.DailyExpenseTotal + previous.RunningExpenseTotal
		e.RunningMileageTotal = e.Mileage + previous.RunningMileageTotal
	}

	return j, nil
}

func (j *Journal) MissingEntries() []Entry {
	var missing []Entry
	for _, e := range j.Entries {
		if !e.HasDailyMapFile() {
			log.Printf("WARNING: map for %s does not exist\n", e.DailyMapFilePath())
		}
		if !e.HasTotalMapFile() {
			log.Printf("WARNING: total map for %s does not exist\n", e.TotalMapFilePath())
		}
		if !e.HasEntryFile() {
			missing = append(missing, e)
		}
	}
	return missing
}
