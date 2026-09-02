package repository

import (
	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
)

// Movies reads downloaded movies.
type Movies struct{}

// NewMovies creates the movies repository.
func NewMovies() *Movies { return &Movies{} }

// List returns every stored movie sorted by title.
func (r *Movies) List() ([]*models.Movie, error) {
	return common.GetMovies()
}

// Delete removes a movie record; media files and torrents stay.
func (r *Movies) Delete(id string) (*models.Movie, error) {
	return common.RemoveMovie(id)
}
