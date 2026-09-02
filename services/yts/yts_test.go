package yts

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/highercomve/couchness/models"
)

const ytsFixture = `{
	"status": "ok",
	"data": {
		"movie_count": 1,
		"movies": [
			{
				"title": "The Shawshank Redemption",
				"title_long": "The Shawshank Redemption (1994)",
				"year": 1994,
				"imdb_code": "tt0111161",
				"torrents": [
					{"hash": "ABC123", "quality": "1080p", "type": "bluray", "seeds": 10, "peers": 2, "size_bytes": 123456},
					{"hash": "DEF456", "quality": "720p", "type": "web", "seeds": 3, "peers": 1, "size_bytes": 1000}
				]
			}
		]
	}
}`

func newTestService(baseURL string, mirrors []string) Service {
	return Service{
		ID:      ServiceType,
		BaseURL: baseURL,
		Mirrors: mirrors,
		Client:  &http.Client{},
	}
}

func newFixtureServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetShowDataMovies(t *testing.T) {
	server := newFixtureServer(t, ytsFixture, http.StatusOK)
	service := newTestService(server.URL, nil)

	show := &models.Show{ID: "shawshank", Title: "The Shawshank Redemption", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "movies")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}

	if len(result.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(result.Episodes))
	}

	first, second := result.Episodes[0], result.Episodes[1]
	if first.Seeds != 10 || second.Seeds != 3 {
		t.Errorf("expected seeds 10 and 3, got %d and %d", first.Seeds, second.Seeds)
	}
	if first.Size != 123456 || second.Size != 1000 {
		t.Errorf("expected sizes 123456 and 1000, got %d and %d", first.Size, second.Size)
	}
	if first.Resolution != "1080p" || second.Resolution != "720p" {
		t.Errorf("expected resolutions 1080p and 720p, got %q and %q", first.Resolution, second.Resolution)
	}
	if !strings.HasPrefix(first.MagnetURL, "magnet:?xt=urn:btih:ABC123") {
		t.Errorf("expected magnet to start with the ABC123 hash, got %q", first.MagnetURL)
	}
	if result.TorrentCount != 2 {
		t.Errorf("expected TorrentCount 2, got %d", result.TorrentCount)
	}
}

func TestGetShowDataNonMovieTypeReturnsNoEpisodes(t *testing.T) {
	server := newFixtureServer(t, ytsFixture, http.StatusOK)
	service := newTestService(server.URL, nil)

	show := &models.Show{ID: "shawshank", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}
	if len(result.Episodes) != 0 {
		t.Errorf("expected no episodes for a non-movie type, got %d", len(result.Episodes))
	}
}

func TestGetShowDataEmptyExternalIDErrors(t *testing.T) {
	server := newFixtureServer(t, ytsFixture, http.StatusOK)
	service := newTestService(server.URL, nil)

	_, err := service.GetShowData(&models.Show{ID: "shawshank"}, 1, 50, "movies")
	if err == nil {
		t.Fatal("expected an error when ExternalID is empty")
	}
}

func TestGetShowDataSkipsOtherMovies(t *testing.T) {
	body := strings.Replace(ytsFixture, "tt0111161", "tt9999999", 1)
	server := newFixtureServer(t, body, http.StatusOK)
	service := newTestService(server.URL, nil)

	show := &models.Show{ID: "shawshank", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "movies")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}
	if len(result.Episodes) != 0 {
		t.Errorf("expected a different imdb_code movie to be skipped, got %d episodes", len(result.Episodes))
	}
}

func TestGetShowDataMirrorFallback(t *testing.T) {
	good := newFixtureServer(t, ytsFixture, http.StatusOK)
	bad := newFixtureServer(t, "internal error", http.StatusInternalServerError)

	service := newTestService(bad.URL, []string{good.URL})

	show := &models.Show{ID: "shawshank", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "movies")
	if err != nil {
		t.Fatalf("expected the mirror fallback to succeed, got error: %v", err)
	}
	if len(result.Episodes) != 2 {
		t.Errorf("expected 2 episodes from the mirror, got %d", len(result.Episodes))
	}
}
