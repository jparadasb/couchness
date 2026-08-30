package app

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	"github.com/urfave/cli/v2"
)

func TestTelegramUserAddAcceptsRoleAfterID(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-telegram-cli-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)

	originalDB := storage.Db
	defer func() { storage.Db = originalDB }()
	storage.Db, err = storage.New(directory, nil)
	if err != nil {
		t.Fatal(err)
	}

	application := &cli.App{Commands: []*cli.Command{Telegram()}}
	const userID int64 = 123456789
	err = application.Run([]string{"couchness", "telegram", "users", "add", strconv.FormatInt(userID, 10), "--role", "owner"})
	if err != nil {
		t.Fatal(err)
	}

	configuration, err := storage.GetTelegramConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	user := configuration.Users[strconv.FormatInt(userID, 10)]
	if user == nil || user.Role != models.TelegramRoleOwner {
		t.Fatalf("expected owner role, got %#v", user)
	}
}
