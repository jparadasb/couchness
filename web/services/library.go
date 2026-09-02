package services

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/web/jobs"
	"github.com/highercomve/couchness/web/repository"
)

// Library orchestrates library-wide operations (scan, update-all) and per-show downloads.
type Library struct {
	config *repository.Config
}

// NewLibrary creates a new library service.
func NewLibrary(config *repository.Config) *Library {
	return &Library{config: config}
}

// Scan mirrors `couchness scan` (non-interactive): for every directory in config.ShowsDirs()
// it resolves filepath.Abs, logs "Scanning folder: <abs>", calls common.Scan(abs+"/", "", false, false),
// then logs "Show <Title> with <N> episodes in total" per show. Returns the first error.
// Logs "No show directories configured" and returns an error if ShowsDirs() is empty.
func (l *Library) Scan(log jobs.Logger) error {
	dirs := l.config.ShowsDirs()
	if len(dirs) == 0 {
		log("No show directories configured")
		return errors.New("no show directories configured")
	}

	for _, directory := range dirs {
		abs, err := filepath.Abs(directory)
		if err != nil {
			return err
		}
		log(fmt.Sprintf("Scanning folder: %s", abs))
		shows, err := common.Scan(abs+"/", "", false, false)
		if err != nil {
			return err
		}
		for _, show := range shows {
			log(fmt.Sprintf("Show %s with %d episodes in total", show.Title, len(show.Episodes)))
		}
	}
	return nil
}

// UpdateAll runs common.UpdateAll forwarding progress lines to log.
func (l *Library) UpdateAll(log jobs.Logger) error {
	return common.UpdateAll(log)
}

// DownloadShow logs "Searching latest episode of <id>", calls common.Download(id),
// logs "Show <id> is now in transmission download queue" on success. Returns the error.
func (l *Library) DownloadShow(id string, log jobs.Logger) error {
	log(fmt.Sprintf("Searching latest episode of %s", id))
	if err := common.Download(id); err != nil {
		return err
	}
	log(fmt.Sprintf("Show %s is now in transmission download queue", id))
	return nil
}

// UpdateShow logs "Rescanning and updating <id>", calls common.Update(id), logs "Done" on success.
func (l *Library) UpdateShow(id string, log jobs.Logger) error {
	log(fmt.Sprintf("Rescanning and updating %s", id))
	if err := common.Update(id); err != nil {
		return err
	}
	log("Done")
	return nil
}
