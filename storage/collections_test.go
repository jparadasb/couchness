package storage

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestMissingMediaCollectionsAreEmpty(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-empty-collections-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)

	originalDB := Db
	defer func() { Db = originalDB }()
	Db, err = New(directory, nil)
	if err != nil {
		t.Fatal(err)
	}

	shows, err := GetAllShows()
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 0 {
		t.Fatalf("expected no shows, got %d", len(shows))
	}

	movies, err := GetAllMovies()
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 0 {
		t.Fatalf("expected no movies, got %d", len(movies))
	}
}
