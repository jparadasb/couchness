package common

import (
	"fmt"
	"path/filepath"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// UpdateAll scans every configured show directory and updates followed shows.
// Progress receives human-readable status messages when non-nil.
func UpdateAll(progress func(string)) error {
	report := func(message string) {
		if progress != nil {
			progress(message)
		}
	}

	for _, directory := range storage.AppConfiguration.ShowsDirs {
		folderPath, err := filepath.Abs(directory)
		if err != nil {
			return err
		}

		shows, err := Scan(folderPath+"/", "", false, false)
		if err != nil {
			return err
		}

		for _, show := range shows {
			if show.Configuration.FollowType == models.FollowTypeManual {
				report(fmt.Sprintf("Skipping %s: manual mode", show.Title))
				continue
			}

			report(fmt.Sprintf("Searching for episodes of %s", show.Title))
			if err := Download(show.ID); err != nil {
				report(fmt.Sprintf("Could not update %s: %s", show.Title, err))
			}
		}
	}

	return nil
}
