package telegram

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func TestRemoveShowWizardKeepsMedia(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-remove-show-wizard-test-")
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
	configuration := &models.TelegramConfiguration{
		Enabled: true,
		Users: map[string]*models.TelegramUser{
			"42": {ID: 42, Role: models.TelegramRoleOwner},
		},
		Invites: make(map[string]*models.TelegramInvite),
	}
	if err := storage.SaveTelegramConfiguration(configuration); err != nil {
		t.Fatal(err)
	}

	mediaDirectory := filepath.Join(directory, "media", "example-show")
	if err := os.MkdirAll(mediaDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	show := &models.Show{ID: "example-show", Title: "Example Show", Directory: mediaDirectory, Configuration: &models.ShowConf{FollowType: models.FollowTypeLatest}}
	if _, err := storage.NewShowStorage(show).Save(); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(request.URL.Path, "/sendMessage") {
			fmt.Fprint(writer, `{"ok":true,"result":{"message_id":99}}`)
			return
		}
		fmt.Fprint(writer, `{"ok":true,"result":true}`)
	}))
	defer server.Close()

	bot := &Bot{
		client:   &apiClient{token: "test", baseURL: server.URL, httpClient: &http.Client{Timeout: time.Second}},
		sessions: make(map[int64]*addShowSession),
		removals: make(map[int64]*removeShowSession),
	}
	message := &Message{MessageID: 1, Text: "/remove_show", From: User{ID: 42}, Chat: Chat{ID: 42, Type: "private"}}
	bot.handleMessage(context.Background(), message)
	session := bot.removals[42]
	if session == nil || session.MessageID != 99 {
		t.Fatalf("removal session not started: %#v", session)
	}
	if callbackData := session.callback("select|0"); len([]byte(callbackData)) > 64 {
		t.Fatalf("callback exceeds Telegram limit: %s", strconv.Itoa(len([]byte(callbackData))))
	}

	callback := func(action string) {
		bot.handleCallback(context.Background(), &CallbackQuery{
			ID:      "callback-" + action,
			From:    User{ID: 42},
			Message: &Message{MessageID: 99, Chat: Chat{ID: 42, Type: "private"}},
			Data:    "remove|" + session.ID + "|" + action,
		})
	}
	callback("select|0")
	callback("confirm")

	stored := &models.Show{}
	if err := storage.Db.Driver.Read(storage.Db.Collections.Shows, show.ID, stored); err == nil {
		t.Fatal("show record still exists")
	}
	if _, err := os.Stat(mediaDirectory); err != nil {
		t.Fatalf("media directory was changed: %v", err)
	}
}

func TestRemoveShowPagination(t *testing.T) {
	shows := make([]*models.Show, 13)
	for index := range shows {
		shows[index] = &models.Show{ID: fmt.Sprintf("show-%d", index), Title: fmt.Sprintf("Show %d", index)}
	}
	session := &removeShowSession{ID: strings.Repeat("x", 24), Shows: shows, Page: 1}
	if session.pageCount() != 3 {
		t.Fatalf("expected 3 pages, got %d", session.pageCount())
	}
	markup := session.listKeyboard()
	if len(markup.InlineKeyboard) != 8 {
		t.Fatalf("expected 6 shows, navigation, and cancel rows; got %d", len(markup.InlineKeyboard))
	}
}
