package services

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	viewmodels "github.com/highercomve/couchness/web/models"
	"github.com/highercomve/couchness/web/repository"
)

// Movies exposes downloaded movies, OMDb search, torrent lookup and queueing.
type Movies struct {
	repo   *repository.Movies
	config *repository.Config
}

// NewMovies creates the movies service.
func NewMovies(repo *repository.Movies, config *repository.Config) *Movies {
	return &Movies{repo: repo, config: config}
}

// List returns every stored movie.
func (m *Movies) List() ([]*models.Movie, error) {
	return m.repo.List()
}

// Delete removes a movie record; media files and torrents stay.
func (m *Movies) Delete(id string) (*models.Movie, error) {
	return m.repo.Delete(id)
}

// Search returns OMDb "movie" results. Empty query returns (nil, nil).
func (m *Movies) Search(query string) ([]common.OmdbResults, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	response, err := common.SearchShowInfo(query, "movie")
	if err != nil {
		return nil, err
	}
	return response.Search, nil
}

// Torrents builds the movie then calls common.SearchMovieTorrents.
func (m *Movies) Torrents(request viewmodels.MovieRequest) (*models.Movie, models.Episodes, error) {
	movie, err := m.BuildMovie(request)
	if err != nil {
		return nil, nil, err
	}
	torrents, err := common.SearchMovieTorrents(movie)
	if err != nil {
		return movie, nil, err
	}
	return movie, torrents, nil
}

// Download builds the movie, converts the torrent request and calls common.DownloadMovie.
func (m *Movies) Download(request viewmodels.MovieRequest, torrent viewmodels.TorrentRequest) (*models.Movie, error) {
	movie, err := m.BuildMovie(request)
	if err != nil {
		return nil, err
	}
	if torrent.Magnet == "" {
		return nil, errors.New("a magnet link is required")
	}
	torrentInfo := &models.TorrentInfo{
		Title:      torrent.Title,
		MagnetURL:  torrent.Magnet,
		Size:       torrent.Size,
		Seeds:      torrent.Seeds,
		Resolution: torrent.Resolution,
		Quality:    torrent.Quality,
		Codec:      torrent.Codec,
	}
	if err := common.DownloadMovie(movie, torrentInfo); err != nil {
		return movie, err
	}
	return movie, nil
}

// BuildMovie validates and converts a request into a models.Movie ready for search/download.
func (m *Movies) BuildMovie(request viewmodels.MovieRequest) (*models.Movie, error) {
	imdbID := strings.TrimSpace(request.ImdbID)
	if imdbID == "" {
		return nil, errors.New("an IMDb ID is required")
	}

	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = imdbID
	}

	key := strings.TrimSpace(request.Key)
	if key == "" {
		key = slug.Make(strings.TrimSpace(title + " " + request.Year))
	}

	folder := strings.TrimSpace(request.Folder)
	if folder == "" {
		folder = m.config.MoviesDir()
	}
	if folder == "" {
		return nil, errors.New("no movies directory configured, set COUCHNESS_MOVIES_DIR or pass a folder")
	}

	directory, err := filepath.Abs(folder)
	if err != nil {
		return nil, err
	}
	directory += "/"

	return &models.Movie{
		Show: models.Show{
			ID:            key,
			Title:         title,
			ExternalID:    imdbID,
			Directory:     directory,
			Configuration: &models.ShowConf{Services: common.DefaultMovieServices},
		},
	}, nil
}
