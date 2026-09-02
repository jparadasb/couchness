package common

import (
	"errors"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// IdentifyShow links a scanned show to its IMDb entry keeping the show ID stable.
func IdentifyShow(showID, title, externalID string) (*models.Show, error) {
	if externalID == "" {
		return nil, errors.New("an IMDb ID is required")
	}
	show := &models.Show{}
	if err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show); err != nil {
		return nil, err
	}
	if title != "" {
		show.Title = title
	}
	show.ExternalID = externalID
	if show.Configuration == nil {
		show.Configuration = &models.ShowConf{FollowType: models.FollowTypeLatest, Service: "eztv"}
	}
	if _, err := storage.NewShowStorage(show).Save(); err != nil {
		return nil, err
	}
	return show, nil
}
