package common

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func TestRemoveShowKeepsMediaDirectory(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-remove-test-")
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

	mediaDirectory := filepath.Join(directory, "media", "example-show")
	if err := os.MkdirAll(mediaDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	episode, err := os.Create(filepath.Join(mediaDirectory, "Example.Show.S01E01.mkv"))
	if err != nil {
		t.Fatal(err)
	}
	if err := episode.Close(); err != nil {
		t.Fatal(err)
	}
	show := &models.Show{ID: "example-show", Title: "Example Show", Directory: mediaDirectory, Configuration: &models.ShowConf{FollowType: models.FollowTypeLatest}}
	if _, err := storage.NewShowStorage(show).Save(); err != nil {
		t.Fatal(err)
	}
	if show.ID == "" {
		t.Fatal("show ID cleared after save")
	}

	removed, err := RemoveShow(show.ID)
	if err != nil {
		t.Fatal(err)
	}
	if removed.ID != show.ID {
		t.Fatalf("unexpected removed show: %#v", removed)
	}
	if show.ID == "" {
		t.Fatal("show ID cleared after remove")
	}
	if _, err := os.Stat(mediaDirectory); err != nil {
		t.Fatalf("media directory was changed: %v", err)
	}
	rescanned, err := Scan(filepath.Join(directory, "media")+"/", "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rescanned) != 0 {
		t.Fatalf("removed show was rediscovered: %#v", rescanned)
	}
	if show.ID == "" {
		t.Fatal("show ID cleared after scan")
	}
	if _, err := getShowForRemovalTest(show.ID); err == nil {
		t.Fatal("show record still exists")
	}
	if _, err := Add(show); err != nil {
		t.Fatalf("could not explicitly add ignored show again (id=%q): %v", show.ID, err)
	}
	ignored, err := storage.GetIgnoredShowKeys()
	if err != nil {
		t.Fatal(err)
	}
	if ignored[show.ID] {
		t.Fatal("explicitly added show remains ignored")
	}
}

func getShowForRemovalTest(showID string) (*models.Show, error) {
	show := &models.Show{}
	err := storage.Db.Driver.Read(storage.Db.Collections.Shows, showID, show)
	return show, err
}
