package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

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

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name,
	)
}

func Load() (Config, error) {
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

	if cfg.DB.Password == "" {
		return Config{}, fmt.Errorf("config: DB_PASSWORD must be set")
	}

	return cfg, nil
}

func getEnvOrDefault(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
