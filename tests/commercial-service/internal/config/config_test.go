package config_test

import (
	"os"
	"testing"

	"metarang/commercial-service/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_DefaultsAndOverrides(t *testing.T) {
	keys := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_DATABASE", "GRPC_PORT", "HTTP_PORT", "LOCALE"}
	prev := map[string]string{}
	for _, k := range keys {
		prev[k] = os.Getenv(k)
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range prev {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	})

	cfg := config.LoadConfig()
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "3306", cfg.Database.Port)
	assert.Equal(t, "root", cfg.Database.User)
	assert.Equal(t, "", cfg.Database.Password)
	assert.Equal(t, "metarang", cfg.Database.Database)
	assert.Equal(t, "50052", cfg.Server.GRPCPort)
	assert.Equal(t, "8080", cfg.Server.HTTPPort)
	assert.Equal(t, "en", cfg.Server.Locale)

	_ = os.Setenv("DB_HOST", "db.example")
	_ = os.Setenv("DB_PORT", "3307")
	_ = os.Setenv("DB_USER", "app")
	_ = os.Setenv("DB_PASSWORD", "secret")
	_ = os.Setenv("DB_DATABASE", "commercial")
	_ = os.Setenv("GRPC_PORT", "50099")
	_ = os.Setenv("HTTP_PORT", "9090")
	_ = os.Setenv("LOCALE", "fa")

	cfg = config.LoadConfig()
	assert.Equal(t, "db.example", cfg.Database.Host)
	assert.Equal(t, "3307", cfg.Database.Port)
	assert.Equal(t, "app", cfg.Database.User)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "commercial", cfg.Database.Database)
	assert.Equal(t, "50099", cfg.Server.GRPCPort)
	assert.Equal(t, "9090", cfg.Server.HTTPPort)
	assert.Equal(t, "fa", cfg.Server.Locale)
}
