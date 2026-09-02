package tpb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/highercomve/couchness/models"
)

const tpbFixture = `[
	{
		"id": "1",
		"name": "The.Shawshank.Redemption.1994.1080p.BluRay.x264",
		"info_hash": "E0D00667650ABA9EE05AACBBBD8B55EA8A51F534",
		"seeders": "728",
		"size": "1722274049",
		"category": "207",
		"imdb": "tt0111161"
	},
	{
		"id": "2",
		"name": "The.Shawshank.Redemption.S01E01.720p.HDTV.x264",
		"info_hash": "1111111111111111111111111111111111111111",
		"seeders": "42",
		"size": "500000000",
		"category": "208",
		"imdb": "tt0111161"
	},
	{
		"id": "3",
		"name": "Some.Other.Movie.1994.1080p.BluRay.x264",
		"info_hash": "2222222222222222222222222222222222222222",
		"seeders": "99",
		"size": "900000000",
		"category": "207",
		"imdb": "tt0000001"
	}
]`

const tpbNoResults = `[{"id":"0","name":"No results returned","info_hash":"0000000000000000000000000000000000000000","seeders":"0","size":"0","category":"0","imdb":""}]`

func newTestService(baseURL string) Service {
	return Service{
		ID:      ServiceType,
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

func newTPBServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetShowDataMovies(t *testing.T) {
	server := newTPBServer(t, tpbFixture)
	service := newTestService(server.URL)

	show := &models.Show{ID: "shawshank", Title: "The Shawshank Redemption", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "movies")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}

	if len(result.Episodes) != 1 {
		t.Fatalf("expected only the movie entry, got %d episodes", len(result.Episodes))
	}

	first := result.Episodes[0]
	if first.Seeds != 728 {
		t.Errorf("expected 728 seeds, got %d", first.Seeds)
	}
	if first.Size != 1722274049 {
		t.Errorf("expected size 1722274049, got %d", first.Size)
	}
	if !strings.HasPrefix(first.MagnetURL, "magnet:?xt=urn:btih:e0d00667650aba9ee05aacbbbd8b55ea8a51f534") {
		t.Errorf("expected lowercase info hash magnet prefix, got %q", first.MagnetURL)
	}
	if result.TorrentCount != 1 {
		t.Errorf("expected TorrentCount 1, got %d", result.TorrentCount)
	}
}

func TestGetShowDataDefaultTypeKeepsTVEntries(t *testing.T) {
	server := newTPBServer(t, tpbFixture)
	service := newTestService(server.URL)

	show := &models.Show{ID: "shawshank", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}

	// Category 208 is a TV category; the other-imdb movie is filtered out.
	if len(result.Episodes) != 1 {
		t.Fatalf("expected only the TV entry, got %d episodes", len(result.Episodes))
	}
	if result.Episodes[0].Seeds != 42 {
		t.Errorf("expected the TV entry with 42 seeds, got %d", result.Episodes[0].Seeds)
	}
}

func TestGetShowDataNoResultsPayload(t *testing.T) {
	server := newTPBServer(t, tpbNoResults)
	service := newTestService(server.URL)

	show := &models.Show{ID: "unknown", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 50, "movies")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}
	if len(result.Episodes) != 0 {
		t.Errorf("expected zero episodes for the no-results payload, got %d", len(result.Episodes))
	}
}

func TestGetShowDataEmptyExternalIDErrors(t *testing.T) {
	server := newTPBServer(t, tpbFixture)
	service := newTestService(server.URL)

	_, err := service.GetShowData(&models.Show{ID: "shawshank"}, 1, 50, "movies")
	if err == nil {
		t.Fatal("expected an error when ExternalID is empty")
	}
}

func TestGetShowDataLimitCapsResults(t *testing.T) {
	// Two matching show-category entries so the limit actually cuts something.
	body := strings.Replace(tpbFixture, `"imdb": "tt0000001"`, `"imdb": "tt0111161"`, 1)
	server := newTPBServer(t, body)
	service := newTestService(server.URL)

	show := &models.Show{ID: "shawshank", ExternalID: "tt0111161"}
	result, err := service.GetShowData(show, 1, 1, "")
	if err != nil {
		t.Fatalf("GetShowData returned an error: %v", err)
	}
	if len(result.Episodes) != 1 {
		t.Fatalf("expected the limit of 1 to cap the results, got %d episodes", len(result.Episodes))
	}
}
