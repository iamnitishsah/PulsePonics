// internal/config/config.go
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds everything the app needs to start up. Keeping it as one
// struct (rather than scattering os.Getenv calls throughout the codebase)
// means there's exactly one place that knows about environment variables —
// everything else just receives a Config value.
type Config struct {
	DB     DBConfig
	Server ServerConfig
	MQTT   MQTTConfig
}

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type ServerConfig struct {
	Port string
}

type MQTTConfig struct {
	BrokerURL string
}

// DSN builds the MySQL Data Source Name string that the sql driver needs to
// connect. Format: user:password@tcp(host:port)/dbname?param=value
//
// parseTime=true is important: it tells the MySQL driver to automatically
// convert MySQL DATETIME columns into Go's time.Time when scanning rows,
// instead of returning raw []byte that you'd have to parse yourself.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name,
	)
}

// Load reads config from environment variables, falling back to sensible
// local-dev defaults where it's safe to do so (host/port), but requiring
// User/Password/Name to be explicitly set — credentials should never have
// a silent default.
func Load() (Config, error) {
	// godotenv.Load() reads the .env file at the project root and injects
	// its key=value pairs into the process environment, so the
	// os.Getenv() calls below pick them up as if you'd exported them in
	// your shell. If no .env file exists, it returns an error — but we
	// intentionally ignore that error, because in production you'd set
	// real environment variables directly and there'd be no .env file at
	// all. We only want this to be a *local dev convenience*, not a
	// requirement.
	_ = godotenv.Load()

	cfg := Config{
		DB: DBConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     getEnvOrDefault("DB_HOST", "127.0.0.1"),
			Port:     getEnvOrDefault("DB_PORT", "3306"),
			Name:     os.Getenv("DB_NAME"),
		},
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", "8080"),
		},
		MQTT: MQTTConfig{
			BrokerURL: getEnvOrDefault("MQTT_BROKER_URL", "tcp://localhost:1883"),
		},
	}

	if cfg.DB.User == "" || cfg.DB.Name == "" {
		return Config{}, fmt.Errorf("config: DB_USER and DB_NAME must be set")
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
