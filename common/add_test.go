package common

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func TestAddCreatesShowDirectory(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-add-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)

	originalDB := storage.Db
	defer func() { storage.Db = originalDB }()
	storage.Db, err = storage.New(filepath.Join(directory, "database"), nil)
	if err != nil {
		t.Fatal(err)
	}

	showDirectory := filepath.Join(directory, "media", "example-show")
	show := &models.Show{
		ID:         "example-show",
		Title:      "Example Show",
		ExternalID: "tt1234567",
		Directory:  showDirectory,
		Configuration: &models.ShowConf{
			FollowType: models.FollowTypeLatest,
		},
	}
	if _, err := Add(show); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(showDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", showDirectory)
	}
}
