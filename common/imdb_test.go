package common

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func TestSearchShowInfo(t *testing.T) {
	originalURL := omdbAPIURL
	originalConfiguration := storage.AppConfiguration
	defer func() {
		omdbAPIURL = originalURL
		storage.AppConfiguration = originalConfiguration
	}()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("apikey") != "test-key" || request.URL.Query().Get("type") != "series" {
			t.Errorf("unexpected query: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		if strings.Contains(request.URL.Query().Get("s"), "missing") {
			fmt.Fprint(writer, `{"Response":"False","Error":"Movie not found!"}`)
			return
		}
		fmt.Fprint(writer, `{"Search":[{"Title":"Example","Year":"2026","imdbID":"tt1234567","Type":"series"}],"totalResults":"1","Response":"True"}`)
	}))
	defer server.Close()
	omdbAPIURL = server.URL + "/"
	storage.AppConfiguration = &models.AppConfiguration{OmdbAPIKey: "test-key"}

	results, err := SearchShowInfo("Example", "series")
	if err != nil {
		t.Fatal(err)
	}
	if len(results.Search) != 1 || results.Search[0].ImdbID != "tt1234567" {
		t.Fatalf("unexpected results: %#v", results)
	}

	results, err = SearchShowInfo("missing", "series")
	if err == nil || results == nil || len(results.Search) != 0 {
		t.Fatalf("expected a useful no-results error, got %#v, %v", results, err)
	}
}

func TestSearchShowInfoRequiresAPIKey(t *testing.T) {
	originalConfiguration := storage.AppConfiguration
	defer func() { storage.AppConfiguration = originalConfiguration }()
	storage.AppConfiguration = &models.AppConfiguration{}

	if _, err := SearchShowInfo("Example", "series"); err == nil {
		t.Fatal("expected missing API key error")
	}
}
