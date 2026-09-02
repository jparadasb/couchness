package storage

import (
	"os"
	"testing"

	"github.com/highercomve/couchness/models"
)

func TestApplyEnvironmentOverrides(t *testing.T) {
	const key = "COUCHNESS_OMDB_API_KEY"
	previous, existed := os.LookupEnv(key)
	defer func() {
		if existed {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	}()
	if err := os.Setenv(key, "runtime-key"); err != nil {
		t.Fatal(err)
	}

	configuration := &models.AppConfiguration{OmdbAPIKey: "stored-key"}
	applyEnvironmentOverrides(configuration)
	if configuration.OmdbAPIKey != "runtime-key" {
		t.Fatalf("expected runtime key override, got %q", configuration.OmdbAPIKey)
	}
}

func TestApplyEnvironmentOverridesIgnoresEmptyValues(t *testing.T) {
	const key = "COUCHNESS_OMDB_API_KEY"
	previous, existed := os.LookupEnv(key)
	defer func() {
		if existed {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	}()
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}

	configuration := &models.AppConfiguration{OmdbAPIKey: "stored-key"}
	applyEnvironmentOverrides(configuration)
	if configuration.OmdbAPIKey != "stored-key" {
		t.Fatalf("expected stored key to remain, got %q", configuration.OmdbAPIKey)
	}
}
