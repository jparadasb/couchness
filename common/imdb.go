package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/highercomve/couchness/storage"
)

var omdbAPIURL = "https://www.omdbapi.com/"

// OmdbResponse response from open movie database API
type OmdbResponse struct {
	Search       []OmdbResults
	TotalResults string `json:"totalResults"`
	Response     string `json:"Response"`
	Error        string `json:"Error"`
}

// OmdbResults response from open movie database API search results
type OmdbResults struct {
	Title  string `json:"Title"`
	Year   string `json:"Year"`
	ImdbID string `json:"imdbID"`
	Type   string `json:"Type"`
	Poster string `json:"Poster"`
}

// SearchShowInfo Search  show information in omdb
func SearchShowInfo(showName string, typeOf string) (*OmdbResponse, error) {
	if storage.AppConfiguration.OmdbAPIKey == "" {
		return nil, errors.New("COUCHNESS_OMDB_API_KEY is required")
	}
	if typeOf == "" {
		typeOf = "series"
	}
	query := fmt.Sprintf(
		"?apikey=%s&s=%s&type=%s",
		url.QueryEscape(storage.AppConfiguration.OmdbAPIKey),
		url.QueryEscape(showName),
		typeOf,
	)
	url := omdbAPIURL + query
	fmt.Printf("Getting Show information from IMDB... \n")
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("OMDb returned HTTP %d", res.StatusCode)
	}
	results := &OmdbResponse{}
	err = json.NewDecoder(res.Body).Decode(results)
	if err != nil {
		fmt.Printf("%s \n", err.Error())
		return nil, err
	}

	if results.Response == "False" {
		if results.Error == "" {
			results.Error = "no results"
		}
		return results, errors.New(results.Error)
	}

	return results, nil
}
