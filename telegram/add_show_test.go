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

	"github.com/highercomve/couchness/common"
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

func TestAddShowResultKeyboardFitsTelegramLimits(t *testing.T) {
	session := &addShowSession{
		ID: strings.Repeat("x", 24),
		Results: []common.OmdbResults{
			{Title: strings.Repeat("Long title ", 20), Year: "2026"},
		},
	}
	markup := session.resultsKeyboard()
	if len(markup.InlineKeyboard) != 2 {
		t.Fatalf("expected result and cancel rows, got %d", len(markup.InlineKeyboard))
	}
	result := markup.InlineKeyboard[0][0]
	if len([]rune(result.Text)) > 60 {
		t.Fatalf("button label too long: %d", len([]rune(result.Text)))
	}
	if len([]byte(result.CallbackData)) > 64 {
		t.Fatalf("callback data too long: %d", len([]byte(result.CallbackData)))
	}
}

func TestTruncateButtonTextPreservesUnicode(t *testing.T) {
	value := strings.Repeat("á", 70)
	result := truncateButtonText(value, 60)
	if len([]rune(result)) != 60 || !strings.HasSuffix(result, "…") {
		t.Fatalf("unexpected truncated value: %q", result)
	}
}

func TestAddShowWizardPersistsSelection(t *testing.T) {
	directory, err := ioutil.TempDir("", "couchness-add-show-wizard-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)

	originalDB := storage.Db
	originalConfiguration := storage.AppConfiguration
	defer func() {
		storage.Db = originalDB
		storage.AppConfiguration = originalConfiguration
	}()
	storage.Db, err = storage.New(filepath.Join(directory, "database"), nil)
	if err != nil {
		t.Fatal(err)
	}
	storage.AppConfiguration = &models.AppConfiguration{ShowsDir: filepath.Join(directory, "shows") + "/"}
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
		searchShows: func(title, typeOf string) (*common.OmdbResponse, error) {
			return &common.OmdbResponse{Search: []common.OmdbResults{{Title: "Example Show", Year: "2026", ImdbID: "tt1234567"}}}, nil
		},
	}
	message := &Message{MessageID: 1, Text: "/add_show Example", From: User{ID: 42}, Chat: Chat{ID: 42, Type: "private"}}
	bot.handleMessage(context.Background(), message)
	session := bot.sessions[42]
	if session == nil || session.MessageID != 99 {
		t.Fatalf("wizard session not started: %#v", session)
	}

	callback := func(action string) {
		bot.handleCallback(context.Background(), &CallbackQuery{
			ID:      "callback-" + action,
			From:    User{ID: 42},
			Message: &Message{MessageID: 99, Chat: Chat{ID: 42, Type: "private"}},
			Data:    "add|" + session.ID + "|" + action,
		})
	}
	callback("select|0")
	callback("follow_latest")
	callback("resolution_1080p")
	callback("confirm")

	value, err := common.GetShow("example-show", false)
	if err != nil {
		t.Fatal(err)
	}
	show := value.(*models.Show)
	if show.ExternalID != "tt1234567" || show.Configuration.Resolution != "1080p" {
		t.Fatalf("unexpected saved show: %#v", show)
	}
	if _, err := os.Stat(filepath.Join(directory, "shows", "example-show")); err != nil {
		t.Fatal(err)
	}
	if callbackData := session.callback("resolution_1080p"); len([]byte(callbackData)) > 64 {
		t.Fatalf("callback exceeds Telegram limit: %s", strconv.Itoa(len([]byte(callbackData))))
	}
}
