package telegram

import (
	"context"
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
