package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/web/repository"
	viewmodels "github.com/highercomve/couchness/web/models"
)

// Shows exposes tracked shows and IMDb identification.
type Shows struct {
	repo *repository.Shows
}

// NewShows creates the shows service.
func NewShows(repo *repository.Shows) *Shows {
	return &Shows{repo: repo}
}

// List returns every tracked show.
func (s *Shows) List() ([]*models.Show, error) {
	return s.repo.List()
}

// Get returns one show including its episodes.
func (s *Shows) Get(id string) (*models.Show, error) {
	return s.repo.Get(id)
}

// SearchIMDb returns OMDb "series" results. Empty/whitespace query returns (nil, nil) without calling OMDb.
func (s *Shows) SearchIMDb(query string) ([]common.OmdbResults, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	response, err := common.SearchShowInfo(query, "series")
	if err != nil {
		return nil, err
	}
	return response.Search, nil
}

// Identify validates externalID != "" then repo.Identify.
func (s *Shows) Identify(id, title, externalID string) (*models.Show, error) {
	if strings.TrimSpace(externalID) == "" {
		return nil, errors.New("an IMDb ID is required")
	}
	return s.repo.Identify(id, title, externalID)
}

// Disable sets a show to manual follow mode.
func (s *Shows) Disable(id string) error {
	return s.repo.Disable(id)
}

// Delete removes a show from tracking; media files and torrents stay.
func (s *Shows) Delete(id string) (*models.Show, error) {
	return s.repo.Delete(id)
}

// allowedServices lists the torrent services configurable from the web UI.
var allowedServices = map[string]bool{"eztv": true, "tpb": true}

// UpdateConfiguration validates a config request and persists it on the show.
func (s *Shows) UpdateConfiguration(id string, request viewmodels.ShowConfigRequest) (*models.Show, error) {
	followType := strings.TrimSpace(request.FollowType)
	switch followType {
	case models.FollowTypeLatest, models.FollowTypeSince, models.FollowTypeAll, models.FollowTypeManual:
	default:
		return nil, fmt.Errorf("invalid follow type %q, must be one of latest, since, all, manual", followType)
	}
	if request.Since < 0 {
		return nil, errors.New("since must be a positive season number")
	}
	if followType == models.FollowTypeSince && request.Since < 1 {
		return nil, errors.New("since requires a season number of 1 or higher")
	}

	services := make([]string, 0, len(request.Services))
	for _, service := range request.Services {
		service = strings.ToLower(strings.TrimSpace(service))
		if service == "" {
			continue
		}
		if !allowedServices[service] {
			return nil, fmt.Errorf("invalid service %q, must be one of eztv, tpb", service)
		}
		services = append(services, service)
	}

	conf := &models.ShowConf{
		FollowType: followType,
		Since:      request.Since,
		Resolution: strings.ToLower(strings.TrimSpace(request.Resolution)),
		Quality:    strings.ToLower(strings.TrimSpace(request.Quality)),
		Codec:      strings.ToLower(strings.TrimSpace(request.Codec)),
		Services:   services,
	}
	if len(conf.Services) == 0 {
		conf.Service = "eztv"
	}

	return s.repo.SaveConfiguration(id, conf)
}
