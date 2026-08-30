package common

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// RemoveShow removes a show from Couchness tracking only.
// Media files and Transmission torrents remain untouched.
func RemoveShow(showID string) (*models.Show, error) {
	show := &models.Show{}
	if err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show); err != nil {
		return nil, err
	}
	if err := storage.IgnoreShow(show.ID, show.Directory); err != nil {
		return nil, err
	}
	if err := storage.DeleteShow(showID); err != nil {
		_ = storage.UnignoreShow(show.ID, show.Directory)
		return nil, err
	}
	return show, nil
}
