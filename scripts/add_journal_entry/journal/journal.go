package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
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

		previous, err = j.PreviousEntry(e)
		if err != nil {
			log.Printf("WARNING: Unable to find previous entry for %s", e.DateString)
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

func (j *Journal) PreviousEntry(entry Entry) (Entry, error) {
	p := entry.Date.AddDate(0, 0, -1)
	for _, e := range j.Entries {
		if e.Date == p {
			return e, nil
		}
	}
	return Entry{}, errors.New("unable to find a previous entry")
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

func (j *Journal) Write(e Entry, lines []string) error {
	destination, err := os.Create(e.EntryFilePath())
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

func (j *Journal) WriteIndex(e Entry) error {
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
