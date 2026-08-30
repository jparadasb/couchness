package storage

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/highercomve/couchness/models"
)

func TestTelegramUserLifecycle(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-telegram-test-")
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

	configuration, err := GetTelegramConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Enabled || len(configuration.Users) != 0 || len(configuration.Invites) != 0 {
		t.Fatalf("unexpected default configuration: %#v", configuration)
	}

	const userID int64 = 123456789
	if err := AddTelegramUser(userID, models.TelegramRoleOwner); err != nil {
		t.Fatal(err)
	}
	configuration, err = GetTelegramConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	user := configuration.Users[strconv.FormatInt(userID, 10)]
	if user == nil || user.Role != models.TelegramRoleOwner {
		t.Fatalf("owner not persisted: %#v", user)
	}

	if err := RemoveTelegramUser(userID); err != nil {
		t.Fatal(err)
	}
	configuration, err = GetTelegramConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.Users) != 0 {
		t.Fatalf("user was not removed: %#v", configuration.Users)
	}
}
