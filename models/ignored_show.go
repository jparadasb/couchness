package models

// IgnoredShow prevents a removed show from being rediscovered during scans.
type IgnoredShow struct {
	ID           string `json:"id"`
	DirectoryKey string `json:"directory_key"`
}
