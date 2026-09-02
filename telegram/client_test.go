package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendMessageSplitsLongUnicodeText(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body), "chat_id=42") {
			t.Errorf("missing chat ID in request: %s", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"ok":true,"result":{}}`)
	}))
	defer server.Close()

	client := &apiClient{
		token:      "test-token",
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: time.Second},
	}
	message := strings.Repeat("á", 4001)
	if err := client.sendMessage(context.Background(), 42, message); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
}

func TestSendMessageWithKeyboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Error(err)
		}
		markup := &InlineKeyboardMarkup{}
		if err := json.Unmarshal([]byte(request.Form.Get("reply_markup")), markup); err != nil {
			t.Error(err)
		}
		if len(markup.InlineKeyboard) != 1 || markup.InlineKeyboard[0][0].CallbackData != "choose" {
			t.Fatalf("unexpected keyboard: %#v", markup)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"ok":true,"result":{"message_id":99}}`)
	}))
	defer server.Close()

	client := &apiClient{token: "test-token", baseURL: server.URL, httpClient: &http.Client{Timeout: time.Second}}
	message, err := client.sendMessageWithKeyboard(context.Background(), 42, "Choose", keyboard(row(button("Choose", "choose"))))
	if err != nil {
		t.Fatal(err)
	}
	if message.MessageID != 99 {
		t.Fatalf("expected message ID 99, got %d", message.MessageID)
	}
}
