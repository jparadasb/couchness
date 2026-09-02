package storage

import (
	"strconv"
	"time"

	"github.com/highercomve/couchness/models"
)

const telegramConfID = "telegram"

// GetTelegramConfiguration loads Telegram settings, returning safe defaults when absent.
func GetTelegramConfiguration() (*models.TelegramConfiguration, error) {
	configuration := &models.TelegramConfiguration{}
	err := Db.Driver.Read(Db.Collections.Configuration, telegramConfID, configuration)
	if err != nil {
		configuration = &models.TelegramConfiguration{
			Users:   make(map[string]*models.TelegramUser),
			Invites: make(map[string]*models.TelegramInvite),
		}
		return configuration, nil
	}
	if configuration.Users == nil {
		configuration.Users = make(map[string]*models.TelegramUser)
	}
	if configuration.Invites == nil {
		configuration.Invites = make(map[string]*models.TelegramInvite)
	}
	return configuration, nil
}

// SaveTelegramConfiguration persists Telegram settings.
func SaveTelegramConfiguration(configuration *models.TelegramConfiguration) error {
	return Db.Driver.Write(Db.Collections.Configuration, telegramConfID, configuration)
}

// AddTelegramUser authorizes a Telegram account.
func AddTelegramUser(id int64, role string) error {
	configuration, err := GetTelegramConfiguration()
	if err != nil {
		return err
	}
	configuration.Users[strconv.FormatInt(id, 10)] = &models.TelegramUser{
		ID:      id,
		Role:    role,
		AddedAt: time.Now().UTC(),
	}
	return SaveTelegramConfiguration(configuration)
}

// RemoveTelegramUser revokes a Telegram account.
func RemoveTelegramUser(id int64) error {
	configuration, err := GetTelegramConfiguration()
	if err != nil {
		return err
	}
	delete(configuration.Users, strconv.FormatInt(id, 10))
	return SaveTelegramConfiguration(configuration)
}
