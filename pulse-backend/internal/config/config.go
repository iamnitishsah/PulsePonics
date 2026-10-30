package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Config struct {
	DB   DBConfig
	HTTP HTTPConfig
	MQTT MQTTConfig
	Auth AuthConfig
}

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type HTTPConfig struct {
	Port string
	// Browser origins allowed to call the API and open the WebSocket.
	CORSOrigins []string
	// True when the backend sits behind Caddy, so X-Forwarded-For is trusted.
	TrustProxy bool
}

type MQTTConfig struct {
	BrokerURL string
	Username  string
	Password  string
	ClientID  string
}

type AuthConfig struct {
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// DSN pins the session time zone and the driver's time location to UTC, so
// stored times never depend on the server's local time zone.
func (c DBConfig) DSN() string {
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, c.Port)
	cfg.DBName = c.Name
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	return cfg.FormatDSN()
}

func Load() (Config, error) {
	_ = godotenv.Load() // optional .env for running outside Docker

	var missing []string
	require := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := Config{
		DB: DBConfig{
			User:     require("DB_USER"),
			Password: require("DB_PASSWORD"),
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			Name:     require("DB_NAME"),
		},
		HTTP: HTTPConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			CORSOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		},
		MQTT: MQTTConfig{
			BrokerURL: getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
			Username:  require("MQTT_USERNAME"),
			Password:  require("MQTT_PASSWORD"),
			ClientID:  getEnv("MQTT_CLIENT_ID", "pulse-backend-"+randomSuffix()),
		},
		Auth: AuthConfig{
			JWTSecret: require("JWT_SECRET"),
		},
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: missing required env vars: %s", strings.Join(missing, ", "))
	}
	if len(cfg.Auth.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("config: JWT_SECRET must be at least 32 characters (try: openssl rand -hex 32)")
	}

	var err error
	if cfg.HTTP.TrustProxy, err = strconv.ParseBool(getEnv("TRUST_PROXY_HEADERS", "false")); err != nil {
		return Config{}, fmt.Errorf("config: TRUST_PROXY_HEADERS: %w", err)
	}
	if cfg.Auth.AccessTTL, err = time.ParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m")); err != nil {
		return Config{}, fmt.Errorf("config: ACCESS_TOKEN_TTL: %w", err)
	}
	if cfg.Auth.RefreshTTL, err = time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "168h")); err != nil {
		return Config{}, fmt.Errorf("config: REFRESH_TOKEN_TTL: %w", err)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimRight(strings.TrimSpace(part), "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// randomSuffix keeps MQTT client IDs unique, so two backend processes (e.g. a
// local run next to the container) don't keep kicking each other off the broker.
func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
