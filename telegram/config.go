package telegram

import "os"

const tokenEnvironmentVariable = "COUCHNESS_TELEGRAM_BOT_TOKEN"

// TokenFromEnvironment returns bot token without persisting it in Couchness storage.
func TokenFromEnvironment() string {
	return os.Getenv(tokenEnvironmentVariable)
}
