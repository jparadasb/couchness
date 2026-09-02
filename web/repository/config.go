package repository

import (
	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
)

// Config exposes the application configuration.
type Config struct{}

// NewConfig creates the configuration repository.
func NewConfig() *Config { return &Config{} }

// Get returns the active configuration.
func (r *Config) Get() *models.AppConfiguration {
	return storage.AppConfiguration
}

// Save persists the configuration and applies it live.
func (r *Config) Save(c *models.AppConfiguration) error {
	if err := storage.Db.SaveAppConfiguration(c); err != nil {
		return err
	}
	*r.Get() = *c
	return nil
}

// ShowsDirs returns the configured show directories without empty entries.
func (r *Config) ShowsDirs() []string {
	configuration := r.Get()
	directories := make([]string, 0, len(configuration.ShowsDirs)+1)
	seen := map[string]bool{}
	for _, directory := range append([]string{configuration.ShowsDir}, configuration.ShowsDirs...) {
		if directory == "" || seen[directory] {
			continue
		}
		seen[directory] = true
		directories = append(directories, directory)
	}
	return directories
}

// MoviesDir returns the movies directory.
func (r *Config) MoviesDir() string {
	return r.Get().MoviesDir
}
