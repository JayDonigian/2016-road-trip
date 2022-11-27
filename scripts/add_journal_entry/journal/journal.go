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

	entryInfo := &Journal{}
	err = json.Unmarshal(bytes, &entryInfo)
	if err != nil {
		return nil, err
	}

	return entryInfo, nil
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
