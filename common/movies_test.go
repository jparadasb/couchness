package common

import (
	"errors"
	"testing"

	"github.com/highercomve/couchness/models"
)

// fakeService is a configurable models.FollowService used to test the movie flows offline.
type fakeService struct {
	episodes models.Episodes
	err      error
}

func (f *fakeService) GetID() string { return "fake" }

func (f *fakeService) GetURL() string { return "http://fake.example" }

func (f *fakeService) ShowURL(showID string, page, limit int) string {
	return "http://fake.example/" + showID
}

func (f *fakeService) GetShowData(show *models.Show, page, limit int, typeOf string) (*models.Show, error) {
	if f.err != nil {
		return nil, f.err
	}
	show.Episodes = append(show.Episodes, f.episodes...)
	return show, nil
}

func registerFake(t *testing.T, fake *fakeService) {
	t.Helper()
	FollowServices["fake"] = fake
	t.Cleanup(func() { delete(FollowServices, "fake") })
}

func newTestMovie(resolution string) *models.Movie {
	movie := &models.Movie{}
	movie.ID = "shawshank"
	movie.Show.Title = "The Shawshank Redemption"
	movie.ExternalID = "tt0111161"
	movie.Directory = "/tmp/couchness-test-movies/shawshank"
	movie.Configuration = &models.ShowConf{
		Services:   []string{"fake"},
		Resolution: resolution,
	}
	return movie
}

func TestSearchMovieTorrentsSortedBySeeds(t *testing.T) {
	registerFake(t, &fakeService{episodes: models.Episodes{
		{Title: "low", Seeds: 5, Resolution: "1080p", MagnetURL: "magnet:?xt=urn:btih:low"},
		{Title: "high", Seeds: 20, Resolution: "1080p", MagnetURL: "magnet:?xt=urn:btih:high"},
		{Title: "mid", Seeds: 12, Resolution: "1080p", MagnetURL: "magnet:?xt=urn:btih:mid"},
	}})

	torrents, err := SearchMovieTorrents(newTestMovie(""))
	if err != nil {
		t.Fatalf("SearchMovieTorrents returned an error: %v", err)
	}

	want := []string{"high", "mid", "low"}
	for i, title := range want {
		if torrents[i].Title != title {
			t.Errorf("expected position %d to be %s, got %s", i, title, torrents[i].Title)
		}
	}
}

func TestSearchMovieTorrentsFiltersByResolution(t *testing.T) {
	registerFake(t, &fakeService{episodes: models.Episodes{
		{Title: "full-hd", Seeds: 5, Resolution: "1080p", MagnetURL: "magnet:?xt=urn:btih:1080"},
		{Title: "hd", Seeds: 50, Resolution: "720p", MagnetURL: "magnet:?xt=urn:btih:720"},
	}})

	torrents, err := SearchMovieTorrents(newTestMovie("1080p"))
	if err != nil {
		t.Fatalf("SearchMovieTorrents returned an error: %v", err)
	}

	if len(torrents) != 1 {
		t.Fatalf("expected only the 1080p torrent, got %d", len(torrents))
	}
	if torrents[0].Title != "full-hd" {
		t.Errorf("expected the 1080p torrent, got %s", torrents[0].Title)
	}
}

func TestSearchMovieTorrentsNoTorrentsErrors(t *testing.T) {
	registerFake(t, &fakeService{episodes: models.Episodes{}})

	_, err := SearchMovieTorrents(newTestMovie(""))
	if err == nil {
		t.Fatal("expected an error when the service returns no torrents")
	}
}

func TestSearchMovieTorrentsServiceErrorPropagates(t *testing.T) {
	registerFake(t, &fakeService{err: errors.New("service down")})

	_, err := SearchMovieTorrents(newTestMovie(""))
	if err == nil {
		t.Fatal("expected an error when the service fails")
	}
}

func TestSearchMovieTorrentsEmptyExternalIDErrors(t *testing.T) {
	registerFake(t, &fakeService{episodes: models.Episodes{
		{Title: "any", Seeds: 1, MagnetURL: "magnet:?xt=urn:btih:any"},
	}})

	movie := newTestMovie("")
	movie.ExternalID = ""

	_, err := SearchMovieTorrents(movie)
	if err == nil {
		t.Fatal("expected an error when ExternalID is empty")
	}
}

func TestDownloadMovieRequiresTorrent(t *testing.T) {
	if err := DownloadMovie(newTestMovie(""), nil); err == nil {
		t.Error("expected an error for a nil torrent")
	}

	noMagnet := &models.TorrentInfo{Title: "no magnet", Seeds: 1}
	if err := DownloadMovie(newTestMovie(""), noMagnet); err == nil {
		t.Error("expected an error for a torrent without a magnet link")
	}
}
