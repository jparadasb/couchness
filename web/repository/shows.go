// Package repository wraps Couchness storage access for the web UI.
package repository

import (
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// Shows reads and writes tracked shows.
type Shows struct{}

// NewShows creates the shows repository.
func NewShows() *Shows { return &Shows{} }

// List returns every show with its episode count (episodes stripped).
func (r *Shows) List() ([]*models.Show, error) {
	return common.GetShows()
}

// Get returns one show including its episodes, newest first.
func (r *Shows) Get(id string) (*models.Show, error) {
	value, err := common.GetShow(id, true)
	if err != nil {
		return nil, err
	}
	show := value.(*models.Show)
	storage.SortEpisodes(show.Episodes)
	return show, nil
}

// Identify links a show to an IMDb entry.
func (r *Shows) Identify(id, title, externalID string) (*models.Show, error) {
	return common.IdentifyShow(id, title, externalID)
}

// SaveConfiguration replaces the configuration of a show and persists it.
func (r *Shows) SaveConfiguration(id string, conf *models.ShowConf) (*models.Show, error) {
	show := &models.Show{}
	if err := storage.Db.Driver.Read(storage.Db.Collections.Shows, id, show); err != nil {
		return nil, err
	}
	show.Configuration = conf
	if _, err := storage.NewShowStorage(show).Save(); err != nil {
		return nil, err
	}
	return show, nil
}

// Disable sets a show to manual follow mode.
func (r *Shows) Disable(id string) error {
	_, err := common.DisableShow(id)
	return err
}

// Delete removes a show from tracking; media files and torrents stay.
func (r *Shows) Delete(id string) (*models.Show, error) {
	return common.RemoveShow(id)
}
