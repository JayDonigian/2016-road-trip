package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type Journal struct {
	indexPath    string
	MileageTotal int      `json:"mileage_total"`
	ExpenseTotal float64  `json:"expense_total"`
	Entries      []*Entry `json:"entries"`
}

func New(jsonPath string) (*Journal, error) {
	j := &Journal{}
	err := j.unmarshal(jsonPath)
	if err != nil {
		return nil, err
	}

	j.indexPath = "README.md"

	var t time.Time
	for i, e := range j.Entries {
		t, err = time.Parse("01-02", e.Name)
		if err != nil {
			return nil, err
		}
		j.Entries[i].Date = time.Date(2016, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())

		if e.DailyExpenses == 0 {
			for _, expense := range e.Expenses {
				e.DailyExpenses += expense.Cost
			}
		}

		e.addHistory(j)
	}

	return j, nil
}

func (j *Journal) unmarshal(jsonPath string) error {
	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		return err
	}
	defer func() { _ = jsonFile.Close() }()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &j)
	if err != nil {
		return err
	}
	return nil
}

func (j *Journal) previousEntry(entry *Entry) (*Entry, error) {
	p := entry.Date.AddDate(0, 0, -1)
	for _, e := range j.Entries {
		if e.Date == p {
			return e, nil
		}
	}
	return nil, errors.New("unable to find a previous entry")
}

func (j *Journal) MissingEntries() []*Entry {
	var missing []*Entry
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

func (j *Journal) Write(e *Entry) error {
	destination, err := os.Create(e.EntryFilePath())
	if err != nil {
		return err
	}
	defer func() { _ = destination.Close() }()

	writer := bufio.NewWriter(destination)
	defer func() { _ = writer.Flush() }()

	lines := e.Write()
	for _, line := range j.TotalTripStats(e) {
		lines = append(lines, line)
	}
	for _, line := range e.PrevNextLinks() {
		lines = append(lines, line)
	}
	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}

	return nil
}

func (j *Journal) WriteIndex(e *Entry) error {
	file, err := os.OpenFile("README.md", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Println(err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), e.Name) {
			return nil
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	_, err = file.WriteString(fmt.Sprintf("%s\n", e.Index()))
	if err != nil {
		return err
	}

	return nil
}

func (j *Journal) Save() error {
	jsonString, _ := json.MarshalIndent(j, "", "    ")
	err := os.WriteFile("journal/journal.json", jsonString, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (j *Journal) TotalTripStats(e *Entry) []string {
	return []string{
		"## Trip Statistics\n",
		fmt.Sprintf("* **Total Distance:** %d miles", j.MileageTotal),
		fmt.Sprintf("* **Total Budget Spent:** $%.2f", j.ExpenseTotal),
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
