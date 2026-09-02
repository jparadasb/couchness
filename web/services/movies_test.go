package services

import (
	"path/filepath"
	"testing"

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	viewmodels "github.com/highercomve/couchness/web/models"
	"github.com/highercomve/couchness/web/repository"
)

func TestBuildMovieKeyFromTitleAndYear(t *testing.T) {
	origAppConfig := storage.AppConfiguration
	defer func() { storage.AppConfiguration = origAppConfig }()
	storage.AppConfiguration = &models.AppConfiguration{MoviesDir: "/tmp/movies"}

	m := NewMovies(repository.NewMovies(), repository.NewConfig())
	movie, err := m.BuildMovie(viewmodels.MovieRequest{ImdbID: "tt0133093", Title: "The Matrix", Year: "1999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if movie.ID != "the-matrix-1999" {
		t.Errorf("ID = %q, want the-matrix-1999", movie.ID)
	}
	if movie.Show.Title != "The Matrix" {
		t.Errorf("Title = %q, want The Matrix", movie.Show.Title)
	}
	if movie.ExternalID != "tt0133093" {
		t.Errorf("ExternalID = %q, want tt0133093", movie.ExternalID)
	}
	if movie.Directory != "/tmp/movies/" {
		t.Errorf("Directory = %q, want /tmp/movies/", movie.Directory)
	}
	if len(movie.Configuration.Services) != len(common.DefaultMovieServices) {
		t.Fatalf("Services length = %d, want %d", len(movie.Configuration.Services), len(common.DefaultMovieServices))
	}
	for i, svc := range common.DefaultMovieServices {
		if movie.Configuration.Services[i] != svc {
			t.Errorf("Services[%d] = %q, want %q", i, movie.Configuration.Services[i], svc)
		}
	}
}

func TestBuildMovieKeepsExplicitKey(t *testing.T) {
	origAppConfig := storage.AppConfiguration
	defer func() { storage.AppConfiguration = origAppConfig }()
	storage.AppConfiguration = &models.AppConfiguration{MoviesDir: "/tmp/movies"}

	m := NewMovies(repository.NewMovies(), repository.NewConfig())
	movie, err := m.BuildMovie(viewmodels.MovieRequest{ImdbID: "tt0133093", Title: "The Matrix", Year: "1999", Key: "matrix"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if movie.ID != "matrix" {
		t.Errorf("ID = %q, want matrix", movie.ID)
	}
}

func TestBuildMovieRequiresImdbID(t *testing.T) {
	m := NewMovies(repository.NewMovies(), repository.NewConfig())
	_, err := m.BuildMovie(viewmodels.MovieRequest{Title: "The Matrix"})
	if err == nil {
		t.Fatal("expected error for empty ImdbID")
	}
}

func TestBuildMovieRequiresDirectory(t *testing.T) {
	origAppConfig := storage.AppConfiguration
	defer func() { storage.AppConfiguration = origAppConfig }()
	storage.AppConfiguration = &models.AppConfiguration{}

	m := NewMovies(repository.NewMovies(), repository.NewConfig())
	_, err := m.BuildMovie(viewmodels.MovieRequest{ImdbID: "tt0133093", Title: "The Matrix"})
	if err == nil {
		t.Fatal("expected error when no directory configured")
	}

	movie, err := m.BuildMovie(viewmodels.MovieRequest{ImdbID: "tt0133093", Title: "The Matrix", Folder: "/data/films"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.Abs("/data/films")
	want += "/"
	if movie.Directory != want {
		t.Errorf("Directory = %q, want %q", movie.Directory, want)
	}
}

func TestDownloadRequiresMagnet(t *testing.T) {
	origAppConfig := storage.AppConfiguration
	defer func() { storage.AppConfiguration = origAppConfig }()
	storage.AppConfiguration = &models.AppConfiguration{MoviesDir: "/tmp/movies"}

	m := NewMovies(repository.NewMovies(), repository.NewConfig())
	validReq := viewmodels.MovieRequest{ImdbID: "tt0133093", Title: "The Matrix", Year: "1999"}
	_, err := m.Download(validReq, viewmodels.TorrentRequest{})
	if err == nil {
		t.Fatal("expected error for missing magnet")
	}
	if !contains(err.Error(), "magnet") {
		t.Errorf("error message should mention magnet, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
