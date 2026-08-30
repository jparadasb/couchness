package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/models"
)

// IgnoreShow prevents a removed show from being recreated by a media scan.
func IgnoreShow(showID, directory string) error {
	record := &models.IgnoredShow{
		ID:           showID,
		DirectoryKey: slug.Make(filepath.Base(filepath.Clean(directory))),
	}
	return Db.Driver.Write(Db.Collections.IgnoredShows, showID, record)
}

// UnignoreShow allows an explicitly added show to be tracked again.
func UnignoreShow(showID, directory string) error {
	records, err := getIgnoredShows()
	if err != nil {
		return err
	}
	directoryKey := slug.Make(filepath.Base(filepath.Clean(directory)))
	for _, record := range records {
		if record.ID == showID || record.DirectoryKey == directoryKey {
			if err := Db.Driver.Delete(Db.Collections.IgnoredShows, record.ID); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// GetIgnoredShowKeys returns show IDs and directory keys excluded from scans.
func GetIgnoredShowKeys() (map[string]bool, error) {
	records, err := getIgnoredShows()
	if err != nil {
		return nil, err
	}
	keys := make(map[string]bool, len(records)*2)
	for _, record := range records {
		keys[record.ID] = true
		keys[record.DirectoryKey] = true
	}
	return keys, nil
}

func getIgnoredShows() ([]*models.IgnoredShow, error) {
	values, err := Db.Driver.ReadAll(Db.Collections.IgnoredShows)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.IgnoredShow{}, nil
		}
		return nil, err
	}
	records := make([]*models.IgnoredShow, 0, len(values))
	for _, value := range values {
		record := &models.IgnoredShow{}
		if err := json.Unmarshal([]byte(value), record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}
