package config

import "testing"

func TestTMDBAPIKeyUsesEnvironment(t *testing.T) {
	t.Setenv("TMDB_API_KEY", "env-key")

	if got := TMDBAPIKey(); got != "env-key" {
		t.Fatalf("TMDBAPIKey = %q, want env-key", got)
	}
}

func TestTMDBAPIKeyFallsBackToPool(t *testing.T) {
	t.Setenv("TMDB_API_KEY", "")

	key := TMDBAPIKey()
	if key == "" {
		t.Fatalf("TMDBAPIKey returned empty key")
	}
	if !IsTMDBPoolKey(key) {
		t.Fatalf("TMDBAPIKey = %q, want pooled key", key)
	}
}
