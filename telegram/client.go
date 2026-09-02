package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiBaseURL = "https://api.telegram.org"

type apiClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type botIdentity struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func newAPIClient(token string) *apiClient {
	return &apiClient{
		token:   token,
		baseURL: apiBaseURL,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (client *apiClient) call(ctx context.Context, method string, values url.Values, result interface{}) error {
	endpoint := fmt.Sprintf("%s/bot%s/%s", client.baseURL, client.token, method)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	payload := &apiResponse{}
	if err := json.NewDecoder(response.Body).Decode(payload); err != nil {
		return err
	}
	if !payload.OK {
		return fmt.Errorf("Telegram API error: %s", payload.Description)
	}
	if result != nil {
		return json.Unmarshal(payload.Result, result)
	}
	return nil
}

func (client *apiClient) getMe(ctx context.Context) (*botIdentity, error) {
	identity := &botIdentity{}
	err := client.call(ctx, "getMe", url.Values{}, identity)
	return identity, err
}

func (client *apiClient) getUpdates(ctx context.Context, offset int64) ([]Update, error) {
	values := url.Values{
		"offset":          {strconv.FormatInt(offset, 10)},
		"timeout":         {"30"},
		"allowed_updates": {`["message","callback_query"]`},
	}
	updates := []Update{}
	err := client.call(ctx, "getUpdates", values, &updates)
	return updates, err
}

func (client *apiClient) deleteWebhook(ctx context.Context) error {
	return client.call(ctx, "deleteWebhook", url.Values{}, nil)
}

func (client *apiClient) setCommands(ctx context.Context, commands []BotCommand) error {
	encoded, err := json.Marshal(commands)
	if err != nil {
		return err
	}
	return client.call(ctx, "setMyCommands", url.Values{"commands": {string(encoded)}}, nil)
}

func (client *apiClient) sendChatAction(ctx context.Context, chatID int64, action string) error {
	values := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"action":  {action},
	}
	return client.call(ctx, "sendChatAction", values, nil)
}

func (client *apiClient) sendMessage(ctx context.Context, chatID int64, message string) error {
	runes := []rune(message)
	for len(runes) > 0 {
		length := len(runes)
		if length > 4000 {
			length = 4000
		}
		part := string(runes[:length])
		runes = runes[length:]
		values := url.Values{
			"chat_id": {strconv.FormatInt(chatID, 10)},
			"text":    {part},
		}
		if err := client.call(ctx, "sendMessage", values, nil); err != nil {
			return err
		}
	}
	return nil
}

func (client *apiClient) sendMessageWithKeyboard(ctx context.Context, chatID int64, message string, keyboard InlineKeyboardMarkup) (*Message, error) {
	normalizeKeyboard(&keyboard)
	markup, err := json.Marshal(keyboard)
	if err != nil {
		return nil, err
	}
	values := url.Values{
		"chat_id":      {strconv.FormatInt(chatID, 10)},
		"text":         {message},
		"reply_markup": {string(markup)},
	}
	result := &Message{}
	if err := client.call(ctx, "sendMessage", values, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (client *apiClient) editMessage(ctx context.Context, chatID, messageID int64, message string, keyboard InlineKeyboardMarkup) error {
	normalizeKeyboard(&keyboard)
	markup, err := json.Marshal(keyboard)
	if err != nil {
		return err
	}
	values := url.Values{
		"chat_id":      {strconv.FormatInt(chatID, 10)},
		"message_id":   {strconv.FormatInt(messageID, 10)},
		"text":         {message},
		"reply_markup": {string(markup)},
	}
	return client.call(ctx, "editMessageText", values, nil)
}

func normalizeKeyboard(keyboard *InlineKeyboardMarkup) {
	if keyboard.InlineKeyboard == nil {
		keyboard.InlineKeyboard = make([][]InlineKeyboardButton, 0)
	}
}

func (client *apiClient) answerCallback(ctx context.Context, callbackID, message string) error {
	values := url.Values{"callback_query_id": {callbackID}}
	if message != "" {
		values.Set("text", message)
	}
	return client.call(ctx, "answerCallbackQuery", values, nil)
}
